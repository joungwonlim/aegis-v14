package exit

import (
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
// 3. TP3 (take profit 3)
// 4. TP2 (take profit 2)
// 5. TP1 (take profit 1)
// 6. TRAIL (trailing stop, only in TRAILING_ACTIVE phase)
// 7. TIME (time-based exit)
//
// Control Mode Filtering:
// - PAUSE_PROFIT: Only SL triggers (block TP/TRAIL)
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

	// If PAUSE_PROFIT mode, block TP/TRAIL
	if controlMode == exit.ControlModePauseProfit {
		log.Debug().Str("symbol", snapshot.Symbol).Msg("PAUSE_PROFIT mode, blocking TP/TRAIL")
		return nil
	}

	// Priority 3: TP3
	// TODO: Implement TP3 evaluation

	// Priority 4: TP2
	// TODO: Implement TP2 evaluation

	// Priority 5: TP1
	if trigger := s.evaluateTP1(snapshot, pnlPct, profile); trigger != nil {
		return trigger
	}

	// Priority 6: TRAIL (only in TRAILING_ACTIVE phase)
	// TODO: Implement TRAIL evaluation

	// Priority 7: TIME
	// TODO: Implement TIME evaluation

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
