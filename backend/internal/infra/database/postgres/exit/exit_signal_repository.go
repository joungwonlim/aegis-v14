package exit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// ExitSignalRepository implements exit.ExitSignalRepository
type ExitSignalRepository struct {
	pool *pgxpool.Pool
}

// NewExitSignalRepository creates a new exit signal repository
func NewExitSignalRepository(pool *pgxpool.Pool) *ExitSignalRepository {
	return &ExitSignalRepository{pool: pool}
}

// InsertSignal inserts a signal record
func (r *ExitSignalRepository) InsertSignal(ctx context.Context, signal *exit.ExitSignal) error {
	query := `
		INSERT INTO trade.exit_signals (
			signal_id,
			position_id,
			rule_name,
			is_triggered,
			reason,
			distance,
			price,
			evaluated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		signal.SignalID,
		signal.PositionID,
		signal.RuleName,
		signal.IsTriggered,
		signal.Reason,
		signal.Distance,
		signal.Price,
		signal.EvaluatedTS,
	)

	if err != nil {
		return fmt.Errorf("insert exit signal: %w", err)
	}

	return nil
}

// GetSignals retrieves signals for a position
func (r *ExitSignalRepository) GetSignals(ctx context.Context, positionID uuid.UUID, limit int) ([]*exit.ExitSignal, error) {
	query := `
		SELECT
			signal_id,
			position_id,
			rule_name,
			is_triggered,
			reason,
			distance,
			price,
			evaluated_ts
		FROM trade.exit_signals
		WHERE position_id = $1
		ORDER BY evaluated_ts DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, positionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query exit signals: %w", err)
	}
	defer rows.Close()

	var signals []*exit.ExitSignal
	for rows.Next() {
		var signal exit.ExitSignal
		err := rows.Scan(
			&signal.SignalID,
			&signal.PositionID,
			&signal.RuleName,
			&signal.IsTriggered,
			&signal.Reason,
			&signal.Distance,
			&signal.Price,
			&signal.EvaluatedTS,
		)

		if err != nil {
			return nil, fmt.Errorf("scan exit signal: %w", err)
		}

		signals = append(signals, &signal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return signals, nil
}
