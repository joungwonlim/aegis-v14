package stock

import "errors"

var (
	// Validation errors
	ErrInvalidSymbol = errors.New("invalid stock symbol format")
	ErrInvalidMarket = errors.New("invalid market value")
	ErrInvalidStatus = errors.New("invalid status value")
	ErrInvalidSort   = errors.New("invalid sort field")
	ErrInvalidOrder  = errors.New("invalid order direction")

	// Data errors
	ErrStockNotFound = errors.New("stock not found")
)
