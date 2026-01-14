package exit

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// PositionStateRepository implements exit.PositionStateRepository
type PositionStateRepository struct {
	pool *pgxpool.Pool
}

// NewPositionStateRepository creates a new position state repository
func NewPositionStateRepository(pool *pgxpool.Pool) *PositionStateRepository {
	return &PositionStateRepository{pool: pool}
}

// GetState retrieves position state (FSM)
func (r *PositionStateRepository) GetState(ctx context.Context, positionID uuid.UUID) (*exit.PositionState, error) {
	query := `
		SELECT
			position_id,
			phase,
			hwm_price,
			stop_floor_price,
			atr,
			cooldown_until,
			last_eval_ts,
			updated_ts
		FROM trade.position_state
		WHERE position_id = $1
	`

	var state exit.PositionState
	err := r.pool.QueryRow(ctx, query, positionID).Scan(
		&state.PositionID,
		&state.Phase,
		&state.HWMPrice,
		&state.StopFloorPrice,
		&state.ATR,
		&state.CooldownUntil,
		&state.LastEvalTS,
		&state.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default state (OPEN phase)
			return &exit.PositionState{
				PositionID: positionID,
				Phase:      exit.PhaseOpen,
			}, nil
		}
		return nil, fmt.Errorf("query position state: %w", err)
	}

	return &state, nil
}

// UpsertState creates or updates position state
func (r *PositionStateRepository) UpsertState(ctx context.Context, state *exit.PositionState) error {
	query := `
		INSERT INTO trade.position_state (
			position_id,
			phase,
			hwm_price,
			stop_floor_price,
			atr,
			cooldown_until,
			last_eval_ts,
			updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (position_id) DO UPDATE
		SET
			phase = EXCLUDED.phase,
			hwm_price = EXCLUDED.hwm_price,
			stop_floor_price = EXCLUDED.stop_floor_price,
			atr = EXCLUDED.atr,
			cooldown_until = EXCLUDED.cooldown_until,
			last_eval_ts = EXCLUDED.last_eval_ts,
			updated_ts = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		state.PositionID,
		state.Phase,
		state.HWMPrice,
		state.StopFloorPrice,
		state.ATR,
		state.CooldownUntil,
		state.LastEvalTS,
	)

	if err != nil {
		return fmt.Errorf("upsert position state: %w", err)
	}

	return nil
}

// UpdatePhase updates FSM phase
func (r *PositionStateRepository) UpdatePhase(ctx context.Context, positionID uuid.UUID, phase string) error {
	query := `
		UPDATE trade.position_state
		SET
			phase = $1,
			updated_ts = NOW()
		WHERE position_id = $2
	`

	_, err := r.pool.Exec(ctx, query, phase, positionID)
	if err != nil {
		return fmt.Errorf("update phase: %w", err)
	}

	return nil
}

// UpdateHWM updates High-Water Mark
func (r *PositionStateRepository) UpdateHWM(ctx context.Context, positionID uuid.UUID, hwmPrice decimal.Decimal) error {
	query := `
		UPDATE trade.position_state
		SET
			hwm_price = $1,
			updated_ts = NOW()
		WHERE position_id = $2
	`

	_, err := r.pool.Exec(ctx, query, hwmPrice, positionID)
	if err != nil {
		return fmt.Errorf("update hwm: %w", err)
	}

	return nil
}

// UpdateStopFloor updates stop floor price
func (r *PositionStateRepository) UpdateStopFloor(ctx context.Context, positionID uuid.UUID, stopFloorPrice decimal.Decimal) error {
	query := `
		UPDATE trade.position_state
		SET
			stop_floor_price = $1,
			updated_ts = NOW()
		WHERE position_id = $2
	`

	_, err := r.pool.Exec(ctx, query, stopFloorPrice, positionID)
	if err != nil {
		return fmt.Errorf("update stop floor: %w", err)
	}

	return nil
}

// UpdateATR updates cached ATR
func (r *PositionStateRepository) UpdateATR(ctx context.Context, positionID uuid.UUID, atr decimal.Decimal) error {
	query := `
		UPDATE trade.position_state
		SET
			atr = $1,
			updated_ts = NOW()
		WHERE position_id = $2
	`

	_, err := r.pool.Exec(ctx, query, atr, positionID)
	if err != nil {
		return fmt.Errorf("update atr: %w", err)
	}

	return nil
}
