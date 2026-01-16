package execution

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// reconcileOrders reconciles order states with KIS
func (s *Service) reconcileOrders(ctx context.Context) error {
	// 1. Load open orders (SUBMITTED, PARTIAL)
	openOrders, err := s.orderRepo.LoadOrdersByStatus(ctx, []string{
		execution.OrderStatusSubmitted,
		execution.OrderStatusPartial,
	})
	if err != nil {
		return fmt.Errorf("load open orders: %w", err)
	}

	if len(openOrders) == 0 {
		return nil
	}

	log.Debug().Int("count", len(openOrders)).Msg("Reconciling open orders")

	// 2. Fetch unfilled orders from KIS
	unfilledOrders, err := s.kisAdapter.GetUnfilledOrders(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("fetch unfilled orders: %w", err)
	}

	// Build unfilled map for quick lookup
	unfilledMap := make(map[string]*execution.KISUnfilledOrder)
	for _, uo := range unfilledOrders {
		unfilledMap[uo.OrderID] = uo
	}

	// 3. Reconcile each open order
	for _, order := range openOrders {
		unfilled, existsInKIS := unfilledMap[order.OrderID]

		if existsInKIS {
			// Order still unfilled in KIS - update status
			if err := s.reconcileUnfilledOrder(ctx, order, unfilled); err != nil {
				log.Error().
					Err(err).
					Str("order_id", order.OrderID).
					Msg("Failed to reconcile unfilled order")
			}
		} else {
			// Order not in unfilled list - might be filled or cancelled
			if err := s.reconcileFilledOrCancelledOrder(ctx, order); err != nil {
				log.Error().
					Err(err).
					Str("order_id", order.OrderID).
					Msg("Failed to reconcile filled/cancelled order")
			}
		}
	}

	return nil
}

// reconcileUnfilledOrder reconciles an unfilled order with KIS data
func (s *Service) reconcileUnfilledOrder(ctx context.Context, order *execution.Order, unfilled *execution.KISUnfilledOrder) error {
	// Update open_qty and filled_qty if changed
	if order.OpenQty != unfilled.OpenQty || order.FilledQty != unfilled.FilledQty {
		log.Debug().
			Str("order_id", order.OrderID).
			Int64("old_open_qty", order.OpenQty).
			Int64("new_open_qty", unfilled.OpenQty).
			Int64("old_filled_qty", order.FilledQty).
			Int64("new_filled_qty", unfilled.FilledQty).
			Msg("Order quantities changed during reconciliation")

		// Update order (need to implement UpdateOrder in repository)
		order.OpenQty = unfilled.OpenQty
		order.FilledQty = unfilled.FilledQty
		order.BrokerStatus = unfilled.Status

		if err := s.orderRepo.UpsertOrder(ctx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		// Derive and update status
		if err := s.deriveAndUpdateOrderStatus(ctx, order.OrderID); err != nil {
			return fmt.Errorf("derive status: %w", err)
		}
	}

	return nil
}

// reconcileFilledOrCancelledOrder reconciles an order that's not in unfilled list
func (s *Service) reconcileFilledOrCancelledOrder(ctx context.Context, order *execution.Order) error {
	// Query fills for this order from KIS
	fills, err := s.kisAdapter.GetFillsForOrder(ctx, order.OrderID)
	if err != nil {
		log.Warn().
			Err(err).
			Str("order_id", order.OrderID).
			Msg("Failed to query fills for order")
		return nil // Don't fail - might be transient
	}

	if len(fills) > 0 {
		// Order was filled - upsert fills
		log.Debug().
			Str("order_id", order.OrderID).
			Int("fill_count", len(fills)).
			Msg("Found fills for reconciled order")

		for _, kf := range fills {
			// ✅ Ensure order exists (should exist, but double-check)
			if err := s.ensureOrderExists(ctx, kf.OrderID); err != nil {
				log.Error().
					Err(err).
					Str("order_id", kf.OrderID).
					Str("kis_exec_id", kf.ExecID).
					Msg("Failed to ensure order exists during reconciliation")
				continue
			}

			fill := &execution.Fill{
				FillID:    kf.ExecID, // Use exec_id as fill_id for deduplication
				OrderID:   kf.OrderID,
				KisExecID: kf.ExecID,
				TS:        kf.Timestamp,
				Qty:       kf.Qty,
				Price:     kf.Price,
				Fee:       kf.Fee,
				Tax:       kf.Tax,
				Seq:       kf.Seq,
			}

			if err := s.fillRepo.UpsertFill(ctx, fill); err != nil {
				log.Error().
					Err(err).
					Str("order_id", kf.OrderID).
					Str("kis_exec_id", kf.ExecID).
					Msg("Failed to upsert fill during reconciliation")
			}

			// Update filled_qty
			if err := s.orderRepo.UpdateFilledQty(ctx, kf.OrderID, kf.Qty); err != nil {
				log.Error().
					Err(err).
					Str("order_id", kf.OrderID).
					Msg("Failed to update filled qty during reconciliation")
			}
		}

		// Derive and update status
		if err := s.deriveAndUpdateOrderStatus(ctx, order.OrderID); err != nil {
			return fmt.Errorf("derive status: %w", err)
		}
	} else {
		// No fills - order might be cancelled
		log.Warn().
			Str("order_id", order.OrderID).
			Msg("Order not in unfilled list and no fills found - marking as UNKNOWN")

		if err := s.orderRepo.UpdateOrderStatus(ctx, order.OrderID, execution.OrderStatusUnknown); err != nil {
			return fmt.Errorf("update status: %w", err)
		}
	}

	return nil
}
