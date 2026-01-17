package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// 한국 시간대
var kst = time.FixedZone("KST", 9*60*60)

// isMarketOpen checks if Korean stock market is open
// Market hours: 09:00 - 15:30 KST (weekdays only)
func isMarketOpen() bool {
	now := time.Now().In(kst)

	// Check weekday (Monday = 1, Sunday = 0)
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Check time (09:00 - 15:30)
	hour, min, _ := now.Clock()
	timeMinutes := hour*60 + min

	marketOpen := 9*60 + 0   // 09:00
	marketClose := 15*60 + 30 // 15:30

	return timeMinutes >= marketOpen && timeMinutes < marketClose
}

// processNewIntents processes all NEW intents
func (s *Service) processNewIntents(ctx context.Context) error {
	// 0. Check market hours - skip processing if market is closed
	if !isMarketOpen() {
		// Market closed - keep intents in NEW status, will be processed when market opens
		return nil
	}

	// 1. Load NEW intents
	intents, err := s.intentRepo.LoadNewIntents(ctx)
	if err != nil {
		return fmt.Errorf("load new intents: %w", err)
	}

	if len(intents) == 0 {
		return nil
	}

	log.Debug().Int("count", len(intents)).Msg("Processing new intents")

	// 2. Process each intent
	for _, intent := range intents {
		if err := s.processIntent(ctx, intent); err != nil {
			log.Error().
				Err(err).
				Str("intent_id", intent.IntentID.String()).
				Str("symbol", intent.Symbol).
				Msg("Intent processing failed")
			// Continue with other intents
		}
	}

	return nil
}

// processIntent processes a single intent
func (s *Service) processIntent(ctx context.Context, intent *exit.OrderIntent) error {
	// 1. Check for duplicate (order already exists for this intent)
	existingOrder, err := s.orderRepo.GetOrderByIntentID(ctx, intent.IntentID)
	if err != nil && err != execution.ErrOrderNotFound {
		return fmt.Errorf("check duplicate: %w", err)
	}

	if existingOrder != nil {
		// Duplicate - order already exists
		log.Warn().
			Str("intent_id", intent.IntentID.String()).
			Str("order_id", existingOrder.OrderID).
			Msg("Intent already submitted (duplicate)")

		// Update intent status to DUPLICATE
		if err := s.intentRepo.UpdateIntentStatus(ctx, intent.IntentID, execution.IntentStatusDuplicate); err != nil {
			return fmt.Errorf("update intent status: %w", err)
		}

		return nil
	}

	// 2. Submit order to KIS
	orderID, err := s.submitOrder(ctx, intent)
	if err != nil {
		// Submit failed - update intent status to FAILED
		if err := s.intentRepo.UpdateIntentStatus(ctx, intent.IntentID, execution.IntentStatusFailed); err != nil {
			log.Error().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to update intent status")
		}
		return fmt.Errorf("submit order: %w", err)
	}

	// 3. Update intent status to SUBMITTED
	if err := s.intentRepo.UpdateIntentStatus(ctx, intent.IntentID, execution.IntentStatusSubmitted); err != nil {
		log.Error().Err(err).Str("intent_id", intent.IntentID.String()).Msg("Failed to update intent status")
	}

	log.Info().
		Str("intent_id", intent.IntentID.String()).
		Str("order_id", orderID).
		Str("symbol", intent.Symbol).
		Str("type", intent.IntentType).
		Int64("qty", intent.Qty).
		Msg("Intent submitted successfully")

	return nil
}

// submitOrder submits an order to KIS and creates order row
func (s *Service) submitOrder(ctx context.Context, intent *exit.OrderIntent) (string, error) {
	// 1. Build KIS request
	req := execution.KISOrderRequest{
		AccountID:  s.accountID,
		Symbol:     intent.Symbol,
		Side:       s.intentTypeToSide(intent.IntentType),
		OrderType:  intent.OrderType,
		Qty:        intent.Qty,
		LimitPrice: intent.LimitPrice,
	}

	// 2. Submit to KIS
	resp, err := s.kisAdapter.SubmitOrder(ctx, req)
	if err != nil {
		return "", fmt.Errorf("KIS submit: %w", err)
	}

	// 3. Create order row
	order := &execution.Order{
		OrderID:      resp.OrderID,
		IntentID:     intent.IntentID,
		SubmittedTS:  resp.Timestamp,
		Status:       execution.OrderStatusSubmitted,
		BrokerStatus: "SUBMITTED",
		Qty:          intent.Qty,
		OpenQty:      intent.Qty,
		FilledQty:    0,
		Raw:          resp.Raw,
		UpdatedTS:    resp.Timestamp,
	}

	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		// Order creation failed - this is a critical issue
		// The order was submitted to KIS but we can't track it
		log.Error().
			Err(err).
			Str("order_id", resp.OrderID).
			Str("intent_id", intent.IntentID.String()).
			Msg("Failed to create order row (orphan order!)")
		return resp.OrderID, fmt.Errorf("create order: %w", err)
	}

	return resp.OrderID, nil
}

// intentTypeToSide converts intent type to KIS side
func (s *Service) intentTypeToSide(intentType string) string {
	switch intentType {
	case execution.IntentTypeEntry:
		return execution.SideBuy
	case exit.IntentTypeExitPartial, exit.IntentTypeExitFull:
		return execution.SideSell
	default:
		log.Warn().Str("intent_type", intentType).Msg("Unknown intent type, defaulting to SELL")
		return execution.SideSell
	}
}
