package pricesync

import "errors"

// Service errors
var (
	ErrInvalidTier         = errors.New("invalid tier")
	ErrTierMaxSizeExceeded = errors.New("tier max size exceeded")
	ErrSymbolNotInTier     = errors.New("symbol not in tier")
)
