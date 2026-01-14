package exit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// ExitProfileRepository implements exit.ExitProfileRepository
type ExitProfileRepository struct {
	pool *pgxpool.Pool
}

// NewExitProfileRepository creates a new exit profile repository
func NewExitProfileRepository(pool *pgxpool.Pool) *ExitProfileRepository {
	return &ExitProfileRepository{pool: pool}
}

// GetProfile retrieves a profile by ID
func (r *ExitProfileRepository) GetProfile(ctx context.Context, profileID string) (*exit.ExitProfile, error) {
	query := `
		SELECT
			profile_id,
			name,
			description,
			config,
			is_active,
			created_by,
			created_ts
		FROM trade.exit_profiles
		WHERE profile_id = $1
	`

	var profile exit.ExitProfile
	var configJSON []byte

	err := r.pool.QueryRow(ctx, query, profileID).Scan(
		&profile.ProfileID,
		&profile.Name,
		&profile.Description,
		&configJSON,
		&profile.IsActive,
		&profile.CreatedBy,
		&profile.CreatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("profile not found: %s", profileID)
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}

	// Unmarshal config JSON
	if err := json.Unmarshal(configJSON, &profile.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &profile, nil
}

// GetActiveProfiles retrieves all active profiles
func (r *ExitProfileRepository) GetActiveProfiles(ctx context.Context) ([]*exit.ExitProfile, error) {
	query := `
		SELECT
			profile_id,
			name,
			description,
			config,
			is_active,
			created_by,
			created_ts
		FROM trade.exit_profiles
		WHERE is_active = true
		ORDER BY created_ts DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*exit.ExitProfile
	for rows.Next() {
		var profile exit.ExitProfile
		var configJSON []byte

		err := rows.Scan(
			&profile.ProfileID,
			&profile.Name,
			&profile.Description,
			&configJSON,
			&profile.IsActive,
			&profile.CreatedBy,
			&profile.CreatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan profile: %w", err)
		}

		// Unmarshal config JSON
		if err := json.Unmarshal(configJSON, &profile.Config); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profiles: %w", err)
	}

	return profiles, nil
}

// CreateOrUpdateProfile creates or updates a profile
func (r *ExitProfileRepository) CreateOrUpdateProfile(ctx context.Context, profile *exit.ExitProfile) error {
	// Marshal config to JSON
	configJSON, err := json.Marshal(profile.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	query := `
		INSERT INTO trade.exit_profiles (
			profile_id,
			name,
			description,
			config,
			is_active,
			created_by,
			created_ts
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (profile_id) DO UPDATE
		SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			config = EXCLUDED.config,
			is_active = EXCLUDED.is_active
	`

	_, err = r.pool.Exec(ctx, query,
		profile.ProfileID,
		profile.Name,
		profile.Description,
		configJSON,
		profile.IsActive,
		profile.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("upsert profile: %w", err)
	}

	return nil
}

// DeleteProfile deactivates a profile
func (r *ExitProfileRepository) DeleteProfile(ctx context.Context, profileID string) error {
	query := `
		UPDATE trade.exit_profiles
		SET is_active = false
		WHERE profile_id = $1
	`

	result, err := r.pool.Exec(ctx, query, profileID)
	if err != nil {
		return fmt.Errorf("deactivate profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found: %s", profileID)
	}

	return nil
}
