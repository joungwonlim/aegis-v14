package reentry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// ControlRepository implements reentry.ControlRepository
type ControlRepository struct {
	db *pgxpool.Pool
}

// NewControlRepository creates a new control repository
func NewControlRepository(db *pgxpool.Pool) *ControlRepository {
	return &ControlRepository{db: db}
}

// GetControl retrieves the singleton control row
func (r *ControlRepository) GetControl(ctx context.Context) (*reentry.ReentryControl, error) {
	query := `
		SELECT id, mode, reason, updated_by, updated_ts
		FROM trade.reentry_control
		WHERE id = 1
	`

	var control reentry.ReentryControl
	var reason sql.NullString

	err := r.db.QueryRow(ctx, query).Scan(
		&control.ID,
		&control.Mode,
		&reason,
		&control.UpdatedBy,
		&control.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, reentry.ErrControlNotFound
		}
		return nil, fmt.Errorf("query control: %w", err)
	}

	if reason.Valid {
		control.Reason = &reason.String
	}

	return &control, nil
}

// UpdateControl updates the control row
func (r *ControlRepository) UpdateControl(ctx context.Context, control *reentry.ReentryControl) error {
	query := `
		INSERT INTO trade.reentry_control (id, mode, reason, updated_by, updated_ts)
		VALUES (1, $1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			mode = EXCLUDED.mode,
			reason = EXCLUDED.reason,
			updated_by = EXCLUDED.updated_by,
			updated_ts = EXCLUDED.updated_ts
	`

	_, err := r.db.Exec(ctx, query,
		control.Mode,
		control.Reason,
		control.UpdatedBy,
		control.UpdatedTS,
	)

	if err != nil {
		return fmt.Errorf("upsert control: %w", err)
	}

	return nil
}
