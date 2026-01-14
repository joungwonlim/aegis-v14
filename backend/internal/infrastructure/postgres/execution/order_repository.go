package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// OrderRepository implements execution.OrderRepository
type OrderRepository struct {
	db *pgxpool.Pool
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder creates a new order
func (r *OrderRepository) CreateOrder(ctx context.Context, order *execution.Order) error {
	rawJSON, err := json.Marshal(order.Raw)
	if err != nil {
		return fmt.Errorf("marshal raw: %w", err)
	}

	query := `
		INSERT INTO trade.orders (
			order_id, intent_id, submitted_ts, status, broker_status,
			qty, open_qty, filled_qty, raw, updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Exec(ctx, query,
		order.OrderID,
		order.IntentID,
		order.SubmittedTS,
		order.Status,
		order.BrokerStatus,
		order.Qty,
		order.OpenQty,
		order.FilledQty,
		rawJSON,
		order.UpdatedTS,
	)

	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by ID
func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (*execution.Order, error) {
	query := `
		SELECT order_id, intent_id, submitted_ts, status, broker_status,
		       qty, open_qty, filled_qty, raw, updated_ts
		FROM trade.orders
		WHERE order_id = $1
	`

	var order execution.Order
	var rawJSON []byte

	err := r.db.QueryRow(ctx, query, orderID).Scan(
		&order.OrderID,
		&order.IntentID,
		&order.SubmittedTS,
		&order.Status,
		&order.BrokerStatus,
		&order.Qty,
		&order.OpenQty,
		&order.FilledQty,
		&rawJSON,
		&order.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrOrderNotFound
		}
		return nil, fmt.Errorf("query order: %w", err)
	}

	if err := json.Unmarshal(rawJSON, &order.Raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw: %w", err)
	}

	return &order, nil
}

// GetOrderByIntentID retrieves an order by intent ID
func (r *OrderRepository) GetOrderByIntentID(ctx context.Context, intentID uuid.UUID) (*execution.Order, error) {
	query := `
		SELECT order_id, intent_id, submitted_ts, status, broker_status,
		       qty, open_qty, filled_qty, raw, updated_ts
		FROM trade.orders
		WHERE intent_id = $1
	`

	var order execution.Order
	var rawJSON []byte

	err := r.db.QueryRow(ctx, query, intentID).Scan(
		&order.OrderID,
		&order.IntentID,
		&order.SubmittedTS,
		&order.Status,
		&order.BrokerStatus,
		&order.Qty,
		&order.OpenQty,
		&order.FilledQty,
		&rawJSON,
		&order.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrOrderNotFound
		}
		return nil, fmt.Errorf("query order: %w", err)
	}

	if err := json.Unmarshal(rawJSON, &order.Raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw: %w", err)
	}

	return &order, nil
}

// UpsertOrder creates or updates an order (idempotent)
func (r *OrderRepository) UpsertOrder(ctx context.Context, order *execution.Order) error {
	rawJSON, err := json.Marshal(order.Raw)
	if err != nil {
		return fmt.Errorf("marshal raw: %w", err)
	}

	query := `
		INSERT INTO trade.orders (
			order_id, intent_id, submitted_ts, status, broker_status,
			qty, open_qty, filled_qty, raw, updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (order_id) DO UPDATE SET
			status = EXCLUDED.status,
			broker_status = EXCLUDED.broker_status,
			open_qty = EXCLUDED.open_qty,
			filled_qty = EXCLUDED.filled_qty,
			raw = EXCLUDED.raw,
			updated_ts = EXCLUDED.updated_ts
	`

	_, err = r.db.Exec(ctx, query,
		order.OrderID,
		order.IntentID,
		order.SubmittedTS,
		order.Status,
		order.BrokerStatus,
		order.Qty,
		order.OpenQty,
		order.FilledQty,
		rawJSON,
		order.UpdatedTS,
	)

	if err != nil {
		return fmt.Errorf("upsert order: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates order status
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	query := `
		UPDATE trade.orders
		SET status = $1, updated_ts = $2
		WHERE order_id = $3
	`

	result, err := r.db.Exec(ctx, query, status, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return execution.ErrOrderNotFound
	}

	return nil
}

// UpdateFilledQty increments filled_qty and updates open_qty
func (r *OrderRepository) UpdateFilledQty(ctx context.Context, orderID string, filledQty int64) error {
	query := `
		UPDATE trade.orders
		SET filled_qty = filled_qty + $1,
		    open_qty = qty - (filled_qty + $1),
		    updated_ts = $2
		WHERE order_id = $3
	`

	result, err := r.db.Exec(ctx, query, filledQty, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("update filled qty: %w", err)
	}

	if result.RowsAffected() == 0 {
		return execution.ErrOrderNotFound
	}

	return nil
}

// LoadOpenOrders loads all SUBMITTED/PARTIAL orders
func (r *OrderRepository) LoadOpenOrders(ctx context.Context) ([]*execution.Order, error) {
	return r.LoadOrdersByStatus(ctx, []string{
		execution.OrderStatusSubmitted,
		execution.OrderStatusPartial,
	})
}

// LoadOrdersByStatus loads orders by status
func (r *OrderRepository) LoadOrdersByStatus(ctx context.Context, statuses []string) ([]*execution.Order, error) {
	query := `
		SELECT order_id, intent_id, submitted_ts, status, broker_status,
		       qty, open_qty, filled_qty, raw, updated_ts
		FROM trade.orders
		WHERE status = ANY($1)
		ORDER BY submitted_ts DESC
	`

	rows, err := r.db.Query(ctx, query, statuses)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []*execution.Order
	for rows.Next() {
		var order execution.Order
		var rawJSON []byte

		err := rows.Scan(
			&order.OrderID,
			&order.IntentID,
			&order.SubmittedTS,
			&order.Status,
			&order.BrokerStatus,
			&order.Qty,
			&order.OpenQty,
			&order.FilledQty,
			&rawJSON,
			&order.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		if len(rawJSON) > 0 {
			if err := json.Unmarshal(rawJSON, &order.Raw); err != nil {
				return nil, fmt.Errorf("unmarshal raw: %w", err)
			}
		}

		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}
