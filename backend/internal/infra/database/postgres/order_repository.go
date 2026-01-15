package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// OrderRepository implements execution.OrderRepository
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool: pool,
	}
}

// CreateOrder creates a new order
func (r *OrderRepository) CreateOrder(ctx context.Context, order *execution.Order) error {
	query := `
		INSERT INTO trade.orders (
			order_id,
			intent_id,
			submitted_ts,
			status,
			broker_status,
			qty,
			open_qty,
			filled_qty,
			raw,
			updated_ts
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		order.OrderID,
		order.IntentID,
		order.SubmittedTS,
		order.Status,
		order.BrokerStatus,
		order.Qty,
		order.OpenQty,
		order.FilledQty,
		order.Raw,
		order.UpdatedTS,
	)

	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

// GetOrderByID retrieves an order by ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID string) (*execution.Order, error) {
	query := `
		SELECT
			order_id,
			intent_id,
			submitted_ts,
			status,
			broker_status,
			qty,
			open_qty,
			filled_qty,
			raw,
			updated_ts
		FROM trade.orders
		WHERE order_id = $1
	`

	order := &execution.Order{}
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&order.OrderID,
		&order.IntentID,
		&order.SubmittedTS,
		&order.Status,
		&order.BrokerStatus,
		&order.Qty,
		&order.OpenQty,
		&order.FilledQty,
		&order.Raw,
		&order.UpdatedTS,
	)

	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}

	return order, nil
}

// GetOrderByIntentID retrieves an order by intent ID
func (r *OrderRepository) GetOrderByIntentID(ctx context.Context, intentID uuid.UUID) (*execution.Order, error) {
	query := `
		SELECT
			order_id,
			intent_id,
			submitted_ts,
			status,
			broker_status,
			qty,
			open_qty,
			filled_qty,
			raw,
			updated_ts
		FROM trade.orders
		WHERE intent_id = $1
	`

	order := &execution.Order{}
	err := r.pool.QueryRow(ctx, query, intentID).Scan(
		&order.OrderID,
		&order.IntentID,
		&order.SubmittedTS,
		&order.Status,
		&order.BrokerStatus,
		&order.Qty,
		&order.OpenQty,
		&order.FilledQty,
		&order.Raw,
		&order.UpdatedTS,
	)

	if err != nil {
		return nil, fmt.Errorf("get order by intent id: %w", err)
	}

	return order, nil
}

// GetRecentOrders retrieves recent orders (for API)
func (r *OrderRepository) GetRecentOrders(ctx context.Context, limit int) ([]*execution.Order, error) {
	query := `
		SELECT
			order_id,
			intent_id,
			submitted_ts,
			status,
			broker_status,
			qty,
			open_qty,
			filled_qty,
			raw,
			updated_ts
		FROM trade.orders
		ORDER BY submitted_ts DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent orders: %w", err)
	}
	defer rows.Close()

	var orders []*execution.Order
	for rows.Next() {
		order := &execution.Order{}
		err := rows.Scan(
			&order.OrderID,
			&order.IntentID,
			&order.SubmittedTS,
			&order.Status,
			&order.BrokerStatus,
			&order.Qty,
			&order.OpenQty,
			&order.FilledQty,
			&order.Raw,
			&order.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

// UpdateOrder updates an order
func (r *OrderRepository) UpdateOrder(ctx context.Context, order *execution.Order) error {
	query := `
		UPDATE trade.orders
		SET
			status = $1,
			broker_status = $2,
			open_qty = $3,
			filled_qty = $4,
			raw = $5,
			updated_ts = $6
		WHERE order_id = $7
	`

	_, err := r.pool.Exec(ctx, query,
		order.Status,
		order.BrokerStatus,
		order.OpenQty,
		order.FilledQty,
		order.Raw,
		order.UpdatedTS,
		order.OrderID,
	)

	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by ID (alias for GetOrderByID to match interface)
func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (*execution.Order, error) {
	return r.GetOrderByID(ctx, orderID)
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
		SELECT
			order_id,
			intent_id,
			submitted_ts,
			status,
			broker_status,
			qty,
			open_qty,
			filled_qty,
			raw,
			updated_ts
		FROM trade.orders
		WHERE status = ANY($1)
		ORDER BY submitted_ts DESC
	`

	rows, err := r.pool.Query(ctx, query, statuses)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []*execution.Order
	for rows.Next() {
		order := &execution.Order{}
		err := rows.Scan(
			&order.OrderID,
			&order.IntentID,
			&order.SubmittedTS,
			&order.Status,
			&order.BrokerStatus,
			&order.Qty,
			&order.OpenQty,
			&order.FilledQty,
			&order.Raw,
			&order.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

// UpsertOrder creates or updates an order (idempotent)
func (r *OrderRepository) UpsertOrder(ctx context.Context, order *execution.Order) error {
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

	_, err := r.pool.Exec(ctx, query,
		order.OrderID,
		order.IntentID,
		order.SubmittedTS,
		order.Status,
		order.BrokerStatus,
		order.Qty,
		order.OpenQty,
		order.FilledQty,
		order.Raw,
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
		SET status = $1, updated_ts = NOW()
		WHERE order_id = $2
	`

	result, err := r.pool.Exec(ctx, query, status, orderID)
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
		    updated_ts = NOW()
		WHERE order_id = $2
	`

	result, err := r.pool.Exec(ctx, query, filledQty, orderID)
	if err != nil {
		return fmt.Errorf("update filled qty: %w", err)
	}

	if result.RowsAffected() == 0 {
		return execution.ErrOrderNotFound
	}

	return nil
}

// GetRecentOrders retrieves recent orders (for monitoring)
