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
			last_avg_price,
			updated_ts,
			stop_floor_breach_ticks,
			trailing_breach_ticks
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
		&state.LastAvgPrice,
		&state.UpdatedTS,
		&state.StopFloorBreachTicks,
		&state.TrailingBreachTicks,
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

// UpdateLastAvgPrice updates only last_avg_price (for 부분체결/정정)
func (r *PositionStateRepository) UpdateLastAvgPrice(ctx context.Context, positionID uuid.UUID, newAvgPrice decimal.Decimal) error {
	query := `
		UPDATE trade.position_state
		SET
			last_avg_price = $1,
			updated_ts = NOW()
		WHERE position_id = $2
	`

	_, err := r.pool.Exec(ctx, query, newAvgPrice, positionID)
	if err != nil {
		return fmt.Errorf("update last_avg_price: %w", err)
	}

	return nil
}

// ResetStateToOpen resets state to OPEN phase (for 평단가 변경 시)
// Resets: Phase=OPEN, HWM=null, StopFloor=null, StopFloorBreachTicks=0, TrailingBreachTicks=0, LastAvgPrice=newAvgPrice
func (r *PositionStateRepository) ResetStateToOpen(ctx context.Context, positionID uuid.UUID, newAvgPrice decimal.Decimal) error {
	query := `
		INSERT INTO trade.position_state (
			position_id,
			phase,
			hwm_price,
			stop_floor_price,
			atr,
			cooldown_until,
			last_eval_ts,
			last_avg_price,
			stop_floor_breach_ticks,
			trailing_breach_ticks,
			updated_ts
		) VALUES ($1, $2, NULL, NULL, NULL, NULL, NOW(), $3, 0, 0, NOW())
		ON CONFLICT (position_id) DO UPDATE
		SET
			phase = 'OPEN',
			hwm_price = NULL,
			stop_floor_price = NULL,
			stop_floor_breach_ticks = 0,
			trailing_breach_ticks = 0,
			last_avg_price = EXCLUDED.last_avg_price,
			last_eval_ts = NOW(),
			updated_ts = NOW()
	`

	_, err := r.pool.Exec(ctx, query, positionID, exit.PhaseOpen, newAvgPrice)
	if err != nil {
		return fmt.Errorf("reset state to open: %w", err)
	}

	return nil
}

// IncrementStopFloorBreachTicks increments stop floor breach tick counter
func (r *PositionStateRepository) IncrementStopFloorBreachTicks(ctx context.Context, positionID uuid.UUID) error {
	query := `
		UPDATE trade.position_state
		SET
			stop_floor_breach_ticks = stop_floor_breach_ticks + 1,
			updated_ts = NOW()
		WHERE position_id = $1
	`

	_, err := r.pool.Exec(ctx, query, positionID)
	if err != nil {
		return fmt.Errorf("increment stop_floor_breach_ticks: %w", err)
	}

	return nil
}

// ResetStopFloorBreachTicks resets stop floor breach tick counter to 0
func (r *PositionStateRepository) ResetStopFloorBreachTicks(ctx context.Context, positionID uuid.UUID) error {
	query := `
		UPDATE trade.position_state
		SET
			stop_floor_breach_ticks = 0,
			updated_ts = NOW()
		WHERE position_id = $1
	`

	_, err := r.pool.Exec(ctx, query, positionID)
	if err != nil {
		return fmt.Errorf("reset stop_floor_breach_ticks: %w", err)
	}

	return nil
}

// IncrementTrailingBreachTicks increments trailing breach tick counter
func (r *PositionStateRepository) IncrementTrailingBreachTicks(ctx context.Context, positionID uuid.UUID) error {
	query := `
		UPDATE trade.position_state
		SET
			trailing_breach_ticks = trailing_breach_ticks + 1,
			updated_ts = NOW()
		WHERE position_id = $1
	`

	_, err := r.pool.Exec(ctx, query, positionID)
	if err != nil {
		return fmt.Errorf("increment trailing_breach_ticks: %w", err)
	}

	return nil
}

// ResetTrailingBreachTicks resets trailing breach tick counter to 0
func (r *PositionStateRepository) ResetTrailingBreachTicks(ctx context.Context, positionID uuid.UUID) error {
	query := `
		UPDATE trade.position_state
		SET
			trailing_breach_ticks = 0,
			updated_ts = NOW()
		WHERE position_id = $1
	`

	_, err := r.pool.Exec(ctx, query, positionID)
	if err != nil {
		return fmt.Errorf("reset trailing_breach_ticks: %w", err)
	}

	return nil
}
