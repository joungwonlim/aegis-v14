package execution

import "errors"

// Execution errors
var (
	// Order errors
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderExists        = errors.New("order already exists")
	ErrDuplicateOrder     = errors.New("duplicate order (intent already submitted)")

	// Fill errors
	ErrFillNotFound       = errors.New("fill not found")
	ErrOrphanFill         = errors.New("orphan fill (order not found)")

	// Holding errors
	ErrHoldingNotFound    = errors.New("holding not found")

	// ExitEvent errors
	ErrExitEventNotFound  = errors.New("exit event not found")
	ErrExitEventExists    = errors.New("exit event already exists")

	// Intent errors
	ErrIntentNotFound     = errors.New("intent not found")

	// Position errors
	ErrPositionNotFound   = errors.New("position not found")

	// KIS API errors
	ErrKISAPIFailed       = errors.New("KIS API failed")
	ErrKISRateLimit       = errors.New("KIS rate limit exceeded")
	ErrKISTimeout         = errors.New("KIS API timeout")
	ErrKISRejected        = errors.New("KIS order rejected")

	// Sync errors
	ErrSyncFailed         = errors.New("sync failed")
	ErrStaleData          = errors.New("stale data")
)
