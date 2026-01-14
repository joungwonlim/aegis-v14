package reentry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// ProfileRepository implements reentry.ProfileRepository
type ProfileRepository struct {
	db *pgxpool.Pool
}

// NewProfileRepository creates a new profile repository
func NewProfileRepository(db *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// GetProfile retrieves a profile by ID
func (r *ProfileRepository) GetProfile(ctx context.Context, profileID string) (*reentry.ReentryProfile, error) {
	query := `
		SELECT profile_id, name, description, config, is_active, created_by, created_ts
		FROM trade.reentry_profiles
		WHERE profile_id = $1
	`

	var profile reentry.ReentryProfile
	var configJSON []byte

	err := r.db.QueryRow(ctx, query, profileID).Scan(
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
			return nil, reentry.ErrProfileNotFound
		}
		return nil, fmt.Errorf("query profile: %w", err)
	}

	if err := json.Unmarshal(configJSON, &profile.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &profile, nil
}

// GetDefaultProfile retrieves the default profile
func (r *ProfileRepository) GetDefaultProfile(ctx context.Context) (*reentry.ReentryProfile, error) {
	query := `
		SELECT profile_id, name, description, config, is_active, created_by, created_ts
		FROM trade.reentry_profiles
		WHERE is_active = true
		ORDER BY created_ts ASC
		LIMIT 1
	`

	var profile reentry.ReentryProfile
	var configJSON []byte

	err := r.db.QueryRow(ctx, query).Scan(
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
			return nil, reentry.ErrProfileNotFound
		}
		return nil, fmt.Errorf("query default profile: %w", err)
	}

	if err := json.Unmarshal(configJSON, &profile.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &profile, nil
}

// ListProfiles lists all active profiles
func (r *ProfileRepository) ListProfiles(ctx context.Context) ([]*reentry.ReentryProfile, error) {
	query := `
		SELECT profile_id, name, description, config, is_active, created_by, created_ts
		FROM trade.reentry_profiles
		WHERE is_active = true
		ORDER BY created_ts DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*reentry.ReentryProfile
	for rows.Next() {
		var profile reentry.ReentryProfile
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

		if err := json.Unmarshal(configJSON, &profile.Config); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return profiles, nil
}
