package pricesync

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// ==============================================================================
// Repository Adapters for PriorityManager
// ==============================================================================

// PositionAdapter adapts exit.PositionRepository to pricesync.PositionRepository
type PositionAdapter struct {
	repo exit.PositionRepository
}

// NewPositionAdapter creates a new PositionAdapter
func NewPositionAdapter(repo exit.PositionRepository) *PositionAdapter {
	return &PositionAdapter{repo: repo}
}

// GetOpenPositions returns OPEN positions
func (a *PositionAdapter) GetOpenPositions(ctx context.Context) ([]PositionSummary, error) {
	positions, err := a.repo.GetAllOpenPositions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]PositionSummary, 0, len(positions))
	for _, pos := range positions {
		if pos.Status == "OPEN" {
			result = append(result, PositionSummary{
				Symbol: pos.Symbol,
				Status: pos.Status,
			})
		}
	}

	return result, nil
}

// GetClosingPositions returns CLOSING positions
func (a *PositionAdapter) GetClosingPositions(ctx context.Context) ([]PositionSummary, error) {
	positions, err := a.repo.GetAllOpenPositions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]PositionSummary, 0)
	for _, pos := range positions {
		if pos.Status == "CLOSING" {
			result = append(result, PositionSummary{
				Symbol: pos.Symbol,
				Status: pos.Status,
			})
		}
	}

	return result, nil
}

// ==============================================================================
// OrderAdapter - Adapts OrderIntent to get active order symbols
// ==============================================================================

// OrderAdapter provides active order symbols from order_intents table
type OrderAdapter struct {
	pool *pgxpool.Pool
}

// NewOrderAdapter creates a new OrderAdapter
func NewOrderAdapter(pool *pgxpool.Pool) *OrderAdapter {
	return &OrderAdapter{pool: pool}
}

// GetActiveOrderSymbols returns symbols with active (PENDING/SUBMITTED) orders
func (a *OrderAdapter) GetActiveOrderSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM trade.order_intents
		WHERE status IN ('PENDING', 'SUBMITTED', 'PARTIAL')
		ORDER BY symbol
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, rows.Err()
}

// ==============================================================================
// WatchlistAdapter - Adapts WatchlistReader interface
// ==============================================================================

// WatchlistAdapter adapts universe.WatchlistReader to pricesync.WatchlistRepository
type WatchlistAdapter struct {
	pool *pgxpool.Pool
}

// NewWatchlistAdapter creates a new WatchlistAdapter
func NewWatchlistAdapter(pool *pgxpool.Pool) *WatchlistAdapter {
	return &WatchlistAdapter{pool: pool}
}

// GetWatchlistSymbols returns active watchlist symbols
func (a *WatchlistAdapter) GetWatchlistSymbols(ctx context.Context) ([]string, error) {
	// v14 watchlist uses stock_code column (no is_active column)
	query := `
		SELECT DISTINCT stock_code
		FROM portfolio.watchlist
		ORDER BY stock_code
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, rows.Err()
}

// ==============================================================================
// SystemAdapter - Provides system-critical symbols (indices, ETFs, etc.)
// ==============================================================================

// SystemAdapter provides system-critical symbols
type SystemAdapter struct {
	pool *pgxpool.Pool
}

// NewSystemAdapter creates a new SystemAdapter
func NewSystemAdapter(pool *pgxpool.Pool) *SystemAdapter {
	return &SystemAdapter{pool: pool}
}

// GetSystemSymbols returns system-critical symbols
// These are hardcoded for now, but could be stored in a DB table
func (a *SystemAdapter) GetSystemSymbols(ctx context.Context) ([]string, error) {
	// System-critical symbols:
	// - Major indices ETFs
	// - Benchmark symbols for market sentiment
	return []string{
		// KOSPI 200 ETF
		"069500", // KODEX 200
		"102110", // TIGER 200

		// KOSDAQ 150 ETF
		"229200", // KODEX KOSDAQ 150

		// Inverse/Leverage (for market sentiment)
		"114800", // KODEX Inverse
		"122630", // KODEX Leverage

		// Market leaders (for sector sentiment)
		"005930", // Samsung Electronics
		"000660", // SK Hynix
		"035420", // NAVER
		"035720", // Kakao
	}, nil
}

// ==============================================================================
// DB-based System Symbols (optional - for future use)
// ==============================================================================

// GetSystemSymbolsFromDB fetches system symbols from database
// Use this if system symbols should be configurable via DB
func (a *SystemAdapter) GetSystemSymbolsFromDB(ctx context.Context) ([]string, error) {
	query := `
		SELECT symbol
		FROM market.system_symbols
		WHERE is_active = true
		ORDER BY priority DESC
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		// Fallback to hardcoded if table doesn't exist
		return a.GetSystemSymbols(ctx)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	if len(symbols) == 0 {
		// Fallback to hardcoded if no rows
		return a.GetSystemSymbols(ctx)
	}

	return symbols, rows.Err()
}
