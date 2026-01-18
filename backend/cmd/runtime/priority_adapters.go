package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// PositionRepoAdapter adapts exit.PositionRepository + HoldingRepository to pricesync.PositionRepository
type PositionRepoAdapter struct {
	exitRepo    exit.PositionRepository
	holdingRepo interface {
		GetAllHoldings(ctx context.Context) ([]*execution.Holding, error)
	}
	accountID string
}

func NewPositionRepoAdapter(
	exitRepo exit.PositionRepository,
	holdingRepo interface {
		GetAllHoldings(ctx context.Context) ([]*execution.Holding, error)
	},
	accountID string,
) *PositionRepoAdapter {
	return &PositionRepoAdapter{
		exitRepo:    exitRepo,
		holdingRepo: holdingRepo,
		accountID:   accountID,
	}
}

func (a *PositionRepoAdapter) GetOpenPositions(ctx context.Context) ([]pricesync.PositionSummary, error) {
	// Use holdings instead of positions to get all holdings
	holdings, err := a.holdingRepo.GetAllHoldings(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]pricesync.PositionSummary, 0, len(holdings))
	for _, h := range holdings {
		if h.Qty > 0 {
			summaries = append(summaries, pricesync.PositionSummary{
				Symbol: h.Symbol,
				Status: "OPEN",
			})
		}
	}

	return summaries, nil
}

func (a *PositionRepoAdapter) GetClosingPositions(ctx context.Context) ([]pricesync.PositionSummary, error) {
	positions, err := a.exitRepo.GetOpenPositions(ctx, a.accountID)
	if err != nil {
		return nil, err
	}

	summaries := make([]pricesync.PositionSummary, 0)
	for _, p := range positions {
		if p.Status == "CLOSING" {
			summaries = append(summaries, pricesync.PositionSummary{
				Symbol: p.Symbol,
				Status: "CLOSING",
			})
		}
	}

	return summaries, nil
}

// OrderRepoAdapter adapts order repository to pricesync.OrderRepository
type OrderRepoAdapter struct {
	kisAdapter interface {
		GetUnfilledOrders(ctx context.Context, accountID string) ([]*execution.KISUnfilledOrder, error)
	}
	accountID string
}

func NewOrderRepoAdapter(kisAdapter interface {
	GetUnfilledOrders(ctx context.Context, accountID string) ([]*execution.KISUnfilledOrder, error)
}, accountID string) *OrderRepoAdapter {
	return &OrderRepoAdapter{
		kisAdapter: kisAdapter,
		accountID:  accountID,
	}
}

func (a *OrderRepoAdapter) GetActiveOrderSymbols(ctx context.Context) ([]string, error) {
	// Get unfilled orders from KIS
	orders, err := a.kisAdapter.GetUnfilledOrders(ctx, a.accountID)
	if err != nil {
		return nil, err
	}

	// Extract unique symbols
	symbolSet := make(map[string]bool)
	for _, o := range orders {
		if o.Symbol != "" && o.OpenQty > 0 {
			symbolSet[o.Symbol] = true
		}
	}

	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// WatchlistRepoAdapter provides watchlist symbols
type WatchlistRepoAdapter struct {
	pool *pgxpool.Pool
}

func NewWatchlistRepoAdapter(pool *pgxpool.Pool) *WatchlistRepoAdapter {
	return &WatchlistRepoAdapter{pool: pool}
}

func (a *WatchlistRepoAdapter) GetWatchlistSymbols(ctx context.Context) ([]string, error) {
	if a.pool == nil {
		return []string{}, nil
	}

	// v14 watchlist uses stock_code column (no is_active column)
	query := `
		SELECT DISTINCT stock_code
		FROM portfolio.watchlist
		ORDER BY stock_code
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		// If table doesn't exist, return empty
		return []string{}, nil
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

// HoldingRepoAdapter provides all holdings symbols
type HoldingRepoAdapter struct {
	holdingRepo interface {
		GetAllHoldings(ctx context.Context) ([]*execution.Holding, error)
	}
}

func NewHoldingRepoAdapter(holdingRepo interface {
	GetAllHoldings(ctx context.Context) ([]*execution.Holding, error)
}) *HoldingRepoAdapter {
	return &HoldingRepoAdapter{
		holdingRepo: holdingRepo,
	}
}

func (a *HoldingRepoAdapter) GetHoldingSymbols(ctx context.Context) ([]string, error) {
	holdings, err := a.holdingRepo.GetAllHoldings(ctx)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, 0, len(holdings))
	for _, h := range holdings {
		symbols = append(symbols, h.Symbol)
	}

	return symbols, nil
}

// SystemRepoAdapter provides system-critical symbols (indices, etc.)
type SystemRepoAdapter struct{}

func NewSystemRepoAdapter() *SystemRepoAdapter {
	return &SystemRepoAdapter{}
}

func (a *SystemRepoAdapter) GetSystemSymbols(ctx context.Context) ([]string, error) {
	// System symbols: KOSPI 200 ETF (069500), KOSDAQ 150 ETF (229200)
	return []string{"069500", "229200"}, nil
}

// RankingRepoAdapter provides ranking symbols from data.stock_rankings table
// (collected by fetcher every 10 minutes from Naver)
type RankingRepoAdapter struct {
	pool *pgxpool.Pool
}

func NewRankingRepoAdapter(pool *pgxpool.Pool) *RankingRepoAdapter {
	return &RankingRepoAdapter{
		pool: pool,
	}
}

// GetRankingSymbols returns unique stock codes from the stock_rankings table
// Categories included: volume, trading_value, gainers, foreign_net_buy, inst_net_buy, volume_surge, high_52week
func (a *RankingRepoAdapter) GetRankingSymbols(ctx context.Context) ([]string, error) {
	if a.pool == nil {
		return []string{}, nil
	}

	// Get unique stock codes from the latest ranking collection
	// Uses the most recent collected_at timestamp to avoid stale data
	query := `
		WITH latest_collection AS (
			SELECT MAX(collected_at) as max_collected_at
			FROM data.stock_rankings
			WHERE collected_at >= NOW() - INTERVAL '30 minutes'
		)
		SELECT DISTINCT stock_code
		FROM data.stock_rankings r
		INNER JOIN latest_collection lc ON r.collected_at = lc.max_collected_at
		ORDER BY stock_code
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		// If query fails (table might not exist), return empty list
		return []string{}, nil
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
