package exit

import "errors"

// Domain errors
var (
	// Position errors
	ErrPositionNotFound    = errors.New("position not found")
	ErrPositionClosed      = errors.New("position already closed")
	ErrPositionChanged     = errors.New("position changed during evaluation")
	ErrPositionVersionMismatch = errors.New("position version mismatch")

	// Exit control errors
	ErrExitDisabled    = errors.New("exit disabled for this position")
	ErrExitManualOnly  = errors.New("exit manual only mode")
	ErrExitPaused      = errors.New("exit globally paused")

	// Price errors
	ErrPriceFetchFailed = errors.New("price fetch failed")
	ErrStalePrice       = errors.New("price is stale")
	ErrPriceNotAvailable = errors.New("price not available")

	// Profile errors
	ErrProfileNotFound = errors.New("exit profile not found")
	ErrInvalidProfile  = errors.New("invalid exit profile configuration")

	// Intent errors
	ErrIntentExists    = errors.New("intent already exists (idempotent)")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrNoAvailableQty  = errors.New("no available quantity (already locked)")

	// Evaluation errors
	ErrMaxRetriesExceeded = errors.New("max evaluation retries exceeded")
	ErrNoTrigger          = errors.New("no trigger hit")
)
