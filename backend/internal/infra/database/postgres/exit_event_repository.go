package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// ExitEventRepository implements execution.ExitEventRepository
type ExitEventRepository struct {
	pool *pgxpool.Pool
}

// NewExitEventRepository creates a new ExitEventRepository
func NewExitEventRepository(pool *pgxpool.Pool) *ExitEventRepository {
	return &ExitEventRepository{
		pool: pool,
	}
}

// CreateExitEvent creates a new exit event
func (r *ExitEventRepository) CreateExitEvent(ctx context.Context, event *execution.ExitEvent) error {
	query := `
		INSERT INTO trade.exit_events (
			exit_event_id,
			position_id,
			account_id,
			symbol,
			exit_ts,
			exit_qty,
			exit_avg_price,
			exit_reason_code,
			source,
			intent_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		event.ExitEventID,
		event.PositionID,
		event.AccountID,
		event.Symbol,
		event.ExitTS,
		event.ExitQty,
		event.ExitAvgPrice,
		event.ExitReasonCode,
		event.Source,
		event.IntentID,
	)

	if err != nil {
		return fmt.Errorf("create exit event: %w", err)
	}

	return nil
}

// GetExitEvent retrieves an exit event by ID
func (r *ExitEventRepository) GetExitEvent(ctx context.Context, exitEventID uuid.UUID) (*execution.ExitEvent, error) {
	query := `
		SELECT
			exit_event_id,
			position_id,
			account_id,
			symbol,
			exit_ts,
			exit_qty,
			exit_avg_price,
			exit_reason_code,
			source,
			intent_id
		FROM trade.exit_events
		WHERE exit_event_id = $1
	`

	event := &execution.ExitEvent{}
	err := r.pool.QueryRow(ctx, query, exitEventID).Scan(
		&event.ExitEventID,
		&event.PositionID,
		&event.AccountID,
		&event.Symbol,
		&event.ExitTS,
		&event.ExitQty,
		&event.ExitAvgPrice,
		&event.ExitReasonCode,
		&event.Source,
		&event.IntentID,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("exit event not found: %s", exitEventID)
	}

	if err != nil {
		return nil, fmt.Errorf("get exit event: %w", err)
	}

	return event, nil
}

// GetExitEventByPosition retrieves an exit event by position ID
func (r *ExitEventRepository) GetExitEventByPosition(ctx context.Context, positionID uuid.UUID) (*execution.ExitEvent, error) {
	query := `
		SELECT
			exit_event_id,
			position_id,
			account_id,
			symbol,
			exit_ts,
			exit_qty,
			exit_avg_price,
			exit_reason_code,
			source,
			intent_id
		FROM trade.exit_events
		WHERE position_id = $1
	`

	event := &execution.ExitEvent{}
	err := r.pool.QueryRow(ctx, query, positionID).Scan(
		&event.ExitEventID,
		&event.PositionID,
		&event.AccountID,
		&event.Symbol,
		&event.ExitTS,
		&event.ExitQty,
		&event.ExitAvgPrice,
		&event.ExitReasonCode,
		&event.Source,
		&event.IntentID,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("exit event not found for position: %s", positionID)
	}

	if err != nil {
		return nil, fmt.Errorf("get exit event by position: %w", err)
	}

	return event, nil
}

// ExitEventExists checks if an exit event exists for a position
func (r *ExitEventRepository) ExitEventExists(ctx context.Context, positionID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM trade.exit_events
			WHERE position_id = $1
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, positionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check exit event exists: %w", err)
	}

	return exists, nil
}

// LoadExitEventsSince loads exit events since timestamp
func (r *ExitEventRepository) LoadExitEventsSince(ctx context.Context, since time.Time) ([]*execution.ExitEvent, error) {
	query := `
		SELECT
			exit_event_id,
			position_id,
			account_id,
			symbol,
			exit_ts,
			exit_qty,
			exit_avg_price,
			exit_reason_code,
			source,
			intent_id
		FROM trade.exit_events
		WHERE exit_ts >= $1
		ORDER BY exit_ts ASC
	`

	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("load exit events since: %w", err)
	}
	defer rows.Close()

	var events []*execution.ExitEvent
	for rows.Next() {
		event := &execution.ExitEvent{}
		err := rows.Scan(
			&event.ExitEventID,
			&event.PositionID,
			&event.AccountID,
			&event.Symbol,
			&event.ExitTS,
			&event.ExitQty,
			&event.ExitAvgPrice,
			&event.ExitReasonCode,
			&event.Source,
			&event.IntentID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan exit event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}
