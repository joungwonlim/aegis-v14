package stock

import "context"

// Repository defines the interface for stock data access
type Repository interface {
	// List returns paginated stocks with filters
	List(ctx context.Context, filter ListFilter) (*ListResult, error)

	// GetBySymbol returns a stock by symbol
	GetBySymbol(ctx context.Context, symbol string) (*Stock, error)
}
