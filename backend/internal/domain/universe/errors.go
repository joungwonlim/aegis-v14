package universe

import "errors"

var (
	ErrSnapshotNotFound = errors.New("universe snapshot not found")
	ErrInvalidCriteria  = errors.New("invalid filter criteria")
	ErrNoActiveStocks   = errors.New("no active stocks found")
)
