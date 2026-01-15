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
	evaluationInterval     = 3 * time.Second  // 1~5초 권장
	reconciliationInterval = 30 * time.Second // Intent 조정 주기
	maxRetries             = 3                // 최대 재평가 횟수
	freshnessThreshold     = 10 * time.Second // 가격 신선도 임계값
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

// reconciliationLoop runs the intent reconciliation loop (30초 주기)
func (s *Service) reconciliationLoop() {
	ticker := time.NewTicker(reconciliationInterval)
	defer ticker.Stop()

	// Wait a bit before first run to let other services initialize
	time.Sleep(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			// Reconcile intents with actual holdings/fills
			if err := s.ReconcileIntents(s.ctx); err != nil {
				log.Error().Err(err).Msg("Intent reconciliation failed")
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// evaluateAllPositions evaluates all OPEN positions for exit triggers
func (s *Service) evaluateAllPositions(ctx context.Context) error {
	// 1. Check Control Gate
	control, err := s.controlRepo.GetControl(ctx)
	if err != nil {
		return fmt.Errorf("get control: %w", err)
	}

	log.Debug().Str("mode", control.Mode).Msg("Exit control mode")

	// 2. Load OPEN positions (모든 계정)
	// NOTE: 모든 OPEN positions를 조회 (account 필터링 없음)
	positions, err := s.posRepo.GetAllOpenPositions(ctx)
	if err != nil {
		return fmt.Errorf("get open positions: %w", err)
	}

	log.Debug().Int("count", len(positions)).Msg("Evaluating positions")

	// 3. Evaluate each position
	for _, pos := range positions {
		// Skip disabled positions
		if pos.ExitMode == exit.ExitModeDisabled {
			log.Debug().Str("symbol", pos.Symbol).Msg("Position exit disabled, skipping")
			continue
		}

		// Evaluate position with retry
		if err := s.evaluatePositionWithRetry(ctx, pos, control.Mode, 0); err != nil {
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
	// 1. Create position snapshot (v10 방어: 평가 시작 시 snapshot)
	snapshot := PositionSnapshot{
		PositionID:  pos.PositionID,
		Symbol:      pos.Symbol,
		Qty:         pos.Qty,
		OriginalQty: pos.OriginalQty,
		AvgPrice:    pos.AvgPrice,
		EntryTS:     pos.EntryTS,
		Version:     pos.Version,
	}

	// 2. Get position state (FSM)
	state, err := s.stateRepo.GetState(ctx, pos.PositionID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	// 2.5. Check for 평단가 변경 (추가 매수 감지)
	if state.LastAvgPrice != nil {
		// Tolerance: 0.5% 이상 변경 시에만 리셋
		// 계산 오차나 미세한 반올림 차이는 무시
		diff := pos.AvgPrice.Sub(*state.LastAvgPrice).Abs()
		threshold := state.LastAvgPrice.Mul(decimal.NewFromFloat(0.005)) // 0.5%

		if diff.GreaterThan(threshold) {
			log.Warn().
				Str("symbol", pos.Symbol).
				Str("old_avg_price", state.LastAvgPrice.String()).
				Str("new_avg_price", pos.AvgPrice.String()).
				Str("diff", diff.String()).
				Str("threshold", threshold.String()).
				Str("old_phase", state.Phase).
				Msg("평단가 유의미한 변경 감지: Exit State를 OPEN으로 리셋 (추가매수로 인한 재진입)")

			// State를 OPEN으로 리셋
			err = s.stateRepo.ResetStateToOpen(ctx, pos.PositionID, pos.AvgPrice)
			if err != nil {
				return fmt.Errorf("reset state to open: %w", err)
			}

			// State 재로드
			state, err = s.stateRepo.GetState(ctx, pos.PositionID)
			if err != nil {
				return fmt.Errorf("get state after reset: %w", err)
			}
		} else if diff.GreaterThan(decimal.Zero) {
			// 미세한 변경 (threshold 이하): 로그만 남기고 무시
			log.Debug().
				Str("symbol", pos.Symbol).
				Str("old_avg_price", state.LastAvgPrice.String()).
				Str("new_avg_price", pos.AvgPrice.String()).
				Str("diff", diff.String()).
				Str("threshold", threshold.String()).
				Msg("평단가 미세 변동 감지 (threshold 이하): 무시")
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
		log.Warn().
			Str("symbol", pos.Symbol).
			Msg("Price is stale, skipping evaluation (Fail-Closed)")
		return exit.ErrStalePrice
	}

	// Check timestamp age
	age := time.Since(bestPrice.BestTS)
	if age > freshnessThreshold {
		log.Warn().
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

	// 7. Evaluate triggers (우선순위 순서)
	trigger := s.evaluateTriggers(snapshot, state, bestPrice, profile, controlMode)

	// 7.5. Record exit signal for debugging/backtest (best-effort, non-blocking)
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

	// 8. Create intent (v10 방어: Intent 생성 직전 DB 재확인)
	return s.createIntentWithVersionCheck(ctx, snapshot, trigger)
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
		log.Warn().
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
	actionKey := fmt.Sprintf("%s:%s", snapshot.PositionID.String(), trigger.ReasonCode)
	intent := &exit.OrderIntent{
		IntentID:   uuid.New(),
		PositionID: snapshot.PositionID,
		Symbol:     snapshot.Symbol,
		IntentType: intentType,
		Qty:        qty,
		OrderType:  trigger.OrderType,
		LimitPrice: trigger.LimitPrice,
		ReasonCode: trigger.ReasonCode,
		ActionKey:  actionKey,
		Status:     exit.IntentStatusPendingApproval, // 사용자 승인 대기
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
