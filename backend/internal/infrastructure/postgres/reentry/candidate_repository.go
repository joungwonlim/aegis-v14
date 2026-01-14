package reentry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// CandidateRepository implements reentry.CandidateRepository
type CandidateRepository struct {
	db *pgxpool.Pool
}

// NewCandidateRepository creates a new candidate repository
func NewCandidateRepository(db *pgxpool.Pool) *CandidateRepository {
	return &CandidateRepository{db: db}
}

// CreateCandidate creates a new reentry candidate
func (r *CandidateRepository) CreateCandidate(ctx context.Context, candidate *reentry.ReentryCandidate) error {
	query := `
		INSERT INTO trade.reentry_candidates (
			candidate_id, exit_event_id, symbol, origin_position_id,
			exit_reason_code, exit_ts, exit_price, exit_profile_id,
			cooldown_until, state, max_reentries, reentry_count,
			reentry_profile_id, last_eval_ts, updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.Exec(ctx, query,
		candidate.CandidateID,
		candidate.ExitEventID,
		candidate.Symbol,
		candidate.OriginPositionID,
		candidate.ExitReasonCode,
		candidate.ExitTS,
		candidate.ExitPrice,
		candidate.ExitProfileID,
		candidate.CooldownUntil,
		candidate.State,
		candidate.MaxReentries,
		candidate.ReentryCount,
		candidate.ReentryProfileID,
		candidate.LastEvalTS,
		candidate.UpdatedTS,
	)

	if err != nil {
		// Check for unique constraint violation (exit_event_id already exists)
		if pgErr := err.Error(); len(pgErr) > 0 {
			return reentry.ErrCandidateExists
		}
		return fmt.Errorf("insert candidate: %w", err)
	}

	return nil
}

// GetCandidate retrieves a candidate by ID
func (r *CandidateRepository) GetCandidate(ctx context.Context, candidateID uuid.UUID) (*reentry.ReentryCandidate, error) {
	query := `
		SELECT candidate_id, exit_event_id, symbol, origin_position_id,
		       exit_reason_code, exit_ts, exit_price, exit_profile_id,
		       cooldown_until, state, max_reentries, reentry_count,
		       reentry_profile_id, last_eval_ts, updated_ts
		FROM trade.reentry_candidates
		WHERE candidate_id = $1
	`

	var candidate reentry.ReentryCandidate
	var exitProfileID sql.NullString
	var reentryProfileID sql.NullString
	var lastEvalTS sql.NullTime

	err := r.db.QueryRow(ctx, query, candidateID).Scan(
		&candidate.CandidateID,
		&candidate.ExitEventID,
		&candidate.Symbol,
		&candidate.OriginPositionID,
		&candidate.ExitReasonCode,
		&candidate.ExitTS,
		&candidate.ExitPrice,
		&exitProfileID,
		&candidate.CooldownUntil,
		&candidate.State,
		&candidate.MaxReentries,
		&candidate.ReentryCount,
		&reentryProfileID,
		&lastEvalTS,
		&candidate.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, reentry.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("query candidate: %w", err)
	}

	if exitProfileID.Valid {
		candidate.ExitProfileID = &exitProfileID.String
	}

	if reentryProfileID.Valid {
		candidate.ReentryProfileID = &reentryProfileID.String
	}

	if lastEvalTS.Valid {
		candidate.LastEvalTS = &lastEvalTS.Time
	}

	return &candidate, nil
}

// GetCandidateByExitEvent retrieves a candidate by exit event ID (unique constraint)
func (r *CandidateRepository) GetCandidateByExitEvent(ctx context.Context, exitEventID uuid.UUID) (*reentry.ReentryCandidate, error) {
	query := `
		SELECT candidate_id, exit_event_id, symbol, origin_position_id,
		       exit_reason_code, exit_ts, exit_price, exit_profile_id,
		       cooldown_until, state, max_reentries, reentry_count,
		       reentry_profile_id, last_eval_ts, updated_ts
		FROM trade.reentry_candidates
		WHERE exit_event_id = $1
	`

	var candidate reentry.ReentryCandidate
	var exitProfileID sql.NullString
	var reentryProfileID sql.NullString
	var lastEvalTS sql.NullTime

	err := r.db.QueryRow(ctx, query, exitEventID).Scan(
		&candidate.CandidateID,
		&candidate.ExitEventID,
		&candidate.Symbol,
		&candidate.OriginPositionID,
		&candidate.ExitReasonCode,
		&candidate.ExitTS,
		&candidate.ExitPrice,
		&exitProfileID,
		&candidate.CooldownUntil,
		&candidate.State,
		&candidate.MaxReentries,
		&candidate.ReentryCount,
		&reentryProfileID,
		&lastEvalTS,
		&candidate.UpdatedTS,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, reentry.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("query candidate by exit event: %w", err)
	}

	if exitProfileID.Valid {
		candidate.ExitProfileID = &exitProfileID.String
	}

	if reentryProfileID.Valid {
		candidate.ReentryProfileID = &reentryProfileID.String
	}

	if lastEvalTS.Valid {
		candidate.LastEvalTS = &lastEvalTS.Time
	}

	return &candidate, nil
}

// UpdateCandidateState updates candidate FSM state
func (r *CandidateRepository) UpdateCandidateState(ctx context.Context, candidateID uuid.UUID, state string) error {
	query := `
		UPDATE trade.reentry_candidates
		SET state = $1, updated_ts = $2
		WHERE candidate_id = $3
	`

	result, err := r.db.Exec(ctx, query, state, time.Now(), candidateID)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	if result.RowsAffected() == 0 {
		return reentry.ErrCandidateNotFound
	}

	return nil
}

// UpdateReentryCount increments reentry count
func (r *CandidateRepository) UpdateReentryCount(ctx context.Context, candidateID uuid.UUID) error {
	query := `
		UPDATE trade.reentry_candidates
		SET reentry_count = reentry_count + 1, updated_ts = $1
		WHERE candidate_id = $2
	`

	result, err := r.db.Exec(ctx, query, time.Now(), candidateID)
	if err != nil {
		return fmt.Errorf("update reentry count: %w", err)
	}

	if result.RowsAffected() == 0 {
		return reentry.ErrCandidateNotFound
	}

	return nil
}

// UpdateLastEvalTS updates last evaluation timestamp
func (r *CandidateRepository) UpdateLastEvalTS(ctx context.Context, candidateID uuid.UUID, ts time.Time) error {
	query := `
		UPDATE trade.reentry_candidates
		SET last_eval_ts = $1, updated_ts = $2
		WHERE candidate_id = $3
	`

	result, err := r.db.Exec(ctx, query, ts, time.Now(), candidateID)
	if err != nil {
		return fmt.Errorf("update last eval ts: %w", err)
	}

	if result.RowsAffected() == 0 {
		return reentry.ErrCandidateNotFound
	}

	return nil
}

// LoadCandidatesByState loads all candidates in specific states
func (r *CandidateRepository) LoadCandidatesByState(ctx context.Context, states []string) ([]*reentry.ReentryCandidate, error) {
	query := `
		SELECT candidate_id, exit_event_id, symbol, origin_position_id,
		       exit_reason_code, exit_ts, exit_price, exit_profile_id,
		       cooldown_until, state, max_reentries, reentry_count,
		       reentry_profile_id, last_eval_ts, updated_ts
		FROM trade.reentry_candidates
		WHERE state = ANY($1)
		ORDER BY updated_ts DESC
	`

	rows, err := r.db.Query(ctx, query, states)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()

	var candidates []*reentry.ReentryCandidate
	for rows.Next() {
		var candidate reentry.ReentryCandidate
		var exitProfileID sql.NullString
		var reentryProfileID sql.NullString
		var lastEvalTS sql.NullTime

		err := rows.Scan(
			&candidate.CandidateID,
			&candidate.ExitEventID,
			&candidate.Symbol,
			&candidate.OriginPositionID,
			&candidate.ExitReasonCode,
			&candidate.ExitTS,
			&candidate.ExitPrice,
			&exitProfileID,
			&candidate.CooldownUntil,
			&candidate.State,
			&candidate.MaxReentries,
			&candidate.ReentryCount,
			&reentryProfileID,
			&lastEvalTS,
			&candidate.UpdatedTS,
		)
		if err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}

		if exitProfileID.Valid {
			candidate.ExitProfileID = &exitProfileID.String
		}

		if reentryProfileID.Valid {
			candidate.ReentryProfileID = &reentryProfileID.String
		}

		if lastEvalTS.Valid {
			candidate.LastEvalTS = &lastEvalTS.Time
		}

		candidates = append(candidates, &candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return candidates, nil
}

// LoadActiveCandidates loads all active candidates (COOLDOWN, WATCH, READY)
func (r *CandidateRepository) LoadActiveCandidates(ctx context.Context) ([]*reentry.ReentryCandidate, error) {
	return r.LoadCandidatesByState(ctx, []string{
		reentry.StateCooldown,
		reentry.StateWatch,
		reentry.StateReady,
	})
}
