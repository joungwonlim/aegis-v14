package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// syncFills syncs fills from KIS since last cursor
func (s *Service) syncFills(ctx context.Context) error {
	// 1. Load last cursor
	cursor, err := s.fillRepo.GetLastCursor(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load cursor, using default")
		// Default: today's market open
		cursor = &execution.FillCursor{
			LastTS:  time.Now().Truncate(24 * time.Hour),
			LastSeq: 0,
		}
	}

	// 2. Fetch fills from KIS since cursor
	kisFills, err := s.kisAdapter.GetFills(ctx, s.accountID, cursor.LastTS)
	if err != nil {
		return fmt.Errorf("fetch fills: %w", err)
	}

	if len(kisFills) == 0 {
		return nil
	}

	log.Debug().
		Int("count", len(kisFills)).
		Str("since", cursor.LastTS.Format(time.RFC3339)).
		Msg("Fills fetched")

	// 3. Process fills
	newCursor := *cursor
	for _, kf := range kisFills {
		// ✅ Ensure order exists before upserting fill (prevents FK constraint violation)
		if err := s.ensureOrderExists(ctx, kf.OrderID); err != nil {
			log.Error().
				Err(err).
				Str("order_id", kf.OrderID).
				Str("kis_exec_id", kf.ExecID).
				Msg("Failed to ensure order exists, skipping fill")
			continue
		}

		// Upsert fill (idempotent by fill_id = kis_exec_id)
		fill := &execution.Fill{
			FillID:    uuid.New().String(), // Generate new UUID
			OrderID:   kf.OrderID,
			KisExecID: kf.ExecID, // Unique key for deduplication
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
				Msg("Failed to upsert fill")
			continue
		}

		// Update order filled_qty
		if err := s.orderRepo.UpdateFilledQty(ctx, kf.OrderID, kf.Qty); err != nil {
			log.Error().
				Err(err).
				Str("order_id", kf.OrderID).
				Msg("Failed to update filled qty")
		}

		// Derive and update order status
		if err := s.deriveAndUpdateOrderStatus(ctx, kf.OrderID); err != nil {
			log.Error().
				Err(err).
				Str("order_id", kf.OrderID).
				Msg("Failed to derive order status")
		}

		// Update cursor
		if kf.Timestamp.After(newCursor.LastTS) || (kf.Timestamp.Equal(newCursor.LastTS) && kf.Seq > newCursor.LastSeq) {
			newCursor.LastTS = kf.Timestamp
			newCursor.LastSeq = kf.Seq
		}
	}

	// 4. Save cursor
	if err := s.fillRepo.SaveCursor(ctx, newCursor); err != nil {
		log.Error().Err(err).Msg("Failed to save cursor")
		// Don't return error - fills were processed
	}

	log.Debug().
		Int("count", len(kisFills)).
		Str("new_cursor", newCursor.LastTS.Format(time.RFC3339)).
		Int("new_seq", newCursor.LastSeq).
		Msg("Fills synced")

	return nil
}

// deriveAndUpdateOrderStatus derives order status from fills and updates it
func (s *Service) deriveAndUpdateOrderStatus(ctx context.Context, orderID string) error {
	// 1. Load order
	order, err := s.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// 2. Derive status
	newStatus := s.deriveOrderStatus(order)

	// 3. Update status if changed
	if newStatus != order.Status {
		if err := s.orderRepo.UpdateOrderStatus(ctx, orderID, newStatus); err != nil {
			return fmt.Errorf("update status: %w", err)
		}

		log.Debug().
			Str("order_id", orderID).
			Str("old_status", order.Status).
			Str("new_status", newStatus).
			Msg("Order status updated")
	}

	return nil
}

// deriveOrderStatus derives order status from filled_qty and open_qty
func (s *Service) deriveOrderStatus(order *execution.Order) string {
	// 1. Check if fully filled
	if order.FilledQty >= order.Qty {
		return execution.OrderStatusFilled
	}

	// 2. Check if partially filled
	if order.FilledQty > 0 && order.OpenQty > 0 {
		return execution.OrderStatusPartial
	}

	// 3. Check if unfilled
	if order.FilledQty == 0 {
		// Use broker status
		switch order.BrokerStatus {
		case "CANCELLED":
			return execution.OrderStatusCancelled
		case "REJECTED":
			return execution.OrderStatusRejected
		case "ERROR":
			return execution.OrderStatusError
		default:
			return execution.OrderStatusSubmitted
		}
	}

	// 4. Partial fill + cancelled
	if order.FilledQty > 0 && order.BrokerStatus == "CANCELLED" {
		return execution.OrderStatusCancelledPartial
	}

	// 5. Unknown state
	return execution.OrderStatusUnknown
}

// ensureOrderExists checks if order exists in DB, if not fetches from KIS and creates it
// This prevents FK constraint violations when fills arrive before order is synced
func (s *Service) ensureOrderExists(ctx context.Context, orderID string) error {
	// 1. Check if order already exists
	_, err := s.orderRepo.GetOrder(ctx, orderID)
	if err == nil {
		return nil // Order exists
	}

	// 2. If not found, need to create it (fetch from KIS or create placeholder)
	if err != execution.ErrOrderNotFound {
		// Unexpected error
		return fmt.Errorf("check order existence: %w", err)
	}

	// 3. Order not found - try to fetch from KIS unfilled orders
	log.Info().
		Str("order_id", orderID).
		Msg("Order not in DB, attempting to fetch from KIS unfilled orders")

	unfilledOrders, err := s.kisAdapter.GetUnfilledOrders(ctx, s.accountID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch unfilled orders from KIS")
		// Fall through to create placeholder
	} else {
		// Search for this order in unfilled list
		for _, uo := range unfilledOrders {
			if uo.OrderID == orderID {
				// Found - create order
				order := &execution.Order{
					OrderID:      uo.OrderID,
					IntentID:     uuid.UUID{}, // Unknown intent
					SubmittedTS:  time.Now(),  // Approximate
					Status:       execution.OrderStatusSubmitted,
					BrokerStatus: uo.Status,
					Qty:          uo.Qty,
					OpenQty:      uo.OpenQty,
					FilledQty:    uo.FilledQty,
					Raw:          uo.Raw,
					UpdatedTS:    time.Now(),
				}

				if err := s.orderRepo.UpsertOrder(ctx, order); err != nil {
					return fmt.Errorf("upsert order from KIS: %w", err)
				}

				log.Info().
					Str("order_id", orderID).
					Msg("✅ Order synced from KIS unfilled orders")
				return nil
			}
		}
	}

	// 4. Order not in unfilled list - create placeholder (likely already filled)
	log.Info().
		Str("order_id", orderID).
		Msg("Order not in KIS unfilled list, creating placeholder (likely filled)")

	placeholderOrder := &execution.Order{
		OrderID:      orderID,
		IntentID:     uuid.UUID{}, // Unknown
		SubmittedTS:  time.Now().Add(-1 * time.Hour), // Approximate (past)
		Status:       execution.OrderStatusUnknown,
		BrokerStatus: "UNKNOWN",
		Qty:          0, // Will be updated by fills
		OpenQty:      0,
		FilledQty:    0,
		Raw:          map[string]interface{}{},
		UpdatedTS:    time.Now(),
	}

	if err := s.orderRepo.UpsertOrder(ctx, placeholderOrder); err != nil {
		return fmt.Errorf("create placeholder order: %w", err)
	}

	log.Info().
		Str("order_id", orderID).
		Msg("✅ Placeholder order created")
	return nil
}
