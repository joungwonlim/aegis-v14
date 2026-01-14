package universe

import (
	"context"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// buildSnapshot builds a universe snapshot with filtering
func (s *Service) buildSnapshot(
	ctx context.Context,
	holdings, watchlist []universe.UniverseStock,
	rankings universe.RankingBreakdown,
) (*universe.UniverseSnapshot, error) {

	seen := make(map[string]bool)
	filterStats := universe.FilterStats{}

	var finalHoldings, finalWatchlist []universe.UniverseStock
	var finalRankings universe.RankingBreakdown

	// 1. Holdings (Tier 1) - NO FILTER (always include)
	for _, stock := range holdings {
		if !seen[stock.Symbol] {
			seen[stock.Symbol] = true
			finalHoldings = append(finalHoldings, stock)
		}
	}

	// 2. Watchlist (Tier 2) - Light filter
	for _, stock := range watchlist {
		if seen[stock.Symbol] {
			continue // Already in holdings
		}

		// Apply filters
		if !s.passesFilter(stock) {
			continue
		}

		seen[stock.Symbol] = true
		finalWatchlist = append(finalWatchlist, stock)
	}

	// 3. Rankings (Tier 3) - Full filter
	filterRanking := func(stocks []universe.UniverseStock) []universe.UniverseStock {
		var filtered []universe.UniverseStock
		for _, stock := range stocks {
			if seen[stock.Symbol] {
				continue // Already included
			}

			filterStats.TotalCandidates++

			if !s.passesFilter(stock) {
				continue
			}

			seen[stock.Symbol] = true
			filtered = append(filtered, stock)
		}
		return filtered
	}

	finalRankings.QuantHigh.Kospi = filterRanking(rankings.QuantHigh.Kospi)
	finalRankings.QuantHigh.Kosdaq = filterRanking(rankings.QuantHigh.Kosdaq)
	finalRankings.PriceTop.Kospi = filterRanking(rankings.PriceTop.Kospi)
	finalRankings.PriceTop.Kosdaq = filterRanking(rankings.PriceTop.Kosdaq)
	finalRankings.Upper.Kospi = filterRanking(rankings.Upper.Kospi)
	finalRankings.Upper.Kosdaq = filterRanking(rankings.Upper.Kosdaq)
	finalRankings.Top.Kospi = filterRanking(rankings.Top.Kospi)
	finalRankings.Top.Kosdaq = filterRanking(rankings.Top.Kosdaq)
	finalRankings.Capitalization.Kospi = filterRanking(rankings.Capitalization.Kospi)
	finalRankings.Capitalization.Kosdaq = filterRanking(rankings.Capitalization.Kosdaq)

	// Update filter stats
	filterStats.Final = len(seen)

	// Generate snapshot ID
	snapshotID := time.Now().Format("20060102-1504")

	snapshot := &universe.UniverseSnapshot{
		SnapshotID:  snapshotID,
		GeneratedAt: time.Now(),
		TotalCount:  filterStats.Final,
		Holdings:    finalHoldings,
		Watchlist:   finalWatchlist,
		Rankings:    finalRankings,
		FilterStats: filterStats,
	}

	return snapshot, nil
}

// passesFilter checks if a stock passes filtering criteria
func (s *Service) passesFilter(stock universe.UniverseStock) bool {
	// Market cap filter
	if stock.MarketCap < s.filterCriteria.MinMarketCap {
		return false
	}

	// Liquidity filter (average value)
	if stock.AvgValue5D < s.filterCriteria.MinAvgValue5D {
		return false
	}

	// Volume filter
	if stock.AvgVolume5D < s.filterCriteria.MinAvgVolume5D {
		return false
	}

	// Active check
	if !stock.IsActive {
		return false
	}

	return true
}
