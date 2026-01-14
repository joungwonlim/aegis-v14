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
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// IntentRepository implements execution.IntentReader (read-only)
type IntentRepository struct {
	db *pgxpool.Pool
}

// NewIntentRepository creates a new intent repository
func NewIntentRepository(db *pgxpool.Pool) *IntentRepository {
	return &IntentRepository{db: db}
}

// LoadNewIntents loads all intents with status=NEW
func (r *IntentRepository) LoadNewIntents(ctx context.Context) ([]*exit.OrderIntent, error) {
	query := `
		SELECT intent_id, position_id, symbol, intent_type, qty, order_type,
		       limit_price, reason_code, action_key, status, created_ts
		FROM trade.order_intents
		WHERE status = $1
		ORDER BY created_ts ASC
	`

	rows, err := r.db.Query(ctx, query, exit.IntentStatusNew)
	if err != nil {
		return nil, fmt.Errorf("query new intents: %w", err)
	}
	defer rows.Close()

	var intents []*exit.OrderIntent
	for rows.Next() {
		var intent exit.OrderIntent
		var limitPrice sql.NullString

		err := rows.Scan(
			&intent.IntentID,
			&intent.PositionID,
			&intent.Symbol,
			&intent.IntentType,
			&intent.Qty,
			&intent.OrderType,
			&limitPrice,
			&intent.ReasonCode,
			&intent.ActionKey,
			&intent.Status,
			&intent.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan intent: %w", err)
		}

		if limitPrice.Valid {
			// Parse decimal string to decimal.Decimal
			// For now, just store as pointer (need to adjust exit.OrderIntent type)
			// intent.LimitPrice = decimal.NewFromString(limitPrice.String)
		}

		intents = append(intents, &intent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return intents, nil
}

// GetIntent retrieves an intent by ID
func (r *IntentRepository) GetIntent(ctx context.Context, intentID uuid.UUID) (*exit.OrderIntent, error) {
	query := `
		SELECT intent_id, position_id, symbol, intent_type, qty, order_type,
		       limit_price, reason_code, action_key, status, created_ts
		FROM trade.order_intents
		WHERE intent_id = $1
	`

	var intent exit.OrderIntent
	var limitPrice sql.NullString

	err := r.db.QueryRow(ctx, query, intentID).Scan(
		&intent.IntentID,
		&intent.PositionID,
		&intent.Symbol,
		&intent.IntentType,
		&intent.Qty,
		&intent.OrderType,
		&limitPrice,
		&intent.ReasonCode,
		&intent.ActionKey,
		&intent.Status,
		&intent.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, execution.ErrIntentNotFound
		}
		return nil, fmt.Errorf("query intent: %w", err)
	}

	if limitPrice.Valid {
		// Parse decimal string
	}

	return &intent, nil
}

// LoadIntentsForPosition loads all intents for a position (for exit reason determination)
func (r *IntentRepository) LoadIntentsForPosition(ctx context.Context, positionID uuid.UUID, intentTypes []string, statuses []string, since time.Time) ([]*exit.OrderIntent, error) {
	query := `
		SELECT intent_id, position_id, symbol, intent_type, qty, order_type,
		       limit_price, reason_code, action_key, status, created_ts
		FROM trade.order_intents
		WHERE position_id = $1
		  AND intent_type = ANY($2)
		  AND status = ANY($3)
		  AND created_ts >= $4
		ORDER BY created_ts DESC
	`

	rows, err := r.db.Query(ctx, query, positionID, intentTypes, statuses, since)
	if err != nil {
		return nil, fmt.Errorf("query intents for position: %w", err)
	}
	defer rows.Close()

	var intents []*exit.OrderIntent
	for rows.Next() {
		var intent exit.OrderIntent
		var limitPrice sql.NullString

		err := rows.Scan(
			&intent.IntentID,
			&intent.PositionID,
			&intent.Symbol,
			&intent.IntentType,
			&intent.Qty,
			&intent.OrderType,
			&limitPrice,
			&intent.ReasonCode,
			&intent.ActionKey,
			&intent.Status,
			&intent.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan intent: %w", err)
		}

		if limitPrice.Valid {
			// Parse decimal string
		}

		intents = append(intents, &intent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return intents, nil
}

// UpdateIntentStatus updates intent status (ONLY: NEW → SUBMITTED/FAILED/REJECTED/DUPLICATE)
func (r *IntentRepository) UpdateIntentStatus(ctx context.Context, intentID uuid.UUID, status string) error {
	query := `
		UPDATE trade.order_intents
		SET status = $1
		WHERE intent_id = $2
	`

	result, err := r.db.Exec(ctx, query, status, intentID)
	if err != nil {
		return fmt.Errorf("update intent status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return execution.ErrIntentNotFound
	}

	return nil
}
