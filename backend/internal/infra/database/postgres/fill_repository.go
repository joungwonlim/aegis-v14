package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// FillRepository implements execution.FillRepository
type FillRepository struct {
	pool *pgxpool.Pool
}

// NewFillRepository creates a new FillRepository
func NewFillRepository(pool *pgxpool.Pool) *FillRepository {
	return &FillRepository{
		pool: pool,
	}
}

// CreateFill creates a new fill
func (r *FillRepository) CreateFill(ctx context.Context, fill *execution.Fill) error {
	query := `
		INSERT INTO trade.fills (
			exec_id,
			order_id,
			symbol,
			qty,
			price,
			fee,
			tax,
			timestamp,
			seq,
			raw
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (exec_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query,
		fill.ExecID,
		fill.OrderID,
		fill.Symbol,
		fill.Qty,
		fill.Price,
		fill.Fee,
		fill.Tax,
		fill.Timestamp,
		fill.Seq,
		fill.Raw,
	)

	if err != nil {
		return fmt.Errorf("create fill: %w", err)
	}

	return nil
}

// GetFillsByOrderID retrieves fills for an order
func (r *FillRepository) GetFillsByOrderID(ctx context.Context, orderID string) ([]*execution.Fill, error) {
	query := `
		SELECT
			exec_id,
			order_id,
			symbol,
			qty,
			price,
			fee,
			tax,
			timestamp,
			seq,
			raw
		FROM trade.fills
		WHERE order_id = $1
		ORDER BY timestamp ASC, seq ASC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query fills by order: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		fill := &execution.Fill{}
		err := rows.Scan(
			&fill.ExecID,
			&fill.OrderID,
			&fill.Symbol,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Timestamp,
			&fill.Seq,
			&fill.Raw,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}
		fills = append(fills, fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}

// GetRecentFills retrieves recent fills (for API)
func (r *FillRepository) GetRecentFills(ctx context.Context, limit int) ([]*execution.Fill, error) {
	query := `
		SELECT
			exec_id,
			order_id,
			symbol,
			qty,
			price,
			fee,
			tax,
			timestamp,
			seq,
			raw
		FROM trade.fills
		ORDER BY timestamp DESC, seq DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent fills: %w", err)
	}
	defer rows.Close()

	var fills []*execution.Fill
	for rows.Next() {
		fill := &execution.Fill{}
		err := rows.Scan(
			&fill.ExecID,
			&fill.OrderID,
			&fill.Symbol,
			&fill.Qty,
			&fill.Price,
			&fill.Fee,
			&fill.Tax,
			&fill.Timestamp,
			&fill.Seq,
			&fill.Raw,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fill: %w", err)
		}
		fills = append(fills, fill)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return fills, nil
}
