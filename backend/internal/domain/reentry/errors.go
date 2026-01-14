package reentry

import "errors"

// Reentry errors
var (
	// Candidate errors
	ErrCandidateNotFound  = errors.New("candidate not found")
	ErrCandidateExists    = errors.New("candidate already exists for exit event")

	// Control errors
	ErrControlNotFound    = errors.New("reentry control not found")

	// Profile errors
	ErrProfileNotFound    = errors.New("reentry profile not found")

	// Business logic errors
	ErrCooldownActive     = errors.New("cooldown still active")
	ErrMaxReentriesExceeded = errors.New("max reentries exceeded")
	ErrControlPaused      = errors.New("reentry control paused")
)
