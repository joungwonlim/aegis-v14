package exit

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// scaleTriggerPct scales trigger percentage based on ATR
// Returns scaled percentage clamped between min and max
// If minPct or maxPct is 0, no clamping is applied for that bound
//
// For SL (negative): MinPct is tighter (closer to 0), MaxPct is wider (further from 0)
//   Example: MinPct=-3%, MaxPct=-8%, means -8% <= threshold <= -3%
// For TP (positive): MinPct is tighter (closer to 0), MaxPct is wider (further from 0)
//   Example: MinPct=+5%, MaxPct=+10%, means +5% <= threshold <= +10%
func scaleTriggerPct(basePct, minPct, maxPct float64, atrFactor float64) float64 {
	scaledPct := basePct * atrFactor

	if basePct < 0 {
		// SL (negative values): MinPct > MaxPct (e.g., -3% > -8%)
		// Clamp to tighter stop (closer to 0)
		if minPct != 0 && scaledPct > minPct {
			return minPct
		}
		// Clamp to wider stop (further from 0)
		if maxPct != 0 && scaledPct < maxPct {
			return maxPct
		}
	} else {
		// TP (positive values): MinPct < MaxPct (e.g., 5% < 10%)
		// Clamp to tighter target (closer to 0)
		if minPct != 0 && scaledPct < minPct {
			return minPct
		}
		// Clamp to wider target (further from 0)
		if maxPct != 0 && scaledPct > maxPct {
			return maxPct
		}
	}

	return scaledPct
}

// calculateATRFactor calculates ATR scaling factor
// Returns factor clamped between factorMin and factorMax
func calculateATRFactor(currentATR *decimal.Decimal, atrConfig exit.ATRConfig) float64 {
	// If ATR not available, use factor 1.0 (no scaling)
	if currentATR == nil || currentATR.IsZero() {
		return 1.0
	}

	// If ATR Ref is 0, no scaling
	if atrConfig.Ref == 0 {
		return 1.0
	}

	// Calculate factor: current_atr / atr_ref
	currentATRFloat, _ := currentATR.Float64()
	factor := currentATRFloat / atrConfig.Ref

	// Clamp to [factorMin, factorMax]
	if factor < atrConfig.FactorMin {
		return atrConfig.FactorMin
	}
	if factor > atrConfig.FactorMax {
		return atrConfig.FactorMax
	}

	return factor
}

// evaluateTriggers evaluates all exit triggers in priority order
// Returns the highest priority trigger that is hit, or nil if none
//
// Priority (high → low):
// 1. SL2 (full stop loss) - 가장 위험, 전량 손절
// 2. STOP_FLOOR (본전 방어, TP1 체결 후)
// 3. SL1 (partial stop loss) - 50% 부분 손절
// 4. TP1 (take profit 1) - 5% 도달 시 10% 익절
// 5. TP2 (take profit 2) - 10% 도달 시 20% 익절
// 6. TP3 (take profit 3) - 15% 도달 시 30% 익절
// 7. TRAIL (trailing stop, TP2_DONE/TRAILING_ACTIVE phase)
// 8. TIME (time-based exit)
//
// Control Mode Filtering:
// - PAUSE_PROFIT: Only SL/STOP_FLOOR triggers (block TP/TRAIL)
// - PAUSE_ALL: No triggers (except HardStop if configured)
func (s *Service) evaluateTriggers(
	snapshot PositionSnapshot,
	state *exit.PositionState,
	bestPrice *price.BestPrice,
	profile *exit.ExitProfile,
	controlMode string,
) *exit.ExitTrigger {
	// Calculate ATR factor for dynamic scaling
	atrFactor := calculateATRFactor(state.ATR, profile.Config.ATR)

	log.Debug().
		Str("symbol", snapshot.Symbol).
		Str("atr", func() string {
			if state.ATR != nil {
				return state.ATR.String()
			}
			return "nil"
		}()).
		Float64("atr_factor", atrFactor).
		Msg("ATR factor calculated")
	// Calculate P&L
	// Use BidPrice for exit (conservative), fallback to BestPrice
	var currentPriceInt int64
	if bestPrice.BidPrice != nil {
		currentPriceInt = *bestPrice.BidPrice
	} else {
		currentPriceInt = bestPrice.BestPrice
	}
	currentPrice := decimal.NewFromInt(currentPriceInt)
	pnlPct := currentPrice.Sub(snapshot.AvgPrice).Div(snapshot.AvgPrice).Mul(decimal.NewFromInt(100))

	log.Debug().
		Str("symbol", snapshot.Symbol).
		Str("phase", state.Phase).
		Str("current_price", currentPrice.String()).
		Str("avg_price", snapshot.AvgPrice.String()).
		Str("pnl_pct", pnlPct.StringFixed(2)).
		Msg("Evaluating triggers")

	// ✅ Priority 0: HARDSTOP (bypasses control mode)
	// Emergency stop loss that works even in PAUSE_ALL mode
	if trigger := s.evaluateHardStop(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Control Mode filtering (AFTER HardStop)
	if controlMode == exit.ControlModePauseAll {
		log.Debug().Str("symbol", snapshot.Symbol).Msg("PAUSE_ALL mode, skipping triggers")
		return nil
	}

	// Priority 1: SL2 (Hard Stop Loss - full)
	if trigger := s.evaluateSL2(snapshot, pnlPct, profile, atrFactor); trigger != nil {
		return trigger
	}

	// Priority 2: STOP_FLOOR (본전 방어, TP1 체결 후)
	// v14 핵심 안전장치: SL1보다 먼저 평가
	// TP1_DONE 또는 TP2_DONE phase에서만 평가
	if state.Phase == exit.PhaseTP1Done || state.Phase == exit.PhaseTP2Done || state.Phase == exit.PhaseTP3Done {
		if trigger := s.evaluateStopFloor(snapshot, currentPrice, state); trigger != nil {
			return trigger
		}
	}

	// Priority 3: SL1 (Partial Stop Loss)
	if trigger := s.evaluateSL1(snapshot, pnlPct, profile, atrFactor); trigger != nil {
		return trigger
	}

	// If PAUSE_PROFIT mode, block TP/TRAIL
	if controlMode == exit.ControlModePauseProfit {
		log.Debug().Str("symbol", snapshot.Symbol).Msg("PAUSE_PROFIT mode, blocking TP/TRAIL")
		return nil
	}

	// Priority 4: TP1 (5% 도달 시 10% 매도)
	if trigger := s.evaluateTP1(snapshot, pnlPct, currentPrice, profile, atrFactor); trigger != nil {
		return trigger
	}

	// Priority 5: TP2 (10% 도달 시 20% 매도)
	if trigger := s.evaluateTP2(snapshot, pnlPct, currentPrice, profile, atrFactor); trigger != nil {
		return trigger
	}

	// Priority 6: TP3 (15% 도달 시 30% 매도)
	if trigger := s.evaluateTP3(snapshot, pnlPct, currentPrice, profile, atrFactor); trigger != nil {
		return trigger
	}

	// Priority 7: TRAIL (Phase 1: TP2_DONE or TRAILING_ACTIVE phase)
	// - TP2_DONE: 원본 20% 부분 트레일 (단발)
	// - TRAILING_ACTIVE: 잔량 100% 전량 트레일
	if state.Phase == exit.PhaseTP2Done || state.Phase == exit.PhaseTrailingActive {
		if trigger := s.evaluateTrailing(snapshot, currentPrice, state, profile); trigger != nil {
			return trigger
		}
	}

	// Priority 8: TIME
	if trigger := s.evaluateTimeStop(snapshot, state, currentPrice, profile); trigger != nil {
		return trigger
	}

	// No trigger hit
	return nil
}

// evaluateSL2 evaluates SL2 (full stop loss) trigger with ATR scaling
func (s *Service) evaluateSL2(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile, atrFactor float64) *exit.ExitTrigger {
	// Calculate ATR-scaled threshold
	scaledPct := scaleTriggerPct(
		profile.Config.SL2.BasePct,
		profile.Config.SL2.MinPct,
		profile.Config.SL2.MaxPct,
		atrFactor,
	)
	threshold := decimal.NewFromFloat(scaledPct * 100) // Convert to %

	if pnlPct.LessThanOrEqual(threshold) {
		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Float64("atr_factor", atrFactor).
			Msg("SL2 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonSL2,
			Qty:        snapshot.Qty, // Full qty
			OrderType:  exit.OrderTypeMKT,
		}
	}

	return nil
}

// evaluateSL1 evaluates SL1 (partial stop loss) trigger with ATR scaling
func (s *Service) evaluateSL1(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile, atrFactor float64) *exit.ExitTrigger {
	// Calculate ATR-scaled threshold
	scaledPct := scaleTriggerPct(
		profile.Config.SL1.BasePct,
		profile.Config.SL1.MinPct,
		profile.Config.SL1.MaxPct,
		atrFactor,
	)
	threshold := decimal.NewFromFloat(scaledPct * 100) // Convert to %

	if pnlPct.LessThanOrEqual(threshold) {
		// Partial exit
		qtyPct := profile.Config.SL1.QtyPct
		qty := int64(float64(snapshot.Qty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Float64("atr_factor", atrFactor).
			Int64("qty", qty).
			Msg("SL1 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonSL1,
			Qty:        qty,
			OrderType:  exit.OrderTypeMKT,
		}
	}

	return nil
}

// evaluateStopFloor evaluates Stop Floor trigger (본전 방어, TP1 체결 후)
// Phase 1: 2틱 연속 breach 확인 (confirm_ticks=2, 노이즈 청산 방지)
func (s *Service) evaluateStopFloor(snapshot PositionSnapshot, currentPrice decimal.Decimal, state *exit.PositionState) *exit.ExitTrigger {
	// Check if Stop Floor is set
	if state.StopFloorPrice == nil {
		log.Warn().Str("symbol", snapshot.Symbol).Msg("Stop Floor phase but price not set, skipping")
		return nil
	}

	ctx := s.ctx

	// Check if current price hit Stop Floor
	if currentPrice.LessThanOrEqual(*state.StopFloorPrice) {
		// Phase 1: Increment breach counter
		err := s.stateRepo.IncrementStopFloorBreachTicks(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to increment stop_floor_breach_ticks")
			return nil
		}

		// Reload state to get updated stop_floor_breach_ticks
		state, err = s.stateRepo.GetState(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to reload state")
			return nil
		}

		log.Debug().
			Str("symbol", snapshot.Symbol).
			Str("current_price", currentPrice.String()).
			Str("stop_floor_price", state.StopFloorPrice.String()).
			Int("stop_floor_breach_ticks", state.StopFloorBreachTicks).
			Msg("Stop Floor breach detected")

		// Phase 1: confirm_ticks=2 (6초 연속 조건 충족 필요)
		if state.StopFloorBreachTicks >= 2 {
			log.Info().
				Str("symbol", snapshot.Symbol).
				Str("current_price", currentPrice.String()).
				Str("stop_floor_price", state.StopFloorPrice.String()).
				Int("stop_floor_breach_ticks", state.StopFloorBreachTicks).
				Msg("Stop Floor trigger hit (confirmed)")

			// Reset counter after trigger
			_ = s.stateRepo.ResetStopFloorBreachTicks(ctx, snapshot.PositionID)

			return &exit.ExitTrigger{
				ReasonCode: exit.ReasonStopFloor,
				Qty:        snapshot.Qty, // Full qty (remaining)
				OrderType:  exit.OrderTypeMKT,
			}
		}

		// Not yet confirmed (need more ticks)
		return nil
	}

	// Breach 조건 미충족: 카운터 리셋
	if state.StopFloorBreachTicks > 0 {
		err := s.stateRepo.ResetStopFloorBreachTicks(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to reset stop_floor_breach_ticks")
		} else {
			log.Debug().
				Str("symbol", snapshot.Symbol).
				Msg("Stop Floor breach condition cleared, reset stop_floor_breach_ticks")
		}
	}

	return nil
}

// evaluateTP1 evaluates TP1 (take profit 1) trigger with ATR scaling
func (s *Service) evaluateTP1(snapshot PositionSnapshot, pnlPct, currentPrice decimal.Decimal, profile *exit.ExitProfile, atrFactor float64) *exit.ExitTrigger {
	// Calculate ATR-scaled threshold
	scaledPct := scaleTriggerPct(
		profile.Config.TP1.BasePct,
		profile.Config.TP1.MinPct,
		profile.Config.TP1.MaxPct,
		atrFactor,
	)
	threshold := decimal.NewFromFloat(scaledPct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// v14: 원본 수량 기준 계산 (OriginalQty)
		qtyPct := profile.Config.TP1.QtyPct

		// QtyPct=0이면 TP1 비활성화
		if qtyPct <= 0 {
			return nil
		}

		qty := int64(float64(snapshot.OriginalQty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		// 현재 잔량보다 많으면 잔량으로 제한
		if qty > snapshot.Qty {
			qty = snapshot.Qty
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Float64("atr_factor", atrFactor).
			Int64("original_qty", snapshot.OriginalQty).
			Int64("current_qty", snapshot.Qty).
			Int64("qty", qty).
			Float64("qty_pct", qtyPct).
			Msg("TP1 trigger hit (원본 기준)")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP1,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			LimitPrice: &currentPrice,
		}
	}

	return nil
}

// evaluateTP2 evaluates TP2 (take profit 2) trigger with ATR scaling
func (s *Service) evaluateTP2(snapshot PositionSnapshot, pnlPct, currentPrice decimal.Decimal, profile *exit.ExitProfile, atrFactor float64) *exit.ExitTrigger {
	// Calculate ATR-scaled threshold
	scaledPct := scaleTriggerPct(
		profile.Config.TP2.BasePct,
		profile.Config.TP2.MinPct,
		profile.Config.TP2.MaxPct,
		atrFactor,
	)
	threshold := decimal.NewFromFloat(scaledPct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// v14: 원본 수량 기준 계산 (OriginalQty)
		qtyPct := profile.Config.TP2.QtyPct

		// QtyPct=0이면 TP2 비활성화
		if qtyPct <= 0 {
			return nil
		}

		qty := int64(float64(snapshot.OriginalQty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		// 현재 잔량보다 많으면 잔량으로 제한
		if qty > snapshot.Qty {
			qty = snapshot.Qty
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Float64("atr_factor", atrFactor).
			Int64("original_qty", snapshot.OriginalQty).
			Int64("current_qty", snapshot.Qty).
			Int64("qty", qty).
			Float64("qty_pct", qtyPct).
			Msg("TP2 trigger hit (원본 기준)")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP2,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			LimitPrice: &currentPrice,
		}
	}

	return nil
}

// evaluateTP3 evaluates TP3 (take profit 3) trigger with ATR scaling
func (s *Service) evaluateTP3(snapshot PositionSnapshot, pnlPct, currentPrice decimal.Decimal, profile *exit.ExitProfile, atrFactor float64) *exit.ExitTrigger {
	// Calculate ATR-scaled threshold
	scaledPct := scaleTriggerPct(
		profile.Config.TP3.BasePct,
		profile.Config.TP3.MinPct,
		profile.Config.TP3.MaxPct,
		atrFactor,
	)
	threshold := decimal.NewFromFloat(scaledPct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// v14: 원본 수량 기준 계산 (OriginalQty)
		qtyPct := profile.Config.TP3.QtyPct

		// QtyPct=0이면 TP3 비활성화
		if qtyPct <= 0 {
			return nil
		}

		qty := int64(float64(snapshot.OriginalQty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		// 현재 잔량보다 많으면 잔량으로 제한
		if qty > snapshot.Qty {
			qty = snapshot.Qty
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Float64("atr_factor", atrFactor).
			Int64("original_qty", snapshot.OriginalQty).
			Int64("current_qty", snapshot.Qty).
			Int64("qty", qty).
			Float64("qty_pct", qtyPct).
			Msg("TP3 trigger hit (원본 기준)")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP3,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			LimitPrice: &currentPrice,
		}
	}

	return nil
}

// evaluateTrailing evaluates trailing stop trigger
// Phase 1: Phase별 분기 + 2틱 연속 확인 (confirm_ticks=2)
// - TP2_DONE: 원본 20% 부분 트레일 (단발, fire_once)
// - TRAIL_ACTIVE: 잔량 100% 전량 트레일
func (s *Service) evaluateTrailing(snapshot PositionSnapshot, currentPrice decimal.Decimal, state *exit.PositionState, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Check if HWM is set
	if state.HWMPrice == nil {
		log.Warn().Str("symbol", snapshot.Symbol).Str("phase", state.Phase).Msg("Trailing phase but HWM not set, skipping")
		return nil
	}

	ctx := s.ctx

	// Calculate trailing stop price
	trailingPct := decimal.NewFromFloat(profile.Config.Trailing.PctTrail) // e.g., 0.03 (3%)
	trailingStopPrice := state.HWMPrice.Mul(decimal.NewFromInt(1).Sub(trailingPct))

	if currentPrice.LessThanOrEqual(trailingStopPrice) {
		// Phase 1: Increment breach counter
		err := s.stateRepo.IncrementTrailingBreachTicks(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to increment trailing_breach_ticks")
			return nil
		}

		// Reload state to get updated trailing_breach_ticks
		state, err = s.stateRepo.GetState(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to reload state")
			return nil
		}

		log.Debug().
			Str("symbol", snapshot.Symbol).
			Str("phase", state.Phase).
			Str("current_price", currentPrice.String()).
			Str("hwm_price", state.HWMPrice.String()).
			Str("trailing_stop_price", trailingStopPrice.String()).
			Int("trailing_breach_ticks", state.TrailingBreachTicks).
			Msg("Trailing breach detected")

		// Phase 1: confirm_ticks=2 (6초 연속 조건 충족 필요)
		if state.TrailingBreachTicks >= 2 {
			// Phase별 수량 계산
			var qty int64
			var reasonCode string

			if state.Phase == exit.PhaseTP2Done {
				// v14: TP2 부분 트레일 - 원본의 20% (OriginalQty 기준)
				qtyPct := profile.Config.TP2.QtyPct
				qty = int64(float64(snapshot.OriginalQty) * qtyPct)
				if qty < 1 {
					qty = 1
				}

				// 현재 잔량보다 많으면 잔량으로 제한
				if qty > snapshot.Qty {
					qty = snapshot.Qty
				}

				// Phase 1: fire_once 보장 (action_key 멱등으로 처리됨)
				reasonCode = exit.ReasonTrailPartial

				log.Info().
					Str("symbol", snapshot.Symbol).
					Str("phase", state.Phase).
					Str("current_price", currentPrice.String()).
					Str("hwm_price", state.HWMPrice.String()).
					Str("trailing_stop_price", trailingStopPrice.String()).
					Int64("original_qty", snapshot.OriginalQty).
					Int64("current_qty", snapshot.Qty).
					Int64("qty", qty).
					Float64("qty_pct", qtyPct).
					Int("trailing_breach_ticks", state.TrailingBreachTicks).
					Msg("Trailing PARTIAL trigger hit (원본 기준, TP2 부분 트레일, confirmed)")
			} else {
				// TRAIL_ACTIVE: 잔량 전량
				qty = snapshot.Qty
				reasonCode = exit.ReasonTrail

				log.Info().
					Str("symbol", snapshot.Symbol).
					Str("phase", state.Phase).
					Str("current_price", currentPrice.String()).
					Str("hwm_price", state.HWMPrice.String()).
					Str("trailing_stop_price", trailingStopPrice.String()).
					Int64("qty", qty).
					Int("trailing_breach_ticks", state.TrailingBreachTicks).
					Msg("Trailing FULL trigger hit (confirmed)")
			}

			// Reset counter after trigger
			_ = s.stateRepo.ResetTrailingBreachTicks(ctx, snapshot.PositionID)

			return &exit.ExitTrigger{
				ReasonCode: reasonCode,
				Qty:        qty,
				OrderType:  exit.OrderTypeMKT,
			}
		}

		// Not yet confirmed (need more ticks)
		return nil
	}

	// Breach 조건 미충족: 카운터 리셋
	if state.TrailingBreachTicks > 0 {
		err := s.stateRepo.ResetTrailingBreachTicks(ctx, snapshot.PositionID)
		if err != nil {
			log.Error().Err(err).Str("symbol", snapshot.Symbol).Msg("Failed to reset trailing_breach_ticks")
		} else {
			log.Debug().
				Str("symbol", snapshot.Symbol).
				Str("phase", state.Phase).
				Msg("Trailing breach condition cleared, reset trailing_breach_ticks")
		}
	}

	return nil
}

// evaluateTimeStop evaluates time-based exit trigger
func (s *Service) evaluateTimeStop(snapshot PositionSnapshot, state *exit.PositionState, currentPrice decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Check if max hold days is configured
	if profile.Config.TimeStop.MaxHoldDays <= 0 {
		return nil
	}

	// Calculate holding days
	holdingDays := int(time.Since(snapshot.EntryTS).Hours() / 24)

	// Condition 1: Max hold days exceeded
	if holdingDays >= profile.Config.TimeStop.MaxHoldDays {
		log.Info().
			Str("symbol", snapshot.Symbol).
			Int("holding_days", holdingDays).
			Int("max_hold_days", profile.Config.TimeStop.MaxHoldDays).
			Msg("TIME_STOP: Max hold days exceeded")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTime,
			Qty:        snapshot.Qty, // Full qty (remaining)
			OrderType:  exit.OrderTypeMKT,
		}
	}

	// Condition 2: No momentum (optional)
	if profile.Config.TimeStop.NoMomentumDays > 0 {
		// Calculate max profit during holding period
		var maxProfitPct decimal.Decimal
		if state.HWMPrice != nil {
			// HWM exists, calculate max profit from HWM
			maxProfitPct = state.HWMPrice.Sub(snapshot.AvgPrice).Div(snapshot.AvgPrice)
		} else {
			// No HWM, use current price
			maxProfitPct = currentPrice.Sub(snapshot.AvgPrice).Div(snapshot.AvgPrice)
		}

		noMomentumThreshold := decimal.NewFromFloat(profile.Config.TimeStop.NoMomentumProfit)

		if holdingDays >= profile.Config.TimeStop.NoMomentumDays && maxProfitPct.LessThan(noMomentumThreshold) {
			log.Info().
				Str("symbol", snapshot.Symbol).
				Int("holding_days", holdingDays).
				Int("no_momentum_days", profile.Config.TimeStop.NoMomentumDays).
				Str("max_profit_pct", maxProfitPct.StringFixed(4)).
				Str("threshold", noMomentumThreshold.StringFixed(4)).
				Msg("TIME_STOP: No momentum")

			return &exit.ExitTrigger{
				ReasonCode: exit.ReasonTime,
				Qty:        snapshot.Qty, // Full qty (remaining)
				OrderType:  exit.OrderTypeMKT,
			}
		}
	}

	return nil
}

// evaluateHardStop evaluates HardStop (emergency stop loss) trigger
// HardStop bypasses all control modes (PAUSE_ALL, PAUSE_PROFIT)
// This is the last line of defense against catastrophic losses
func (s *Service) evaluateHardStop(
	snapshot PositionSnapshot,
	pnlPct decimal.Decimal,
	profile *exit.ExitProfile,
) *exit.ExitTrigger {
	// Check if HardStop is enabled
	if !profile.Config.HardStop.Enabled {
		return nil
	}

	// HardStop threshold (e.g., -10%)
	threshold := decimal.NewFromFloat(profile.Config.HardStop.Pct)
	thresholdPct := threshold.Mul(decimal.NewFromInt(100))

	// Trigger if P&L <= threshold (e.g., -12% <= -10%)
	if pnlPct.LessThanOrEqual(thresholdPct) {
		log.Warn().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", thresholdPct.StringFixed(2)).
			Msg("🚨 HARDSTOP TRIGGERED (Emergency Stop)")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonHardStop,
			Qty:        snapshot.Qty, // Full qty (emergency exit)
			OrderType:  exit.OrderTypeMKT,
		}
	}

	return nil
}
