package price

import "errors"

// Domain errors
var (
	// Tick errors
	ErrInvalidSymbol = errors.New("invalid symbol")
	ErrInvalidSource = errors.New("invalid source")
	ErrInvalidPrice  = errors.New("invalid price: must be positive")

	// BestPrice errors
	ErrBestPriceNotFound = errors.New("best price not found")
	ErrAllSourcesStale   = errors.New("all sources are stale")

	// Freshness errors
	ErrFreshnessNotFound = errors.New("freshness data not found")
	ErrNoFreshSource     = errors.New("no fresh source available")

	// Repository errors
	ErrRepositoryFailure = errors.New("repository operation failed")
	ErrDatabaseQuery     = errors.New("database query failed")
	ErrDatabaseInsert    = errors.New("database insert failed")
	ErrDatabaseUpdate    = errors.New("database update failed")
)
