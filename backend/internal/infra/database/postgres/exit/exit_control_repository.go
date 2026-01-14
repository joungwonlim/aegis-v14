package exit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// ExitControlRepository implements exit.ExitControlRepository
type ExitControlRepository struct {
	pool *pgxpool.Pool
}

// NewExitControlRepository creates a new exit control repository
func NewExitControlRepository(pool *pgxpool.Pool) *ExitControlRepository {
	return &ExitControlRepository{pool: pool}
}

// GetControl retrieves current control mode (singleton, id=1)
func (r *ExitControlRepository) GetControl(ctx context.Context) (*exit.ExitControl, error) {
	query := `
		SELECT
			id,
			mode,
			reason,
			updated_by,
			updated_ts
		FROM trade.exit_control
		WHERE id = 1
	`

	var ctrl exit.ExitControl
	err := r.pool.QueryRow(ctx, query).Scan(
		&ctrl.ID,
		&ctrl.Mode,
		&ctrl.Reason,
		&ctrl.UpdatedBy,
		&ctrl.UpdatedTS,
	)

	if err != nil {
		return nil, fmt.Errorf("query exit control: %w", err)
	}

	return &ctrl, nil
}

// UpdateControl updates control mode
func (r *ExitControlRepository) UpdateControl(ctx context.Context, mode string, reason *string, updatedBy string) error {
	query := `
		UPDATE trade.exit_control
		SET
			mode = $1,
			reason = $2,
			updated_by = $3,
			updated_ts = NOW()
		WHERE id = 1
	`

	_, err := r.pool.Exec(ctx, query, mode, reason, updatedBy)
	if err != nil {
		return fmt.Errorf("update exit control: %w", err)
	}

	return nil
}
