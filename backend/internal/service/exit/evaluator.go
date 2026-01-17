package exit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

const (
	evaluationInterval = 3 * time.Second  // 1~5초 권장
	maxRetries         = 3                // 최대 재평가 횟수
	freshnessThreshold = 25 * time.Second // 가격 신선도 임계값 (REST Tier1=10초 × 2 + 5초 버퍼)
)

// evaluationLoop runs the main exit evaluation loop (1~5초 주기)
func (s *Service) evaluationLoop() {
	ticker := time.NewTicker(evaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Evaluate all positions
			if err := s.evaluateAllPositions(s.ctx); err != nil {
				log.Error().Err(err).Msg("Exit evaluation failed")
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// evaluateAllPositions evaluates all OPEN and CLOSING positions for exit triggers
func (s *Service) evaluateAllPositions(ctx context.Context) error {
	// 1. Check Control Gate
	control, err := s.controlRepo.GetControl(ctx)
	if err != nil {
		return fmt.Errorf("get control: %w", err)
	}

	log.Debug().Str("mode", control.Mode).Msg("Exit control mode")

	// 2. Load OPEN and CLOSING positions (모든 계정)
	// NOTE: CLOSING 포지션도 포함하여 부분 청산 후 남은 수량도 계속 평가
	positions, err := s.posRepo.GetAllOpenPositions(ctx)
	if err != nil {
		return fmt.Errorf("get open positions: %w", err)
	}

	log.Debug().Int("count", len(positions)).Msg("Evaluating positions")

	// 3. Evaluate each position
	for _, pos := range positions {
		// Skip disabled or manual-only positions
		if pos.ExitMode == exit.ExitModeDisabled || pos.ExitMode == exit.ExitModeManualOnly {
			log.Debug().Str("symbol", pos.Symbol).Str("exit_mode", pos.ExitMode).
				Msg("Position exit auto-evaluation disabled, skipping")
			continue
		}

		// Evaluate position with retry
		if err := s.evaluatePositionWithRetry(ctx, pos, control.Mode, 0); err != nil {
			// Skip logging for expected business logic conditions
			if err == exit.ErrNoAvailableQty {
				// Normal case: all quantity is locked, skip silently
				continue
			}

			if err == exit.ErrStalePrice {
				// Normal case: price too old, skip evaluation (already logged as Debug in evaluatePosition)
				continue
			}

			log.Error().
				Err(err).
				Str("symbol", pos.Symbol).
				Str("position_id", pos.PositionID.String()).
				Msg("Position evaluation failed")
		}
	}

	return nil
}

// evaluatePositionWithRetry evaluates a position with retry on version conflict
func (s *Service) evaluatePositionWithRetry(ctx context.Context, pos *exit.Position, controlMode string, attempt int) error {
	if attempt >= maxRetries {
		return exit.ErrMaxRetriesExceeded
	}

	err := s.evaluatePosition(ctx, pos, controlMode)
	if err == exit.ErrPositionChanged {
		// Version conflict, retry with updated position
		log.Warn().
			Str("symbol", pos.Symbol).
			Int("attempt", attempt+1).
			Msg("Position changed, retrying evaluation")

		// Reload position
		updatedPos, err := s.posRepo.GetPosition(ctx, pos.PositionID)
		if err != nil {
			return err
		}

		return s.evaluatePositionWithRetry(ctx, updatedPos, controlMode, attempt+1)
	}

	return err
}

// evaluatePosition evaluates a single position for exit triggers
func (s *Service) evaluatePosition(ctx context.Context, pos *exit.Position, controlMode string) error {
	// 0. Validate position data (data integrity check)
	if pos.Symbol == "" {
		log.Error().
			Str("position_id", pos.PositionID.String()).
			Msg("Position has empty symbol (data integrity issue)")
		return fmt.Errorf("position has empty symbol")
	}

	// 1. Create position snapshot (v10 방어: 평가 시작 시 snapshot)
	snapshot := PositionSnapshot{
		PositionID:  pos.PositionID,
		Symbol:      pos.Symbol,
		Qty:         pos.Qty,
		OriginalQty: pos.OriginalQty,
		AvgPrice:    pos.AvgPrice,
		EntryTS:     pos.EntryTS,
		Version:     pos.Version,
		Phase:       "", // Will be set after state retrieval
	}

	// 2. Get position state (FSM)
	state, err := s.stateRepo.GetState(ctx, pos.PositionID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	// Set snapshot phase after state retrieval
	snapshot.Phase = state.Phase

	// 2.5. Check for 평단가 변경 (추가 매수 vs 부분체결/정정 구분)
	if state.LastAvgPrice != nil {
		diff := pos.AvgPrice.Sub(*state.LastAvgPrice).Abs()
		minThreshold := state.LastAvgPrice.Mul(decimal.NewFromFloat(0.005)) // 0.5%
		additionalBuyThreshold := state.LastAvgPrice.Mul(decimal.NewFromFloat(0.02)) // 2%

		if diff.GreaterThan(minThreshold) {
			// 0.5% 이상 변경 감지
			if diff.GreaterThan(additionalBuyThreshold) {
				// 2% 이상 → 추가매수 → OPEN 리셋
				log.Warn().
					Str("symbol", pos.Symbol).
					Str("old_avg_price", state.LastAvgPrice.String()).
					Str("new_avg_price", pos.AvgPrice.String()).
					Str("diff_pct", diff.Div(*state.LastAvgPrice).Mul(decimal.NewFromInt(100)).StringFixed(2)).
					Str("old_phase", state.Phase).
					Msg("추가매수 감지 (>2%) → Exit State OPEN 리셋")

				err = s.stateRepo.ResetStateToOpen(ctx, pos.PositionID, pos.AvgPrice)
				if err != nil {
					return fmt.Errorf("reset state to open: %w", err)
				}

				// State 재로드
				state, err = s.stateRepo.GetState(ctx, pos.PositionID)
				if err != nil {
					return fmt.Errorf("get state after reset: %w", err)
				}
				// Update snapshot phase after reset
				snapshot.Phase = state.Phase
			} else {
				// 0.5~2% → 부분체결/정정 → State 유지, LastAvgPrice만 업데이트
				log.Debug().
					Str("symbol", pos.Symbol).
					Str("old_avg_price", state.LastAvgPrice.String()).
					Str("new_avg_price", pos.AvgPrice.String()).
					Str("diff_pct", diff.Div(*state.LastAvgPrice).Mul(decimal.NewFromInt(100)).StringFixed(2)).
					Msg("부분체결/정정 감지 (0.5~2%) → State 유지, LastAvgPrice 업데이트")

				err = s.stateRepo.UpdateLastAvgPrice(ctx, pos.PositionID, pos.AvgPrice)
				if err != nil {
					log.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to update last_avg_price")
				}
				state.LastAvgPrice = &pos.AvgPrice
			}
		}
	} else if state.LastAvgPrice == nil {
		// 처음 평가하는 경우 LastAvgPrice 설정
		state.LastAvgPrice = &pos.AvgPrice
		err = s.stateRepo.ResetStateToOpen(ctx, pos.PositionID, pos.AvgPrice)
		if err != nil {
			log.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to initialize last_avg_price")
		}
	}

	// 3. Get current price (v10 방어: Freshness 검증)
	bestPrice, err := s.priceSync.GetBestPrice(ctx, pos.Symbol)
	if err != nil {
		return fmt.Errorf("get best price: %w", err)
	}

	// 4. Check price freshness (v10 방어: 타임스탬프 검증)
	// Check if price is stale (from BestPrice)
	if bestPrice.IsStale {
		log.Debug().
			Str("symbol", pos.Symbol).
			Msg("Price is stale, skipping evaluation (Fail-Closed)")
		return exit.ErrStalePrice
	}

	// Check timestamp age
	age := time.Since(bestPrice.BestTS)
	if age > freshnessThreshold {
		log.Debug().
			Str("symbol", pos.Symbol).
			Float64("age_seconds", age.Seconds()).
			Msg("Price too old, skipping evaluation")
		return exit.ErrStalePrice
	}

	// 5. Resolve exit profile (Position > Symbol > Default)
	profile := s.resolveExitProfile(ctx, pos)
	if profile == nil {
		log.Warn().Str("symbol", pos.Symbol).Msg("No exit profile, using default")
		profile = s.defaultProfile
	}

	// 6. Update HWM if in TRAILING_ACTIVE phase
	if state.Phase == exit.PhaseTrailingActive {
		currentPriceInt := bestPrice.BestPrice
		if bestPrice.BidPrice != nil {
			currentPriceInt = *bestPrice.BidPrice
		}
		currentPrice := decimal.NewFromInt(currentPriceInt)

		// Update HWM if current price is higher
		if state.HWMPrice == nil || currentPrice.GreaterThan(*state.HWMPrice) {
			err = s.stateRepo.UpdateHWM(ctx, pos.PositionID, currentPrice)
			if err != nil {
				log.Error().Err(err).Str("symbol", pos.Symbol).Msg("Failed to update HWM")
			} else {
				log.Debug().
					Str("symbol", pos.Symbol).
					Str("hwm_price", currentPrice.String()).
					Msg("HWM updated")
			}
		}
	}

	// 7. Check existing active intents for this position
	existingIntents, err := s.getActiveIntents(ctx, pos.PositionID)
	if err != nil {
		log.Warn().Err(err).Str("symbol", pos.Symbol).Msg("Failed to get existing intents (continuing anyway)")
		existingIntents = nil // continue without existing intent check
	}

	// 8. Evaluate triggers (우선순위 순서)
	trigger := s.evaluateTriggers(ctx, snapshot, state, bestPrice, profile, controlMode)

	// 8.5. Check if new trigger is more severe than existing intents
	if trigger != nil && existingIntents != nil && len(existingIntents) > 0 {
		if !s.isMoreSevere(trigger.ReasonCode, existingIntents) {
			log.Debug().
				Str("symbol", snapshot.Symbol).
				Str("new_trigger", trigger.ReasonCode).
				Msg("New trigger is not more severe than existing intents, skipping")
			return nil // Don't create duplicate/less severe intent
		}
	}

	// 9. Record exit signal for debugging/backtest (best-effort, non-blocking)
	if trigger != nil {
		currentPriceInt := bestPrice.BestPrice
		if bestPrice.BidPrice != nil {
			currentPriceInt = *bestPrice.BidPrice
		}
		currentPrice := decimal.NewFromInt(currentPriceInt)

		signal := &exit.ExitSignal{
			SignalID:    uuid.New(),
			PositionID:  snapshot.PositionID,
			RuleName:    trigger.ReasonCode,
			IsTriggered: true,
			Reason:      fmt.Sprintf("Trigger hit: %s", trigger.ReasonCode),
			Price:       currentPrice,
			EvaluatedTS: time.Now(),
		}

		// Best-effort signal recording (non-blocking)
		if err := s.signalRepo.InsertSignal(ctx, signal); err != nil {
			log.Warn().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to record exit signal (non-fatal)")
		}
	}

	if trigger == nil {
		// No trigger hit
		return nil
	}

	// 10. Create intent (v10 방어: Intent 생성 직전 DB 재확인)
	return s.createIntentWithVersionCheck(ctx, snapshot, trigger)
}

// getActiveIntents retrieves active intents for a position
func (s *Service) getActiveIntents(ctx context.Context, positionID uuid.UUID) ([]*exit.OrderIntent, error) {
	// Get active intents for this position directly from DB
	// Uses idx_order_intents_position_status index for performance
	return s.intentRepo.GetActiveIntentsByPosition(ctx, positionID)
}

// isMoreSevere checks if the new trigger is more severe than existing intents
// Severity order: SL2 > SL1 > TP3 > TP2 > TP1 > TRAIL
func (s *Service) isMoreSevere(newReasonCode string, existingIntents []*exit.OrderIntent) bool {
	newSeverity := getTriggerSeverity(newReasonCode)

	// Check if any existing intent is more or equally severe
	for _, intent := range existingIntents {
		existingSeverity := getTriggerSeverity(intent.ReasonCode)
		if existingSeverity >= newSeverity {
			// Existing intent is more or equally severe
			return false
		}
	}

	// New trigger is more severe than all existing intents
	return true
}

// getTriggerSeverity returns severity score (higher = more severe)
func getTriggerSeverity(reasonCode string) int {
	switch reasonCode {
	case exit.ReasonSL2:
		return 100 // Most severe (full stop loss)
	case exit.ReasonSL1:
		return 90 // Partial stop loss
	case exit.ReasonStopFloor:
		return 80 // Breakeven protection
	case exit.ReasonTP3:
		return 30
	case exit.ReasonTP2:
		return 20
	case exit.ReasonTP1:
		return 10
	case exit.ReasonTrail:
		return 5
	case exit.ReasonTime:
		return 1
	default:
		return 0
	}
}

// PositionSnapshot represents a position snapshot at evaluation start (v10 방어)
type PositionSnapshot struct {
	PositionID  uuid.UUID
	Symbol      string
	Qty         int64
	OriginalQty int64           // Original entry quantity (for TP % calculation)
	AvgPrice    decimal.Decimal
	EntryTS     time.Time
	Version     int
	Phase       string // FSM Phase (for action_key generation)
}

// createIntentWithVersionCheck creates an intent with version check (v10 방어)
func (s *Service) createIntentWithVersionCheck(ctx context.Context, snapshot PositionSnapshot, trigger *exit.ExitTrigger) error {
	// 1. Re-check position version (v10 방어: 버전 기반 낙관적 잠금)
	pos, err := s.posRepo.GetPosition(ctx, snapshot.PositionID)
	if err != nil {
		return err
	}

	// 2. Version mismatch detection
	if pos.Version != snapshot.Version {
		log.Warn().
			Str("symbol", snapshot.Symbol).
			Int("old_version", snapshot.Version).
			Int("new_version", pos.Version).
			Str("old_avg_price", snapshot.AvgPrice.String()).
			Str("new_avg_price", pos.AvgPrice.String()).
			Msg("Position changed during evaluation, need re-evaluation")
		return exit.ErrPositionChanged
	}

	// 3. Get available qty (v10 방어: Locked Qty 차감)
	availableQty, err := s.posRepo.GetAvailableQty(ctx, snapshot.PositionID)
	if err != nil {
		return err
	}

	if availableQty <= 0 {
		log.Debug().
			Str("symbol", snapshot.Symbol).
			Int64("available_qty", availableQty).
			Msg("No available qty, intent creation skipped")
		return exit.ErrNoAvailableQty
	}

	// 4. Clamp qty to available
	qty := trigger.Qty
	if qty > availableQty {
		log.Warn().
			Str("symbol", snapshot.Symbol).
			Int64("trigger_qty", trigger.Qty).
			Int64("available_qty", availableQty).
			Msg("Clamping qty to available")
		qty = availableQty
	}

	// 5. Determine intent type
	intentType := exit.IntentTypeExitPartial
	if qty == pos.Qty {
		intentType = exit.IntentTypeExitFull
	}

	// 6. Create intent (멱등) - PENDING_APPROVAL 상태로 생성 (사용자 승인 대기)
	// action_key에 Phase 포함 → 평단가 리셋 후 재발동 가능
	actionKey := fmt.Sprintf("%s:%s:%s", snapshot.PositionID.String(), snapshot.Phase, trigger.ReasonCode)
	intent := &exit.OrderIntent{
		IntentID:     uuid.New(),
		PositionID:   snapshot.PositionID,
		Symbol:       snapshot.Symbol,
		IntentType:   intentType,
		Qty:          qty,
		OrderType:    trigger.OrderType,
		LimitPrice:   trigger.LimitPrice,
		ReasonCode:   trigger.ReasonCode,
		ReasonDetail: trigger.ReasonDetail, // 상세 사유 (Custom rule description)
		ActionKey:    actionKey,
		Status:       exit.IntentStatusPendingApproval, // 사용자 승인 대기
	}

	err = s.intentRepo.CreateIntent(ctx, intent)
	if err == exit.ErrIntentExists {
		// Idempotent (already exists)
		log.Debug().
			Str("symbol", snapshot.Symbol).
			Str("action_key", actionKey).
			Msg("Intent already exists (idempotent)")
		return nil
	}

	if err != nil {
		return fmt.Errorf("create intent: %w", err)
	}

	log.Info().
		Str("symbol", snapshot.Symbol).
		Str("reason", trigger.ReasonCode).
		Int64("qty", qty).
		Str("type", intentType).
		Msg("Exit intent created")

	// 7. Update position status to CLOSING (Exit Engine owns status)
	// Use version from snapshot to ensure consistency
	if pos.Status == exit.StatusOpen {
		err = s.posRepo.UpdateStatus(ctx, snapshot.PositionID, exit.StatusClosing, snapshot.Version)
		if err != nil {
			log.Warn().
				Err(err).
				Str("symbol", snapshot.Symbol).
				Msg("Failed to update position status to CLOSING (non-fatal)")
			// Non-fatal: Intent already created, status update is best-effort
		} else {
			log.Info().
				Str("symbol", snapshot.Symbol).
				Str("old_status", exit.StatusOpen).
				Str("new_status", exit.StatusClosing).
				Msg("Position status updated to CLOSING")
		}
	}

	return nil
}
