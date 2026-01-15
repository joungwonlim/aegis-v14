package exit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// ReconcileIntents reconciles Intent states with actual Holdings and Fills
// This prevents duplicate/stale intents and ensures data consistency
//
// Reconciliation checks:
// 1. Duplicate intents (same position + reason) → cancel older ones
// 2. Order status sync → update Intent status based on Order/Fill status
// 3. Loss recovery → cancel intents if conditions no longer valid
// 4. Position qty sync → update Position.qty based on actual fills
func (s *Service) ReconcileIntents(ctx context.Context) error {
	log.Info().Msg("🔄 Starting Intent Reconciliation")

	// 1. Find and cancel duplicate intents
	if err := s.cancelDuplicateIntents(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to cancel duplicate intents (non-fatal)")
	}

	// 2. Sync Intent status with Order/Fill status
	// TODO: Requires OrderRepository integration
	// if err := s.syncIntentStatusWithOrders(ctx); err != nil {
	// 	log.Warn().Err(err).Msg("Failed to sync intent status (non-fatal)")
	// }

	// 3. Cancel intents if loss recovered (conditions no longer valid)
	if err := s.cancelRecoveredIntents(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to cancel recovered intents (non-fatal)")
	}

	// 4. Sync Position qty with actual holdings
	// TODO: Requires HoldingRepository integration
	// if err := s.syncPositionQtyWithHoldings(ctx); err != nil {
	// 	log.Warn().Err(err).Msg("Failed to sync position qty (non-fatal)")
	// }

	log.Info().Msg("✅ Intent Reconciliation completed")
	return nil
}

// cancelDuplicateIntents finds and cancels duplicate intents for the same position+reason
func (s *Service) cancelDuplicateIntents(ctx context.Context) error {
	// Get recent intents (last 500)
	intents, err := s.intentRepo.GetRecentIntents(ctx, 500)
	if err != nil {
		return err
	}

	// Group by position_id + reason_code
	type intentKey struct {
		positionID uuid.UUID
		reasonCode string
	}
	intentGroups := make(map[intentKey][]struct {
		intentID  uuid.UUID
		createdTS time.Time
	})

	for _, intent := range intents {
		// Only check active statuses
		if intent.Status != "NEW" && intent.Status != "PENDING_APPROVAL" && intent.Status != "SUBMITTED" {
			continue
		}

		key := intentKey{positionID: intent.PositionID, reasonCode: intent.ReasonCode}
		intentGroups[key] = append(intentGroups[key], struct {
			intentID  uuid.UUID
			createdTS time.Time
		}{intentID: intent.IntentID, createdTS: intent.CreatedTS})
	}

	// Cancel older duplicates (keep the most recent one)
	cancelledCount := 0
	for key, intentList := range intentGroups {
		if len(intentList) <= 1 {
			continue
		}

		// Cancel all except the most recent (last one in the list)
		for i := 0; i < len(intentList)-1; i++ {
			err := s.intentRepo.UpdateIntentStatus(ctx, intentList[i].intentID, "CANCELLED")
			if err != nil {
				log.Warn().Err(err).Str("intent_id", intentList[i].intentID.String()).Msg("Failed to cancel duplicate intent")
				continue
			}

			log.Warn().
				Str("position_id", key.positionID.String()).
				Str("reason_code", key.reasonCode).
				Str("cancelled_intent_id", intentList[i].intentID.String()).
				Msg("⚠️ Cancelled duplicate intent")
			cancelledCount++
		}
	}

	if cancelledCount > 0 {
		log.Info().Int("count", cancelledCount).Msg("Cancelled duplicate intents during reconciliation")
	}

	return nil
}

// syncIntentStatusWithOrders synchronizes Intent status with Order/Fill status
// - If Order is FILLED → update Intent to FILLED
// - If Order is CANCELLED → update Intent to CANCELLED
// - If Order is FAILED → update Intent to FAILED
// TODO: Requires OrderRepository integration
/*
func (s *Service) syncIntentStatusWithOrders(ctx context.Context) error {
	// Get active intents (SUBMITTED status)
	intents, err := s.intentRepo.GetRecentIntents(ctx, 500)
	if err != nil {
		return err
	}

	syncCount := 0
	for _, intent := range intents {
		if intent.Status != "SUBMITTED" {
			continue
		}

		// Check if there's an associated order
		orders, err := s.orderRepo.GetOrdersByIntentID(ctx, intent.IntentID)
		if err != nil {
			log.Warn().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to get orders for intent")
			continue
		}

		if len(orders) == 0 {
			// No order yet - normal case (Execution Service will create it)
			continue
		}

		// Check order status (use latest order)
		order := orders[len(orders)-1]

		var newStatus string
		switch order.Status {
		case "FILLED":
			newStatus = "FILLED"
		case "CANCELLED":
			newStatus = "CANCELLED"
		case "FAILED":
			newStatus = "FAILED"
		default:
			// OPEN, PARTIAL, etc. - keep as SUBMITTED
			continue
		}

		// Update Intent status
		err = s.intentRepo.UpdateIntentStatus(ctx, intent.IntentID, newStatus)
		if err != nil {
			log.Warn().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to update intent status")
			continue
		}

		log.Info().
			Str("symbol", intent.Symbol).
			Str("intent_id", intent.IntentID.String()).
			Str("old_status", "SUBMITTED").
			Str("new_status", newStatus).
			Msg("Intent status synchronized with order")
		syncCount++
	}

	if syncCount > 0 {
		log.Info().Int("count", syncCount).Msg("Synchronized intent statuses with orders")
	}

	return nil
}
*/

// cancelRecoveredIntents cancels intents if loss has recovered
// Example: -5% SL1 intent created, but now at -2% → cancel intent
func (s *Service) cancelRecoveredIntents(ctx context.Context) error {
	// Get active intents (NEW, PENDING_APPROVAL, SUBMITTED)
	intents, err := s.intentRepo.GetRecentIntents(ctx, 500)
	if err != nil {
		return err
	}

	log.Info().Int("total_intents", len(intents)).Msg("Checking intents for recovery cancellation")

	cancelCount := 0
	checkedCount := 0
	for _, intent := range intents {
		// Only check active intents
		if intent.Status != "NEW" && intent.Status != "PENDING_APPROVAL" && intent.Status != "SUBMITTED" {
			continue
		}

		checkedCount++

		// Get position
		pos, err := s.posRepo.GetPosition(ctx, intent.PositionID)
		if err != nil {
			log.Warn().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to get position")
			continue
		}

		// Get current price
		bestPrice, err := s.priceSync.GetBestPrice(ctx, intent.Symbol)
		if err != nil {
			log.Warn().Err(err).Str("symbol", intent.Symbol).Msg("Failed to get best price")
			continue
		}
		if bestPrice.IsStale {
			log.Debug().Str("symbol", intent.Symbol).Msg("Price is stale, skipping recovery check")
			continue
		}

		// Calculate current P&L%
		var currentPriceInt int64
		if bestPrice.BidPrice != nil {
			currentPriceInt = *bestPrice.BidPrice
		} else {
			currentPriceInt = bestPrice.BestPrice
		}
		currentPrice := decimal.NewFromInt(currentPriceInt)
		pnlPct := currentPrice.Sub(pos.AvgPrice).Div(pos.AvgPrice).Mul(decimal.NewFromInt(100))

		// Check if intent should be cancelled based on reason_code
		shouldCancel := false
		reason := ""

		switch intent.ReasonCode {
		case "SL1", "SL2":
			// SL triggers: cancel if loss recovered (e.g., from -5% to -2%)
			// Use a recovery threshold (e.g., -3% for SL1, -4% for SL2)
			recoveryThreshold := -3.0
			if intent.ReasonCode == "SL2" {
				recoveryThreshold = -4.0
			}

			if pnlPct.GreaterThan(decimal.NewFromFloat(recoveryThreshold)) {
				shouldCancel = true
				reason = "loss recovered"
			}

		case "TP1", "TP2", "TP3":
			// TP triggers: cancel if profit decreased significantly
			// For now, keep TP intents active (user may still want to take profit)
			// Future: can add logic to cancel if profit < threshold

		case "TRAIL":
			// Trailing stop: cancel if price went too high again
			// For now, keep TRAIL intents active
		}

		if shouldCancel {
			err = s.intentRepo.UpdateIntentStatus(ctx, intent.IntentID, "CANCELLED")
			if err != nil {
				log.Warn().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to cancel recovered intent")
				continue
			}

			log.Info().
				Str("symbol", intent.Symbol).
				Str("reason_code", intent.ReasonCode).
				Str("pnl_pct", pnlPct.StringFixed(2)).
				Str("reason", reason).
				Msg("Intent cancelled due to recovery")
			cancelCount++
		}
	}

	log.Info().
		Int("checked", checkedCount).
		Int("cancelled", cancelCount).
		Msg("Recovery cancellation check completed")

	return nil
}

// syncPositionQtyWithHoldings synchronizes Position.qty with actual KIS holdings
// This handles partial fills (e.g., 1000 → 500 after partial exit)
// TODO: Requires HoldingRepository integration
/*
func (s *Service) syncPositionQtyWithHoldings(ctx context.Context) error {
	// Get all CLOSING positions (these may have been partially filled)
	positions, err := s.posRepo.GetPositionsByStatus(ctx, "CLOSING")
	if err != nil {
		return err
	}

	syncCount := 0
	for _, pos := range positions {
		// Get actual holding from KIS
		holding, err := s.holdingRepo.GetHoldingBySymbol(ctx, pos.AccountID, pos.Symbol)
		if err != nil {
			log.Warn().Err(err).Str("symbol", pos.Symbol).Msg("Failed to get holding")
			continue
		}

		if holding == nil {
			// Position fully closed - update to CLOSED
			err = s.posRepo.UpdateStatus(ctx, pos.PositionID, "CLOSED", pos.Version)
			if err != nil {
				log.Warn().Err(err).Str("symbol", pos.Symbol).Msg("Failed to update position to CLOSED")
				continue
			}

			log.Info().
				Str("symbol", pos.Symbol).
				Str("position_id", pos.PositionID.String()).
				Msg("Position fully closed - updated to CLOSED")
			syncCount++
			continue
		}

		// Check if qty differs
		if holding.Qty != pos.Qty {
			// Update Position.qty to match actual holding
			err = s.posRepo.UpdateQty(ctx, pos.PositionID, holding.Qty)
			if err != nil {
				log.Warn().Err(err).Str("symbol", pos.Symbol).Msg("Failed to update position qty")
				continue
			}

			log.Info().
				Str("symbol", pos.Symbol).
				Int64("old_qty", pos.Qty).
				Int64("new_qty", holding.Qty).
				Msg("Position qty synchronized with holding")
			syncCount++
		}
	}

	if syncCount > 0 {
		log.Info().Int("count", syncCount).Msg("Synchronized position quantities")
	}

	return nil
}
*/
