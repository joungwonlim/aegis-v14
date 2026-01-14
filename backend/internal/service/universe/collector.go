package universe

import (
	"context"
	"fmt"

	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// collectHoldings collects holdings with enriched data
func (s *Service) collectHoldings(ctx context.Context) ([]universe.UniverseStock, error) {
	// Get holdings symbols
	symbols, err := s.holdingReader.GetHoldings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get holdings: %w", err)
	}

	// Enrich with stock info and statistics
	return s.enrichStocks(ctx, symbols, universe.TierHolding, "holding")
}

// collectWatchlist collects watchlist with enriched data
func (s *Service) collectWatchlist(ctx context.Context) ([]universe.UniverseStock, error) {
	symbols, err := s.watchlistReader.GetWatchlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("get watchlist: %w", err)
	}

	return s.enrichStocks(ctx, symbols, universe.TierWatchlist, "watchlist")
}

// collectRankings collects rankings with enriched data
func (s *Service) collectRankings(ctx context.Context) (universe.RankingBreakdown, error) {
	var breakdown universe.RankingBreakdown

	categories := []struct {
		code   string
		target *universe.RankingData
	}{
		{universe.CategoryQuantHigh, &breakdown.QuantHigh},
		{universe.CategoryPriceTop, &breakdown.PriceTop},
		{universe.CategoryUpper, &breakdown.Upper},
		{universe.CategoryTop, &breakdown.Top},
		{universe.CategoryCapitalization, &breakdown.Capitalization},
	}

	for _, cat := range categories {
		// KOSPI
		kospiSymbols, err := s.rankingReader.GetRanking(ctx, cat.code, "KOSPI", s.filterCriteria.RankingLimit)
		if err == nil {
			cat.target.Kospi, _ = s.enrichStocks(ctx, kospiSymbols, universe.TierRanking, cat.code)
		}

		// KOSDAQ
		kosdaqSymbols, err := s.rankingReader.GetRanking(ctx, cat.code, "KOSDAQ", s.filterCriteria.RankingLimit)
		if err == nil {
			cat.target.Kosdaq, _ = s.enrichStocks(ctx, kosdaqSymbols, universe.TierRanking, cat.code)
		}
	}

	return breakdown, nil
}

// enrichStocks enriches symbols with stock info and statistics
func (s *Service) enrichStocks(ctx context.Context, symbols []string, tier, source string) ([]universe.UniverseStock, error) {
	if len(symbols) == 0 {
		return []universe.UniverseStock{}, nil
	}

	// Get batch statistics
	statsMap, err := s.statsRepo.GetBatchStatistics(ctx, symbols, 5)
	if err != nil {
		return nil, fmt.Errorf("get batch statistics: %w", err)
	}

	var stocks []universe.UniverseStock
	for _, symbol := range symbols {
		// Get stock info
		info, err := s.stockRepo.GetStockInfo(ctx, symbol)
		if err != nil {
			continue // Skip if stock info not found
		}

		// Get statistics
		stats := statsMap[symbol]
		if stats == nil {
			// No statistics, use defaults
			stats = &universe.StockStatistics{
				Symbol:      symbol,
				AvgVolume5D: 0,
				AvgValue5D:  0,
			}
		}

		stock := universe.UniverseStock{
			Symbol:      symbol,
			Name:        info.Name,
			Market:      info.Market,
			Sector:      info.Sector,
			Tier:        tier,
			Source:      source,
			MarketCap:   info.MarketCap,
			AvgVolume5D: stats.AvgVolume5D,
			AvgValue5D:  stats.AvgValue5D,
			IsActive:    info.IsActive,
		}

		stocks = append(stocks, stock)
	}

	return stocks, nil
}
