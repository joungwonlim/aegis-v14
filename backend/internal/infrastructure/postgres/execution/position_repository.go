package execution

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// PositionRepository implements execution.PositionReader (read-only)
type PositionRepository struct {
	db *pgxpool.Pool
}

// NewPositionRepository creates a new position repository
func NewPositionRepository(db *pgxpool.Pool) *PositionRepository {
	return &PositionRepository{db: db}
}

// GetPositionBySymbol retrieves a position by symbol and status
func (r *PositionRepository) GetPositionBySymbol(ctx context.Context, accountID, symbol, status string) (*exit.Position, error) {
	query := `
		SELECT position_id, account_id, symbol, side, qty, avg_price, entry_ts,
		       status, exit_mode, exit_profile_id, strategy_id, updated_ts, version
		FROM trade.positions
		WHERE account_id = $1 AND symbol = $2 AND status = $3
	`

	var position exit.Position

	err := r.db.QueryRow(ctx, query, accountID, symbol, status).Scan(
		&position.PositionID,
		&position.AccountID,
		&position.Symbol,
		&position.Side,
		&position.Qty,
		&position.AvgPrice,
		&position.EntryTS,
		&position.Status,
		&position.ExitMode,
		&position.ExitProfileID,
		&position.StrategyID,
		&position.UpdatedTS,
		&position.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrPositionNotFound
		}
		return nil, fmt.Errorf("query position: %w", err)
	}

	return &position, nil
}

// GetPosition retrieves a position by ID
func (r *PositionRepository) GetPosition(ctx context.Context, positionID uuid.UUID) (*exit.Position, error) {
	query := `
		SELECT position_id, account_id, symbol, side, qty, avg_price, entry_ts,
		       status, exit_mode, exit_profile_id, strategy_id, updated_ts, version
		FROM trade.positions
		WHERE position_id = $1
	`

	var position exit.Position

	err := r.db.QueryRow(ctx, query, positionID).Scan(
		&position.PositionID,
		&position.AccountID,
		&position.Symbol,
		&position.Side,
		&position.Qty,
		&position.AvgPrice,
		&position.EntryTS,
		&position.Status,
		&position.ExitMode,
		&position.ExitProfileID,
		&position.StrategyID,
		&position.UpdatedTS,
		&position.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrPositionNotFound
		}
		return nil, fmt.Errorf("query position: %w", err)
	}

	return &position, nil
}
