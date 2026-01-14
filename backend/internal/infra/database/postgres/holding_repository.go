package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// HoldingRepository implements execution.HoldingRepository
type HoldingRepository struct {
	pool *pgxpool.Pool
}

// NewHoldingRepository creates a new HoldingRepository
func NewHoldingRepository(pool *pgxpool.Pool) *HoldingRepository {
	return &HoldingRepository{
		pool: pool,
	}
}

// GetAllHoldings retrieves all holdings
func (r *HoldingRepository) GetAllHoldings(ctx context.Context) ([]*execution.Holding, error) {
	query := `
		SELECT
			h.account_id,
			h.symbol,
			h.qty,
			h.avg_price,
			h.current_price,
			(h.current_price - h.avg_price) * h.qty as pnl,
			((h.current_price - h.avg_price) / NULLIF(h.avg_price, 0) * 100) as pnl_pct,
			h.updated_ts,
			COALESCE(p.exit_mode, 'ENABLED') as exit_mode,
			h.raw
		FROM trade.holdings h
		LEFT JOIN trade.positions p ON h.account_id = p.account_id AND h.symbol = p.symbol
		WHERE h.qty > 0
		ORDER BY h.symbol ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query holdings: %w", err)
	}
	defer rows.Close()

	var holdings []*execution.Holding
	for rows.Next() {
		h := &execution.Holding{}
		err := rows.Scan(
			&h.AccountID,
			&h.Symbol,
			&h.Qty,
			&h.AvgPrice,
			&h.CurrentPrice,
			&h.Pnl,
			&h.PnlPct,
			&h.UpdatedTS,
			&h.ExitMode,
			&h.Raw,
		)
		if err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return holdings, nil
}

// GetHolding retrieves a specific holding
func (r *HoldingRepository) GetHolding(ctx context.Context, accountID, symbol string) (*execution.Holding, error) {
	query := `
		SELECT
			h.account_id,
			h.symbol,
			h.qty,
			h.avg_price,
			h.current_price,
			(h.current_price - h.avg_price) * h.qty as pnl,
			((h.current_price - h.avg_price) / NULLIF(h.avg_price, 0) * 100) as pnl_pct,
			h.updated_ts,
			COALESCE(p.exit_mode, 'ENABLED') as exit_mode,
			h.raw
		FROM trade.holdings h
		LEFT JOIN trade.positions p ON h.account_id = p.account_id AND h.symbol = p.symbol
		WHERE h.account_id = $1 AND h.symbol = $2
	`

	h := &execution.Holding{}
	err := r.pool.QueryRow(ctx, query, accountID, symbol).Scan(
		&h.AccountID,
		&h.Symbol,
		&h.Qty,
		&h.AvgPrice,
		&h.CurrentPrice,
		&h.Pnl,
		&h.PnlPct,
		&h.UpdatedTS,
		&h.ExitMode,
		&h.Raw,
	)

	if err != nil {
		return nil, fmt.Errorf("get holding: %w", err)
	}

	return h, nil
}

// UpsertHolding creates or updates a holding
func (r *HoldingRepository) UpsertHolding(ctx context.Context, holding *execution.Holding) error {
	query := `
		INSERT INTO trade.holdings (
			account_id, symbol, qty, avg_price,
			current_price, pnl, pnl_pct, updated_ts, raw
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (account_id, symbol)
		DO UPDATE SET
			qty = EXCLUDED.qty,
			avg_price = EXCLUDED.avg_price,
			current_price = EXCLUDED.current_price,
			pnl = EXCLUDED.pnl,
			pnl_pct = EXCLUDED.pnl_pct,
			updated_ts = EXCLUDED.updated_ts,
			raw = EXCLUDED.raw
	`

	_, err := r.pool.Exec(ctx, query,
		holding.AccountID,
		holding.Symbol,
		holding.Qty,
		holding.AvgPrice,
		holding.CurrentPrice,
		holding.Pnl,
		holding.PnlPct,
		holding.UpdatedTS,
		holding.Raw,
	)

	if err != nil {
		return fmt.Errorf("upsert holding: %w", err)
	}

	return nil
}

// DeleteHolding removes a holding
func (r *HoldingRepository) DeleteHolding(ctx context.Context, accountID, symbol string) error {
	query := `DELETE FROM trade.holdings WHERE account_id = $1 AND symbol = $2`

	_, err := r.pool.Exec(ctx, query, accountID, symbol)
	if err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}

	return nil
}

// LoadHoldings loads all holdings for an account
func (r *HoldingRepository) LoadHoldings(ctx context.Context, accountID string) ([]*execution.Holding, error) {
	query := `
		SELECT
			h.account_id,
			h.symbol,
			h.qty,
			h.avg_price,
			h.current_price,
			(h.current_price - h.avg_price) * h.qty as pnl,
			((h.current_price - h.avg_price) / NULLIF(h.avg_price, 0) * 100) as pnl_pct,
			h.updated_ts,
			COALESCE(p.exit_mode, 'ENABLED') as exit_mode,
			h.raw
		FROM trade.holdings h
		LEFT JOIN trade.positions p ON h.account_id = p.account_id AND h.symbol = p.symbol
		WHERE h.account_id = $1 AND h.qty > 0
		ORDER BY h.symbol ASC
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query holdings for account: %w", err)
	}
	defer rows.Close()

	var holdings []*execution.Holding
	for rows.Next() {
		h := &execution.Holding{}
		err := rows.Scan(
			&h.AccountID,
			&h.Symbol,
			&h.Qty,
			&h.AvgPrice,
			&h.CurrentPrice,
			&h.Pnl,
			&h.PnlPct,
			&h.UpdatedTS,
			&h.ExitMode,
			&h.Raw,
		)
		if err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return holdings, nil
}
