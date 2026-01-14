package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// Bootstrap restores state from KIS on startup
func (s *Service) Bootstrap(ctx context.Context) error {
	log.Info().Msg("Starting bootstrap from KIS...")

	// 1. Holdings Sync (최우선: 최종 진실)
	if err := s.syncHoldings(ctx); err != nil {
		return fmt.Errorf("bootstrap holdings failed: %w", err)
	}
	log.Info().Msg("Holdings synced")

	// 2. Unfilled Orders Sync
	if err := s.syncUnfilledOrders(ctx); err != nil {
		return fmt.Errorf("bootstrap unfilled failed: %w", err)
	}
	log.Info().Msg("Unfilled orders synced")

	// 3. Fills Sync (since today's market open)
	if err := s.bootstrapFills(ctx); err != nil {
		return fmt.Errorf("bootstrap fills failed: %w", err)
	}
	log.Info().Msg("Fills synced")

	// 4. Recompute order states
	if err := s.recomputeOrderStates(ctx); err != nil {
		return fmt.Errorf("recompute order states failed: %w", err)
	}
	log.Info().Msg("Order states recomputed")

	log.Info().Msg("Bootstrap completed successfully")
	return nil
}

// syncUnfilledOrders syncs unfilled orders from KIS during bootstrap
func (s *Service) syncUnfilledOrders(ctx context.Context) error {
	// Fetch unfilled orders from KIS
	unfilledOrders, err := s.kisAdapter.GetUnfilledOrders(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("fetch unfilled: %w", err)
	}

	log.Debug().Int("count", len(unfilledOrders)).Msg("Unfilled orders fetched")

	// Upsert orders
	for _, uo := range unfilledOrders {
		order := &execution.Order{
			OrderID:      uo.OrderID,
			IntentID:     s.findIntentID(ctx, uo.OrderID), // Try to find intent ID
			SubmittedTS:  time.Now(),                      // Approximate (KIS doesn't provide)
			Status:       execution.OrderStatusSubmitted,
			BrokerStatus: uo.Status,
			Qty:          uo.Qty,
			OpenQty:      uo.OpenQty,
			FilledQty:    uo.FilledQty,
			Raw:          uo.Raw,
			UpdatedTS:    time.Now(),
		}

		if err := s.orderRepo.UpsertOrder(ctx, order); err != nil {
			log.Error().
				Err(err).
				Str("order_id", uo.OrderID).
				Msg("Failed to upsert unfilled order")
		}
	}

	return nil
}

// bootstrapFills syncs fills since today's market open
func (s *Service) bootstrapFills(ctx context.Context) error {
	// Reset cursor to today's start
	todayStart := time.Now().Truncate(24 * time.Hour)
	cursor := execution.FillCursor{
		LastTS:  todayStart,
		LastSeq: 0,
	}

	if err := s.fillRepo.SaveCursor(ctx, cursor); err != nil {
		log.Warn().Err(err).Msg("Failed to reset cursor")
	}

	// Sync fills using normal sync logic
	return s.syncFills(ctx)
}

// recomputeOrderStates recomputes order statuses from fills
func (s *Service) recomputeOrderStates(ctx context.Context) error {
	// Load all open orders
	openOrders, err := s.orderRepo.LoadOrdersByStatus(ctx, []string{
		execution.OrderStatusSubmitted,
		execution.OrderStatusPartial,
	})
	if err != nil {
		return fmt.Errorf("load open orders: %w", err)
	}

	log.Debug().Int("count", len(openOrders)).Msg("Recomputing order states")

	for _, order := range openOrders {
		if err := s.deriveAndUpdateOrderStatus(ctx, order.OrderID); err != nil {
			log.Error().
				Err(err).
				Str("order_id", order.OrderID).
				Msg("Failed to derive order status")
		}
	}

	return nil
}

// findIntentID attempts to find intent ID for an order (reverse lookup)
func (s *Service) findIntentID(ctx context.Context, orderID string) uuid.UUID {
	// Try to load order first
	order, err := s.orderRepo.GetOrder(ctx, orderID)
	if err == nil {
		return order.IntentID
	}

	// If order not found, we can't reverse lookup (need to implement intent query)
	// For now, return zero UUID
	return uuid.UUID{}
}
