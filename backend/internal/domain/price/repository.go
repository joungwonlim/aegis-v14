package price

import (
	"context"
	"time"
)

// TickRepository defines interface for price tick operations
type TickRepository interface {
	// InsertTick inserts a new price tick
	InsertTick(ctx context.Context, tick Tick) error

	// GetLatestTicks returns latest ticks per source for a symbol
	GetLatestTicks(ctx context.Context, symbol string) ([]Tick, error)

	// GetLatestTickBySource returns latest tick for a symbol from specific source
	GetLatestTickBySource(ctx context.Context, symbol string, source Source) (*Tick, error)

	// GetTicksInTimeRange returns ticks within time range
	GetTicksInTimeRange(ctx context.Context, symbol string, start, end time.Time) ([]Tick, error)
}

// BestPriceRepository defines interface for best price operations
type BestPriceRepository interface {
	// UpsertBestPrice upserts best price for a symbol
	UpsertBestPrice(ctx context.Context, input UpsertBestPriceInput) error

	// GetBestPrice returns best price for a symbol
	GetBestPrice(ctx context.Context, symbol string) (*BestPrice, error)

	// GetBestPrices returns best prices for multiple symbols
	GetBestPrices(ctx context.Context, symbols []string) ([]BestPrice, error)

	// GetAllBestPrices returns all best prices
	GetAllBestPrices(ctx context.Context) ([]BestPrice, error)

	// GetStalePrices returns symbols with stale prices
	GetStalePrices(ctx context.Context) ([]BestPrice, error)
}

// FreshnessRepository defines interface for freshness tracking operations
type FreshnessRepository interface {
	// UpsertFreshness upserts freshness data
	UpsertFreshness(ctx context.Context, input UpsertFreshnessInput) error

	// GetFreshness returns freshness for symbol and source
	GetFreshness(ctx context.Context, symbol string, source Source) (*Freshness, error)

	// GetFreshnessBySymbol returns all freshness data for a symbol
	GetFreshnessBySymbol(ctx context.Context, symbol string) ([]Freshness, error)

	// GetFreshSymbols returns symbols with fresh prices
	GetFreshSymbols(ctx context.Context) ([]string, error)

	// GetStaleSymbols returns symbols with stale prices
	GetStaleSymbols(ctx context.Context) ([]string, error)
}

// PriceRepository combines all price-related repositories
type PriceRepository interface {
	TickRepository
	BestPriceRepository
	FreshnessRepository
}
