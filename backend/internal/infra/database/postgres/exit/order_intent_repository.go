package exit

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// OrderIntentRepository implements exit.OrderIntentRepository
type OrderIntentRepository struct {
	pool *pgxpool.Pool
}

// NewOrderIntentRepository creates a new order intent repository
func NewOrderIntentRepository(pool *pgxpool.Pool) *OrderIntentRepository {
	return &OrderIntentRepository{pool: pool}
}

// CreateIntent creates a new intent (idempotent via action_key unique constraint)
func (r *OrderIntentRepository) CreateIntent(ctx context.Context, intent *exit.OrderIntent) error {
	query := `
		INSERT INTO trade.order_intents (
			intent_id,
			position_id,
			symbol,
			intent_type,
			qty,
			order_type,
			limit_price,
			reason_code,
			action_key,
			status,
			created_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`

	_, err := r.pool.Exec(ctx, query,
		intent.IntentID,
		intent.PositionID,
		intent.Symbol,
		intent.IntentType,
		intent.Qty,
		intent.OrderType,
		intent.LimitPrice,
		intent.ReasonCode,
		intent.ActionKey,
		intent.Status,
	)

	if err != nil {
		// Check for unique constraint violation (action_key already exists)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return exit.ErrIntentExists
		}
		return fmt.Errorf("create intent: %w", err)
	}

	return nil
}

// GetIntent retrieves an intent by ID
func (r *OrderIntentRepository) GetIntent(ctx context.Context, intentID uuid.UUID) (*exit.OrderIntent, error) {
	query := `
		SELECT
			intent_id,
			position_id,
			symbol,
			intent_type,
			qty,
			order_type,
			limit_price,
			reason_code,
			action_key,
			status,
			created_ts
		FROM trade.order_intents
		WHERE intent_id = $1
	`

	var intent exit.OrderIntent
	err := r.pool.QueryRow(ctx, query, intentID).Scan(
		&intent.IntentID,
		&intent.PositionID,
		&intent.Symbol,
		&intent.IntentType,
		&intent.Qty,
		&intent.OrderType,
		&intent.LimitPrice,
		&intent.ReasonCode,
		&intent.ActionKey,
		&intent.Status,
		&intent.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("intent not found")
		}
		return nil, fmt.Errorf("query intent: %w", err)
	}

	return &intent, nil
}

// GetIntentByActionKey retrieves an intent by action key (for idempotency check)
func (r *OrderIntentRepository) GetIntentByActionKey(ctx context.Context, actionKey string) (*exit.OrderIntent, error) {
	query := `
		SELECT
			intent_id,
			position_id,
			symbol,
			intent_type,
			qty,
			order_type,
			limit_price,
			reason_code,
			action_key,
			status,
			created_ts
		FROM trade.order_intents
		WHERE action_key = $1
	`

	var intent exit.OrderIntent
	err := r.pool.QueryRow(ctx, query, actionKey).Scan(
		&intent.IntentID,
		&intent.PositionID,
		&intent.Symbol,
		&intent.IntentType,
		&intent.Qty,
		&intent.OrderType,
		&intent.LimitPrice,
		&intent.ReasonCode,
		&intent.ActionKey,
		&intent.Status,
		&intent.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found (OK, not an error for idempotency check)
		}
		return nil, fmt.Errorf("query intent by action key: %w", err)
	}

	return &intent, nil
}

// UpdateIntentStatus updates intent status
func (r *OrderIntentRepository) UpdateIntentStatus(ctx context.Context, intentID uuid.UUID, status string) error {
	query := `
		UPDATE trade.order_intents
		SET status = $1
		WHERE intent_id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, intentID)
	if err != nil {
		return fmt.Errorf("update intent status: %w", err)
	}

	return nil
}

// GetRecentIntents retrieves recent intents (for API)
func (r *OrderIntentRepository) GetRecentIntents(ctx context.Context, limit int) ([]*exit.OrderIntent, error) {
	query := `
		SELECT
			intent_id,
			position_id,
			symbol,
			intent_type,
			qty,
			order_type,
			limit_price,
			reason_code,
			action_key,
			status,
			created_ts
		FROM trade.order_intents
		ORDER BY created_ts DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent intents: %w", err)
	}
	defer rows.Close()

	var intents []*exit.OrderIntent
	for rows.Next() {
		intent := &exit.OrderIntent{}
		err := rows.Scan(
			&intent.IntentID,
			&intent.PositionID,
			&intent.Symbol,
			&intent.IntentType,
			&intent.Qty,
			&intent.OrderType,
			&intent.LimitPrice,
			&intent.ReasonCode,
			&intent.ActionKey,
			&intent.Status,
			&intent.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan intent: %w", err)
		}
		intents = append(intents, intent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return intents, nil
}
