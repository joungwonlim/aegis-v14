package execution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// ExitEventRepository implements execution.ExitEventRepository
type ExitEventRepository struct {
	db *pgxpool.Pool
}

// NewExitEventRepository creates a new exit event repository
func NewExitEventRepository(db *pgxpool.Pool) *ExitEventRepository {
	return &ExitEventRepository{db: db}
}

// CreateExitEvent creates a new exit event
func (r *ExitEventRepository) CreateExitEvent(ctx context.Context, event *execution.ExitEvent) error {
	query := `
		INSERT INTO trade.exit_events (
			exit_event_id, position_id, account_id, symbol, exit_ts,
			exit_qty, exit_avg_price, exit_reason_code, source, intent_id,
			exit_profile_id, realized_pnl, realized_pnl_pct, created_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(ctx, query,
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
		event.ExitProfileID,
		event.RealizedPnl,
		event.RealizedPnlPct,
		event.CreatedTS,
	)

	if err != nil {
		// Check for unique constraint violation (position_id already has exit event)
		// PostgreSQL error code 23505 = unique_violation
		if pgErr := err.Error(); len(pgErr) > 0 && (len(pgErr) > 4 && pgErr[:5] == "ERROR") {
			return execution.ErrExitEventExists
		}
		return fmt.Errorf("insert exit event: %w", err)
	}

	return nil
}

// GetExitEvent retrieves an exit event by ID
func (r *ExitEventRepository) GetExitEvent(ctx context.Context, exitEventID uuid.UUID) (*execution.ExitEvent, error) {
	query := `
		SELECT exit_event_id, position_id, account_id, symbol, exit_ts,
		       exit_qty, exit_avg_price, exit_reason_code, source, intent_id,
		       exit_profile_id, realized_pnl, realized_pnl_pct, created_ts
		FROM trade.exit_events
		WHERE exit_event_id = $1
	`

	var event execution.ExitEvent
	var intentID sql.NullString
	var exitProfileID sql.NullString

	err := r.db.QueryRow(ctx, query, exitEventID).Scan(
		&event.ExitEventID,
		&event.PositionID,
		&event.AccountID,
		&event.Symbol,
		&event.ExitTS,
		&event.ExitQty,
		&event.ExitAvgPrice,
		&event.ExitReasonCode,
		&event.Source,
		&intentID,
		&exitProfileID,
		&event.RealizedPnl,
		&event.RealizedPnlPct,
		&event.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrExitEventNotFound
		}
		return nil, fmt.Errorf("query exit event: %w", err)
	}

	if intentID.Valid {
		parsedID, err := uuid.Parse(intentID.String)
		if err == nil {
			event.IntentID = &parsedID
		}
	}

	if exitProfileID.Valid {
		event.ExitProfileID = &exitProfileID.String
	}

	return &event, nil
}

// GetExitEventByPosition retrieves an exit event by position ID
func (r *ExitEventRepository) GetExitEventByPosition(ctx context.Context, positionID uuid.UUID) (*execution.ExitEvent, error) {
	query := `
		SELECT exit_event_id, position_id, account_id, symbol, exit_ts,
		       exit_qty, exit_avg_price, exit_reason_code, source, intent_id,
		       exit_profile_id, realized_pnl, realized_pnl_pct, created_ts
		FROM trade.exit_events
		WHERE position_id = $1
	`

	var event execution.ExitEvent
	var intentID sql.NullString
	var exitProfileID sql.NullString

	err := r.db.QueryRow(ctx, query, positionID).Scan(
		&event.ExitEventID,
		&event.PositionID,
		&event.AccountID,
		&event.Symbol,
		&event.ExitTS,
		&event.ExitQty,
		&event.ExitAvgPrice,
		&event.ExitReasonCode,
		&event.Source,
		&intentID,
		&exitProfileID,
		&event.RealizedPnl,
		&event.RealizedPnlPct,
		&event.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrExitEventNotFound
		}
		return nil, fmt.Errorf("query exit event: %w", err)
	}

	if intentID.Valid {
		parsedID, err := uuid.Parse(intentID.String)
		if err == nil {
			event.IntentID = &parsedID
		}
	}

	if exitProfileID.Valid {
		event.ExitProfileID = &exitProfileID.String
	}

	return &event, nil
}

// ExitEventExists checks if an exit event exists for a position
func (r *ExitEventRepository) ExitEventExists(ctx context.Context, positionID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM trade.exit_events WHERE position_id = $1
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, positionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}

	return exists, nil
}

// LoadExitEventsSince loads exit events since timestamp
func (r *ExitEventRepository) LoadExitEventsSince(ctx context.Context, since time.Time) ([]*execution.ExitEvent, error) {
	query := `
		SELECT exit_event_id, position_id, account_id, symbol, exit_ts,
		       exit_qty, exit_avg_price, exit_reason_code, source, intent_id,
		       exit_profile_id, realized_pnl, realized_pnl_pct, created_ts
		FROM trade.exit_events
		WHERE created_ts >= $1
		ORDER BY created_ts DESC
	`

	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("query exit events: %w", err)
	}
	defer rows.Close()

	var events []*execution.ExitEvent
	for rows.Next() {
		var event execution.ExitEvent
		var intentID sql.NullString
		var exitProfileID sql.NullString

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
			&intentID,
			&exitProfileID,
			&event.RealizedPnl,
			&event.RealizedPnlPct,
			&event.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan exit event: %w", err)
		}

		if intentID.Valid {
			parsedID, err := uuid.Parse(intentID.String)
			if err == nil {
				event.IntentID = &parsedID
			}
		}

		if exitProfileID.Valid {
			event.ExitProfileID = &exitProfileID.String
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}
