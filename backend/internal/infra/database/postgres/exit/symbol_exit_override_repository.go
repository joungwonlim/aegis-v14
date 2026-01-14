package exit

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// SymbolExitOverrideRepository implements exit.SymbolExitOverrideRepository
type SymbolExitOverrideRepository struct {
	pool *pgxpool.Pool
}

// NewSymbolExitOverrideRepository creates a new symbol exit override repository
func NewSymbolExitOverrideRepository(pool *pgxpool.Pool) *SymbolExitOverrideRepository {
	return &SymbolExitOverrideRepository{pool: pool}
}

// GetOverride retrieves symbol override
func (r *SymbolExitOverrideRepository) GetOverride(ctx context.Context, symbol string) (*exit.SymbolExitOverride, error) {
	query := `
		SELECT
			symbol,
			profile_id,
			enabled,
			effective_from,
			reason,
			created_by,
			created_ts
		FROM trade.symbol_exit_overrides
		WHERE symbol = $1
	`

	var override exit.SymbolExitOverride
	err := r.pool.QueryRow(ctx, query, symbol).Scan(
		&override.Symbol,
		&override.ProfileID,
		&override.Enabled,
		&override.EffectiveFrom,
		&override.Reason,
		&override.CreatedBy,
		&override.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("override not found for symbol: %s", symbol)
		}
		return nil, fmt.Errorf("query override: %w", err)
	}

	return &override, nil
}

// SetOverride creates or updates symbol override
func (r *SymbolExitOverrideRepository) SetOverride(ctx context.Context, override *exit.SymbolExitOverride) error {
	query := `
		INSERT INTO trade.symbol_exit_overrides (
			symbol,
			profile_id,
			enabled,
			effective_from,
			reason,
			created_by,
			created_ts
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (symbol) DO UPDATE
		SET
			profile_id = EXCLUDED.profile_id,
			enabled = EXCLUDED.enabled,
			effective_from = EXCLUDED.effective_from,
			reason = EXCLUDED.reason,
			created_by = EXCLUDED.created_by
	`

	_, err := r.pool.Exec(ctx, query,
		override.Symbol,
		override.ProfileID,
		override.Enabled,
		override.EffectiveFrom,
		override.Reason,
		override.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("upsert override: %w", err)
	}

	return nil
}

// DeleteOverride removes symbol override
func (r *SymbolExitOverrideRepository) DeleteOverride(ctx context.Context, symbol string) error {
	query := `
		DELETE FROM trade.symbol_exit_overrides
		WHERE symbol = $1
	`

	result, err := r.pool.Exec(ctx, query, symbol)
	if err != nil {
		return fmt.Errorf("delete override: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("override not found for symbol: %s", symbol)
	}

	return nil
}
