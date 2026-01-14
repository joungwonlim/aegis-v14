package universe

import (
	"context"
	"time"
)

// UniverseRepository manages universe snapshots
type UniverseRepository interface {
	// SaveSnapshot saves a universe snapshot
	SaveSnapshot(ctx context.Context, snapshot *UniverseSnapshot) error

	// GetLatestSnapshot retrieves the latest universe snapshot
	GetLatestSnapshot(ctx context.Context) (*UniverseSnapshot, error)

	// GetSnapshotByID retrieves a snapshot by ID
	GetSnapshotByID(ctx context.Context, snapshotID string) (*UniverseSnapshot, error)

	// ListSnapshots lists snapshots within a time range
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*UniverseSnapshot, error)
}

// StockRepository reads stock master data
type StockRepository interface {
	// GetStockInfo retrieves stock information
	GetStockInfo(ctx context.Context, symbol string) (*StockInfo, error)

	// GetActiveStocks retrieves all active stocks with filters
	GetActiveStocks(ctx context.Context, criteria *FilterCriteria) ([]*StockInfo, error)
}

// StockInfo represents stock master information
type StockInfo struct {
	Symbol      string    `json:"symbol"`
	Name        string    `json:"name"`
	Market      string    `json:"market"`
	Sector      string    `json:"sector"`
	MarketCap   int64     `json:"market_cap"`
	IsActive    bool      `json:"is_active"`
	IsManaged   bool      `json:"is_managed"`   // 관리종목 여부
	IsSuspended bool      `json:"is_suspended"` // 거래정지 여부
	ListedDate  time.Time `json:"listed_date"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HoldingReader reads holdings data
type HoldingReader interface {
	// GetHoldings retrieves current holdings
	GetHoldings(ctx context.Context) ([]string, error)
}

// WatchlistReader reads watchlist data
type WatchlistReader interface {
	// GetWatchlist retrieves watchlist
	GetWatchlist(ctx context.Context) ([]string, error)
}

// RankingReader reads ranking data
type RankingReader interface {
	// GetRanking retrieves ranking by category and market
	GetRanking(ctx context.Context, category, market string, limit int) ([]string, error)
}

// StatisticsReader reads stock statistics (volume, value)
type StatisticsReader interface {
	// GetStockStatistics retrieves stock statistics
	GetStockStatistics(ctx context.Context, symbol string, days int) (*StockStatistics, error)

	// GetBatchStatistics retrieves statistics for multiple symbols
	GetBatchStatistics(ctx context.Context, symbols []string, days int) (map[string]*StockStatistics, error)
}

// StockStatistics represents stock statistics
type StockStatistics struct {
	Symbol       string `json:"symbol"`
	AvgVolume5D  int64  `json:"avg_volume_5d"`  // 5일 평균 거래량
	AvgValue5D   int64  `json:"avg_value_5d"`   // 5일 평균 거래대금
	AvgVolume20D int64  `json:"avg_volume_20d"` // 20일 평균 거래량
	AvgValue20D  int64  `json:"avg_value_20d"`  // 20일 평균 거래대금
}
