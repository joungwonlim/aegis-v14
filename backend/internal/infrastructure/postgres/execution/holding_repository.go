package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// HoldingRepository implements execution.HoldingRepository
type HoldingRepository struct {
	db *pgxpool.Pool
}

// NewHoldingRepository creates a new holding repository
func NewHoldingRepository(db *pgxpool.Pool) *HoldingRepository {
	return &HoldingRepository{db: db}
}

// UpsertHolding creates or updates a holding (idempotent by account_id, symbol)
func (r *HoldingRepository) UpsertHolding(ctx context.Context, holding *execution.Holding) error {
	rawJSON, err := json.Marshal(holding.Raw)
	if err != nil {
		return fmt.Errorf("marshal raw: %w", err)
	}

	query := `
		INSERT INTO trade.holdings (
			account_id, symbol, qty, avg_price, current_price, pnl, pnl_pct, updated_ts, raw
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (account_id, symbol) DO UPDATE SET
			qty = EXCLUDED.qty,
			avg_price = EXCLUDED.avg_price,
			current_price = EXCLUDED.current_price,
			pnl = EXCLUDED.pnl,
			pnl_pct = EXCLUDED.pnl_pct,
			updated_ts = EXCLUDED.updated_ts,
			raw = EXCLUDED.raw
	`

	_, err = r.db.Exec(ctx, query,
		holding.AccountID,
		holding.Symbol,
		holding.Qty,
		holding.AvgPrice,
		holding.CurrentPrice,
		holding.Pnl,
		holding.PnlPct,
		holding.UpdatedTS,
		rawJSON,
	)

	if err != nil {
		return fmt.Errorf("upsert holding: %w", err)
	}

	return nil
}

// GetHolding retrieves a holding by account and symbol
func (r *HoldingRepository) GetHolding(ctx context.Context, accountID, symbol string) (*execution.Holding, error) {
	query := `
		SELECT account_id, symbol, qty, avg_price, current_price, pnl, pnl_pct, updated_ts, raw
		FROM trade.holdings
		WHERE account_id = $1 AND symbol = $2
	`

	var holding execution.Holding
	var rawJSON []byte

	err := r.db.QueryRow(ctx, query, accountID, symbol).Scan(
		&holding.AccountID,
		&holding.Symbol,
		&holding.Qty,
		&holding.AvgPrice,
		&holding.CurrentPrice,
		&holding.Pnl,
		&holding.PnlPct,
		&holding.UpdatedTS,
		&rawJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrHoldingNotFound
		}
		return nil, fmt.Errorf("query holding: %w", err)
	}

	if len(rawJSON) > 0 {
		if err := json.Unmarshal(rawJSON, &holding.Raw); err != nil {
			return nil, fmt.Errorf("unmarshal raw: %w", err)
		}
	}

	return &holding, nil
}

// LoadHoldings loads all holdings for an account
func (r *HoldingRepository) LoadHoldings(ctx context.Context, accountID string) ([]*execution.Holding, error) {
	query := `
		SELECT account_id, symbol, qty, avg_price, current_price, pnl, pnl_pct, updated_ts, raw
		FROM trade.holdings
		WHERE account_id = $1
		ORDER BY symbol ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query holdings: %w", err)
	}
	defer rows.Close()

	var holdings []*execution.Holding
	for rows.Next() {
		var holding execution.Holding
		var rawJSON []byte

		err := rows.Scan(
			&holding.AccountID,
			&holding.Symbol,
			&holding.Qty,
			&holding.AvgPrice,
			&holding.CurrentPrice,
			&holding.Pnl,
			&holding.PnlPct,
			&holding.UpdatedTS,
			&rawJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}

		if len(rawJSON) > 0 {
			if err := json.Unmarshal(rawJSON, &holding.Raw); err != nil {
				return nil, fmt.Errorf("unmarshal raw: %w", err)
			}
		}

		holdings = append(holdings, &holding)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return holdings, nil
}

// DeleteHolding deletes a holding (qty=0 cleanup)
func (r *HoldingRepository) DeleteHolding(ctx context.Context, accountID, symbol string) error {
	query := `
		DELETE FROM trade.holdings
		WHERE account_id = $1 AND symbol = $2
	`

	result, err := r.db.Exec(ctx, query, accountID, symbol)
	if err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}

	if result.RowsAffected() == 0 {
		return execution.ErrHoldingNotFound
	}

	return nil
}

// GetAllHoldingsIncludingZero retrieves all holdings including those with qty=0 (for sync comparison)
func (r *HoldingRepository) GetAllHoldingsIncludingZero(ctx context.Context, accountID string) ([]*execution.Holding, error) {
	query := `
		SELECT account_id, symbol, qty, avg_price, current_price, pnl, pnl_pct, updated_ts, raw
		FROM trade.holdings
		WHERE account_id = $1
		ORDER BY symbol ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query all holdings (including zero): %w", err)
	}
	defer rows.Close()

	var holdings []*execution.Holding
	for rows.Next() {
		var holding execution.Holding
		var rawJSON []byte

		err := rows.Scan(
			&holding.AccountID,
			&holding.Symbol,
			&holding.Qty,
			&holding.AvgPrice,
			&holding.CurrentPrice,
			&holding.Pnl,
			&holding.PnlPct,
			&holding.UpdatedTS,
			&rawJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}

		if len(rawJSON) > 0 {
			if err := json.Unmarshal(rawJSON, &holding.Raw); err != nil {
				return nil, fmt.Errorf("unmarshal raw: %w", err)
			}
		}

		holdings = append(holdings, &holding)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return holdings, nil
}

// SetHoldingQtyZero sets a holding's qty to 0 (for syncing with KIS)
func (r *HoldingRepository) SetHoldingQtyZero(ctx context.Context, accountID, symbol string) error {
	query := `
		UPDATE trade.holdings
		SET qty = 0,
			updated_ts = NOW()
		WHERE account_id = $1 AND symbol = $2
	`

	_, err := r.db.Exec(ctx, query, accountID, symbol)
	if err != nil {
		return fmt.Errorf("set holding qty zero: %w", err)
	}

	return nil
}
