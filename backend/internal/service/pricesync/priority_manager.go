package pricesync

import (
	"context"
	"sort"
	"sync"

	"github.com/rs/zerolog/log"
)

// ==============================================================================
// PriorityManager
// ==============================================================================

// PriorityManager manages symbol priorities for WS subscription and REST tiering
type PriorityManager struct {
	mu         sync.RWMutex
	priorities map[string]*SymbolPriority // symbol -> priority

	// External data sources
	positionRepo  PositionRepository
	orderRepo     OrderRepository
	watchlistRepo WatchlistRepository
	systemRepo    SystemRepository
	rankingRepo   RankingRepository // 랭킹 종목 (optional)
}

// SymbolPriority represents priority metadata for a symbol
type SymbolPriority struct {
	Symbol      string
	IsHolding   bool // 보유 포지션
	IsClosing   bool // 청산 진행 중
	IsOrder     bool // 활성 주문
	IsWatchlist bool // 관심 종목
	IsSystem    bool // 시스템 필수 (지수 등)
	IsRanking   bool // 랭킹 종목 (거래량/거래대금/등락률 등)
	Score       int  // 최종 점수
}

// ==============================================================================
// Repository Interfaces
// ==============================================================================

// PositionRepository provides access to position data
type PositionRepository interface {
	GetOpenPositions(ctx context.Context) ([]PositionSummary, error)
	GetClosingPositions(ctx context.Context) ([]PositionSummary, error)
}

// PositionSummary contains minimal position data for priority calculation
type PositionSummary struct {
	Symbol string
	Status string // OPEN, CLOSING
}

// OrderRepository provides access to active order data
type OrderRepository interface {
	GetActiveOrderSymbols(ctx context.Context) ([]string, error)
}

// WatchlistRepository provides access to watchlist data
type WatchlistRepository interface {
	GetWatchlistSymbols(ctx context.Context) ([]string, error)
}

// SystemRepository provides system-critical symbols
type SystemRepository interface {
	GetSystemSymbols(ctx context.Context) ([]string, error)
}

// RankingRepository provides ranking symbols (volume, gainers, etc.)
type RankingRepository interface {
	GetRankingSymbols(ctx context.Context) ([]string, error)
}

// ==============================================================================
// Constructor
// ==============================================================================

// PriorityManagerOption is a functional option for PriorityManager
type PriorityManagerOption func(*PriorityManager)

// WithRankingRepo sets the ranking repository
func WithRankingRepo(repo RankingRepository) PriorityManagerOption {
	return func(pm *PriorityManager) {
		pm.rankingRepo = repo
	}
}

// NewPriorityManager creates a new PriorityManager
func NewPriorityManager(
	positionRepo PositionRepository,
	orderRepo OrderRepository,
	watchlistRepo WatchlistRepository,
	systemRepo SystemRepository,
	opts ...PriorityManagerOption,
) *PriorityManager {
	pm := &PriorityManager{
		priorities:    make(map[string]*SymbolPriority),
		positionRepo:  positionRepo,
		orderRepo:     orderRepo,
		watchlistRepo: watchlistRepo,
		systemRepo:    systemRepo,
	}

	for _, opt := range opts {
		opt(pm)
	}

	return pm
}

// ==============================================================================
// Public API
// ==============================================================================

// Refresh recalculates all priorities by fetching latest data from repositories
func (pm *PriorityManager) Refresh(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Reset priorities
	pm.priorities = make(map[string]*SymbolPriority)

	// 1. Load holding positions (최우선)
	openPositions, err := pm.positionRepo.GetOpenPositions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get open positions")
	} else {
		for _, pos := range openPositions {
			pm.priorities[pos.Symbol] = &SymbolPriority{
				Symbol:    pos.Symbol,
				IsHolding: true,
			}
		}
	}

	// 2. Load closing positions (긴급)
	closingPositions, err := pm.positionRepo.GetClosingPositions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get closing positions")
	} else {
		for _, pos := range closingPositions {
			if p, exists := pm.priorities[pos.Symbol]; exists {
				p.IsClosing = true
			} else {
				pm.priorities[pos.Symbol] = &SymbolPriority{
					Symbol:    pos.Symbol,
					IsHolding: true,
					IsClosing: true,
				}
			}
		}
	}

	// 3. Load active orders
	orderSymbols, err := pm.orderRepo.GetActiveOrderSymbols(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get active orders")
	} else {
		for _, symbol := range orderSymbols {
			if p, exists := pm.priorities[symbol]; exists {
				p.IsOrder = true
			} else {
				pm.priorities[symbol] = &SymbolPriority{
					Symbol:  symbol,
					IsOrder: true,
				}
			}
		}
	}

	// 4. Load watchlist
	watchlistSymbols, err := pm.watchlistRepo.GetWatchlistSymbols(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get watchlist")
	} else {
		for _, symbol := range watchlistSymbols {
			if p, exists := pm.priorities[symbol]; exists {
				p.IsWatchlist = true
			} else {
				pm.priorities[symbol] = &SymbolPriority{
					Symbol:      symbol,
					IsWatchlist: true,
				}
			}
		}
	}

	// 5. Load system symbols (지수 ETF 등)
	systemSymbols, err := pm.systemRepo.GetSystemSymbols(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system symbols")
	} else {
		for _, symbol := range systemSymbols {
			if p, exists := pm.priorities[symbol]; exists {
				p.IsSystem = true
			} else {
				pm.priorities[symbol] = &SymbolPriority{
					Symbol:   symbol,
					IsSystem: true,
				}
			}
		}
	}

	// 6. Load ranking symbols (거래량/거래대금/등락률 등 상위 종목)
	var rankingSymbols []string
	if pm.rankingRepo != nil {
		rankingSymbols, err = pm.rankingRepo.GetRankingSymbols(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get ranking symbols")
		} else {
			for _, symbol := range rankingSymbols {
				if p, exists := pm.priorities[symbol]; exists {
					p.IsRanking = true
				} else {
					pm.priorities[symbol] = &SymbolPriority{
						Symbol:    symbol,
						IsRanking: true,
					}
				}
			}
		}
	}

	// 7. Calculate scores
	for _, p := range pm.priorities {
		p.Score = pm.calculateScore(p)
	}

	log.Info().
		Int("total_symbols", len(pm.priorities)).
		Int("holdings", len(openPositions)).
		Int("closing", len(closingPositions)).
		Int("orders", len(orderSymbols)).
		Int("watchlist", len(watchlistSymbols)).
		Int("system", len(systemSymbols)).
		Int("ranking", len(rankingSymbols)).
		Msg("Priorities refreshed")

	return nil
}

// GetWSSymbols returns Portfolio (Holding) symbols for WS subscription
// Only portfolio positions use WS - watchlist/ranking use REST only
func (pm *PriorityManager) GetWSSymbols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Only return Holding symbols (max 40)
	wsSymbols := make([]string, 0, 40)
	for _, p := range pm.priorities {
		if p.IsHolding && len(wsSymbols) < 40 {
			wsSymbols = append(wsSymbols, p.Symbol)
		}
	}

	return wsSymbols
}

// GetTier0Symbols returns Portfolio symbols for REST Tier0 (WS backup, 3s interval)
// Same as WS symbols - provides backup in case WS disconnects
func (pm *PriorityManager) GetTier0Symbols() []string {
	// Tier0 = same as WS symbols (Portfolio backup)
	return pm.GetWSSymbols()
}

// GetTier1Symbols returns Watchlist + Orders for REST Tier1 (10s interval)
// These don't need WS but should be refreshed reasonably fast
func (pm *PriorityManager) GetTier1Symbols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	tier1 := make([]string, 0, 100)
	for _, p := range pm.priorities {
		// Skip holdings (already in WS + Tier0)
		if p.IsHolding {
			continue
		}
		// Include watchlist and orders
		if p.IsWatchlist || p.IsOrder {
			tier1 = append(tier1, p.Symbol)
		}
	}

	return tier1
}

// GetTier2Symbols returns Ranking + System for REST Tier2 (30s interval)
// Low priority symbols that don't need frequent updates
func (pm *PriorityManager) GetTier2Symbols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	tier2 := make([]string, 0, 200)
	for _, p := range pm.priorities {
		// Skip holdings, watchlist, orders (already in higher tiers)
		if p.IsHolding || p.IsWatchlist || p.IsOrder {
			continue
		}
		// Include ranking and system symbols
		if p.IsRanking || p.IsSystem {
			tier2 = append(tier2, p.Symbol)
		}
	}

	return tier2
}

// GetPriority returns priority for a specific symbol
func (pm *PriorityManager) GetPriority(symbol string) (*SymbolPriority, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, exists := pm.priorities[symbol]
	if !exists {
		return nil, false
	}

	// Return copy to prevent external modification
	return &SymbolPriority{
		Symbol:      p.Symbol,
		IsHolding:   p.IsHolding,
		IsClosing:   p.IsClosing,
		IsOrder:     p.IsOrder,
		IsWatchlist: p.IsWatchlist,
		IsSystem:    p.IsSystem,
		IsRanking:   p.IsRanking,
		Score:       p.Score,
	}, true
}

// GetAllPriorities returns all priorities (sorted by score)
func (pm *PriorityManager) GetAllPriorities() []*SymbolPriority {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.getSortedPriorities()
}

// ==============================================================================
// Internal Methods
// ==============================================================================

// calculateScore calculates priority score based on flags
func (pm *PriorityManager) calculateScore(p *SymbolPriority) int {
	score := 0

	// P0: Holding positions (최우선)
	if p.IsHolding {
		score += 10000

		// P0+: Closing positions (긴급)
		if p.IsClosing {
			score += 5000 // Total: 15000
		}
	}

	// P1: Active orders (높은 우선순위)
	if p.IsOrder {
		score += 5000
	}

	// P2: Watchlist (중간 우선순위)
	if p.IsWatchlist {
		score += 1000
	}

	// P3: System symbols (지수 등)
	if p.IsSystem {
		score += 500
	}

	// P4: Ranking symbols (시장 관심 종목, 낮은 우선순위)
	if p.IsRanking {
		score += 100
	}

	return score
}

// getSortedPriorities returns priorities sorted by score (descending)
// Must be called with lock held
func (pm *PriorityManager) getSortedPriorities() []*SymbolPriority {
	sorted := make([]*SymbolPriority, 0, len(pm.priorities))

	for _, p := range pm.priorities {
		// Create copy
		sorted = append(sorted, &SymbolPriority{
			Symbol:      p.Symbol,
			IsHolding:   p.IsHolding,
			IsClosing:   p.IsClosing,
			IsOrder:     p.IsOrder,
			IsWatchlist: p.IsWatchlist,
			IsSystem:    p.IsSystem,
			IsRanking:   p.IsRanking,
			Score:       p.Score,
		})
	}

	// Sort by score (descending)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	return sorted
}
