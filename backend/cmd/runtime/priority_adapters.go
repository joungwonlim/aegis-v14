package main

import (
	"context"

	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// PositionRepoAdapter adapts exit.PositionRepository to pricesync.PositionRepository
type PositionRepoAdapter struct {
	exitRepo  exit.PositionRepository
	accountID string
}

func NewPositionRepoAdapter(exitRepo exit.PositionRepository, accountID string) *PositionRepoAdapter {
	return &PositionRepoAdapter{
		exitRepo:  exitRepo,
		accountID: accountID,
	}
}

func (a *PositionRepoAdapter) GetOpenPositions(ctx context.Context) ([]pricesync.PositionSummary, error) {
	positions, err := a.exitRepo.GetOpenPositions(ctx, a.accountID)
	if err != nil {
		return nil, err
	}

	summaries := make([]pricesync.PositionSummary, 0, len(positions))
	for _, p := range positions {
		if p.Status == "OPEN" {
			summaries = append(summaries, pricesync.PositionSummary{
				Symbol: p.Symbol,
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
	// TODO: Add real order repository when available
}

func NewOrderRepoAdapter() *OrderRepoAdapter {
	return &OrderRepoAdapter{}
}

func (a *OrderRepoAdapter) GetActiveOrderSymbols(ctx context.Context) ([]string, error) {
	// TODO: Query active orders from database
	// For now, return empty list
	return []string{}, nil
}

// WatchlistRepoAdapter provides watchlist symbols
type WatchlistRepoAdapter struct {
	// TODO: Add real watchlist repository when available
}

func NewWatchlistRepoAdapter() *WatchlistRepoAdapter {
	return &WatchlistRepoAdapter{}
}

func (a *WatchlistRepoAdapter) GetWatchlistSymbols(ctx context.Context) ([]string, error) {
	// TODO: Query watchlist from database
	// For now, return empty list
	return []string{}, nil
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
