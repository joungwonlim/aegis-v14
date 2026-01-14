package exit

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// evaluateTriggers evaluates all exit triggers in priority order
// Returns the highest priority trigger that is hit, or nil if none
//
// Priority (high → low):
// 1. SL2 (full stop loss) - 가장 위험
// 2. SL1 (partial stop loss)
// 3. STOP_FLOOR (본전 방어, TP1 체결 후)
// 4. TP3 (take profit 3)
// 5. TP2 (take profit 2)
// 6. TP1 (take profit 1)
// 7. TRAIL (trailing stop, only in TRAILING_ACTIVE phase)
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

	// Control Mode filtering
	if controlMode == exit.ControlModePauseAll {
		// No triggers (except HardStop if configured)
		// TODO: Implement HardStop bypass
		log.Debug().Str("symbol", snapshot.Symbol).Msg("PAUSE_ALL mode, skipping triggers")
		return nil
	}

	// Priority 1: SL2 (Hard Stop Loss - full)
	if trigger := s.evaluateSL2(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 2: SL1 (Partial Stop Loss)
	if trigger := s.evaluateSL1(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 3: STOP_FLOOR (본전 방어, TP1 체결 후)
	// TP1_DONE 또는 TP2_DONE phase에서만 평가
	if state.Phase == exit.PhaseTP1Done || state.Phase == exit.PhaseTP2Done {
		if trigger := s.evaluateStopFloor(snapshot, currentPrice, state); trigger != nil {
			return trigger
		}
	}

	// If PAUSE_PROFIT mode, block TP/TRAIL
	if controlMode == exit.ControlModePauseProfit {
		log.Debug().Str("symbol", snapshot.Symbol).Msg("PAUSE_PROFIT mode, blocking TP/TRAIL")
		return nil
	}

	// Priority 4: TP3
	if trigger := s.evaluateTP3(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 5: TP2
	if trigger := s.evaluateTP2(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 6: TP1
	if trigger := s.evaluateTP1(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 7: TRAIL (only in TRAILING_ACTIVE phase)
	if state.Phase == exit.PhaseTrailingActive {
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

// evaluateSL2 evaluates SL2 (full stop loss) trigger
func (s *Service) evaluateSL2(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Calculate scaled threshold (ATR-based)
	// For now, use base threshold
	threshold := decimal.NewFromFloat(profile.Config.SL2.BasePct * 100) // Convert to %

	if pnlPct.LessThanOrEqual(threshold) {
		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Msg("SL2 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonSL2,
			Qty:        snapshot.Qty, // Full qty
			OrderType:  exit.OrderTypeMKT,
		}
	}

	return nil
}

// evaluateSL1 evaluates SL1 (partial stop loss) trigger
func (s *Service) evaluateSL1(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Calculate scaled threshold
	threshold := decimal.NewFromFloat(profile.Config.SL1.BasePct * 100) // Convert to %

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
func (s *Service) evaluateStopFloor(snapshot PositionSnapshot, currentPrice decimal.Decimal, state *exit.PositionState) *exit.ExitTrigger {
	// Check if Stop Floor is set
	if state.StopFloorPrice == nil {
		log.Warn().Str("symbol", snapshot.Symbol).Msg("Stop Floor phase but price not set, skipping")
		return nil
	}

	// Check if current price hit Stop Floor
	if currentPrice.LessThanOrEqual(*state.StopFloorPrice) {
		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("current_price", currentPrice.String()).
			Str("stop_floor_price", state.StopFloorPrice.String()).
			Msg("Stop Floor trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonStopFloor,
			Qty:        snapshot.Qty, // Full qty (remaining)
			OrderType:  exit.OrderTypeMKT,
		}
	}

	return nil
}

// evaluateTP1 evaluates TP1 (take profit 1) trigger
func (s *Service) evaluateTP1(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Calculate scaled threshold
	threshold := decimal.NewFromFloat(profile.Config.TP1.BasePct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// Partial exit
		qtyPct := profile.Config.TP1.QtyPct
		qty := int64(float64(snapshot.Qty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Int64("qty", qty).
			Msg("TP1 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP1,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			// TODO: Calculate limit price with slippage
		}
	}

	return nil
}

// evaluateTP2 evaluates TP2 (take profit 2) trigger
func (s *Service) evaluateTP2(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Calculate scaled threshold
	threshold := decimal.NewFromFloat(profile.Config.TP2.BasePct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// Partial exit
		qtyPct := profile.Config.TP2.QtyPct
		qty := int64(float64(snapshot.Qty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Int64("qty", qty).
			Msg("TP2 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP2,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			// TODO: Calculate limit price with slippage
		}
	}

	return nil
}

// evaluateTP3 evaluates TP3 (take profit 3) trigger
func (s *Service) evaluateTP3(snapshot PositionSnapshot, pnlPct decimal.Decimal, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Calculate scaled threshold
	threshold := decimal.NewFromFloat(profile.Config.TP3.BasePct * 100) // Convert to %

	if pnlPct.GreaterThanOrEqual(threshold) {
		// Partial exit
		qtyPct := profile.Config.TP3.QtyPct
		qty := int64(float64(snapshot.Qty) * qtyPct)
		if qty < 1 {
			qty = 1
		}

		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("pnl_pct", pnlPct.StringFixed(2)).
			Str("threshold", threshold.StringFixed(2)).
			Int64("qty", qty).
			Msg("TP3 trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTP3,
			Qty:        qty,
			OrderType:  exit.OrderTypeLMT,
			// TODO: Calculate limit price with slippage
		}
	}

	return nil
}

// evaluateTrailing evaluates trailing stop trigger (only in TRAILING_ACTIVE phase)
func (s *Service) evaluateTrailing(snapshot PositionSnapshot, currentPrice decimal.Decimal, state *exit.PositionState, profile *exit.ExitProfile) *exit.ExitTrigger {
	// Check if HWM is set
	if state.HWMPrice == nil {
		log.Warn().Str("symbol", snapshot.Symbol).Msg("TRAILING_ACTIVE but HWM not set, skipping")
		return nil
	}

	// Calculate trailing stop price
	trailingPct := decimal.NewFromFloat(profile.Config.Trailing.PctTrail) // e.g., 0.04 (4%)
	trailingStopPrice := state.HWMPrice.Mul(decimal.NewFromInt(1).Sub(trailingPct))

	if currentPrice.LessThanOrEqual(trailingStopPrice) {
		log.Info().
			Str("symbol", snapshot.Symbol).
			Str("current_price", currentPrice.String()).
			Str("hwm_price", state.HWMPrice.String()).
			Str("trailing_stop_price", trailingStopPrice.String()).
			Msg("Trailing stop trigger hit")

		return &exit.ExitTrigger{
			ReasonCode: exit.ReasonTrail,
			Qty:        snapshot.Qty, // Full qty (remaining)
			OrderType:  exit.OrderTypeMKT,
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
