package exit

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// PositionRepository implements exit.PositionRepository
type PositionRepository struct {
	pool *pgxpool.Pool
}

// NewPositionRepository creates a new position repository
func NewPositionRepository(pool *pgxpool.Pool) *PositionRepository {
	return &PositionRepository{pool: pool}
}

// GetPosition retrieves a position by ID with version for optimistic locking
func (r *PositionRepository) GetPosition(ctx context.Context, positionID uuid.UUID) (*exit.Position, error) {
	query := `
		SELECT
			position_id,
			account_id,
			symbol,
			side,
			qty,
			avg_price,
			entry_ts,
			status,
			COALESCE(exit_mode, 'ENABLED') AS exit_mode,
			exit_profile_id,
			strategy_id,
			updated_ts,
			version
		FROM trade.positions
		WHERE position_id = $1
	`

	var pos exit.Position
	err := r.pool.QueryRow(ctx, query, positionID).Scan(
		&pos.PositionID,
		&pos.AccountID,
		&pos.Symbol,
		&pos.Side,
		&pos.Qty,
		&pos.AvgPrice,
		&pos.EntryTS,
		&pos.Status,
		&pos.ExitMode,
		&pos.ExitProfileID,
		&pos.StrategyID,
		&pos.UpdatedTS,
		&pos.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, exit.ErrPositionNotFound
		}
		return nil, fmt.Errorf("query position: %w", err)
	}

	return &pos, nil
}

// GetOpenPositions retrieves all OPEN positions for an account
func (r *PositionRepository) GetOpenPositions(ctx context.Context, accountID string) ([]*exit.Position, error) {
	query := `
		SELECT
			position_id,
			account_id,
			symbol,
			side,
			qty,
			avg_price,
			entry_ts,
			status,
			COALESCE(exit_mode, 'ENABLED') AS exit_mode,
			exit_profile_id,
			strategy_id,
			updated_ts,
			version
		FROM trade.positions
		WHERE account_id = $1
		  AND status = 'OPEN'
		ORDER BY entry_ts ASC
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query open positions: %w", err)
	}
	defer rows.Close()

	var positions []*exit.Position
	for rows.Next() {
		var pos exit.Position
		err := rows.Scan(
			&pos.PositionID,
			&pos.AccountID,
			&pos.Symbol,
			&pos.Side,
			&pos.Qty,
			&pos.AvgPrice,
			&pos.EntryTS,
			&pos.Status,
			&pos.ExitMode,
			&pos.ExitProfileID,
			&pos.StrategyID,
			&pos.UpdatedTS,
			&pos.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, &pos)
	}

	return positions, rows.Err()
}

// UpdateStatus updates position status (Exit Engine owns this column)
// Uses optimistic locking with version check
func (r *PositionRepository) UpdateStatus(ctx context.Context, positionID uuid.UUID, status string, expectedVersion int) error {
	query := `
		UPDATE trade.positions
		SET
			status = $1,
			updated_ts = NOW()
		WHERE position_id = $2
		  AND version = $3
	`

	result, err := r.pool.Exec(ctx, query, status, positionID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return exit.ErrPositionVersionMismatch
	}

	return nil
}

// UpdateExitMode updates exit mode for a position (Exit Engine owns this column)
func (r *PositionRepository) UpdateExitMode(ctx context.Context, positionID uuid.UUID, mode string, profileID *string) error {
	query := `
		UPDATE trade.positions
		SET
			exit_mode = $1,
			exit_profile_id = $2,
			updated_ts = NOW()
		WHERE position_id = $3
	`

	_, err := r.pool.Exec(ctx, query, mode, profileID, positionID)
	if err != nil {
		return fmt.Errorf("update exit mode: %w", err)
	}

	return nil
}

// GetAvailableQty calculates available qty (position qty - locked qty from pending orders)
func (r *PositionRepository) GetAvailableQty(ctx context.Context, positionID uuid.UUID) (int64, error) {
	query := `
		WITH position_qty AS (
			SELECT qty FROM trade.positions WHERE position_id = $1
		),
		locked_qty AS (
			SELECT COALESCE(SUM(o.qty - o.filled_qty), 0) AS locked
			FROM trade.orders o
			WHERE o.position_id = $1
			  AND o.status IN ('NEW', 'SUBMITTED', 'PARTIAL_FILLED')
		)
		SELECT
			COALESCE(p.qty, 0) - COALESCE(l.locked, 0) AS available_qty
		FROM position_qty p, locked_qty l
	`

	var availableQty int64
	err := r.pool.QueryRow(ctx, query, positionID).Scan(&availableQty)
	if err != nil {
		return 0, fmt.Errorf("get available qty: %w", err)
	}

	// Ensure non-negative
	if availableQty < 0 {
		availableQty = 0
	}

	return availableQty, nil
}
