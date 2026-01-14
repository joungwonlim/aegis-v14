package reentry

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CandidateRepository manages reentry candidate persistence
type CandidateRepository interface {
	// CreateCandidate creates a new reentry candidate
	CreateCandidate(ctx context.Context, candidate *ReentryCandidate) error

	// GetCandidate retrieves a candidate by ID
	GetCandidate(ctx context.Context, candidateID uuid.UUID) (*ReentryCandidate, error)

	// GetCandidateByExitEvent retrieves a candidate by exit event ID (unique constraint)
	GetCandidateByExitEvent(ctx context.Context, exitEventID uuid.UUID) (*ReentryCandidate, error)

	// UpdateCandidateState updates candidate FSM state
	UpdateCandidateState(ctx context.Context, candidateID uuid.UUID, state string) error

	// UpdateReentryCount increments reentry count
	UpdateReentryCount(ctx context.Context, candidateID uuid.UUID) error

	// UpdateLastEvalTS updates last evaluation timestamp
	UpdateLastEvalTS(ctx context.Context, candidateID uuid.UUID, ts time.Time) error

	// LoadCandidatesByState loads all candidates in specific states
	LoadCandidatesByState(ctx context.Context, states []string) ([]*ReentryCandidate, error)

	// LoadActiveCandidates loads all active candidates (COOLDOWN, WATCH, READY)
	LoadActiveCandidates(ctx context.Context) ([]*ReentryCandidate, error)
}

// ControlRepository manages reentry control persistence
type ControlRepository interface {
	// GetControl retrieves the singleton control row
	GetControl(ctx context.Context) (*ReentryControl, error)

	// UpdateControl updates the control row
	UpdateControl(ctx context.Context, control *ReentryControl) error
}

// ProfileRepository manages reentry profile persistence
type ProfileRepository interface {
	// GetProfile retrieves a profile by ID
	GetProfile(ctx context.Context, profileID string) (*ReentryProfile, error)

	// GetDefaultProfile retrieves the default profile
	GetDefaultProfile(ctx context.Context) (*ReentryProfile, error)

	// ListProfiles lists all active profiles
	ListProfiles(ctx context.Context) ([]*ReentryProfile, error)
}
