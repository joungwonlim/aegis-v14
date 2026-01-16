package pricesync

import (
	"context"
	"fmt"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/price"
)

// Service handles price synchronization logic
type Service struct {
	repo price.PriceRepository
}

// NewService creates a new PriceSync service
func NewService(repo price.PriceRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// ProcessTick processes a new price tick
// This is the core PriceSync logic:
// 1. Save tick to DB
// 2. Update freshness for this source
// 3. Get all freshness for this symbol
// 4. Select best source based on quality score
// 5. Update best price
func (s *Service) ProcessTick(ctx context.Context, tick price.Tick) error {
	// 1. Save tick to prices_ticks
	if err := s.repo.InsertTick(ctx, tick); err != nil {
		return fmt.Errorf("insert tick: %w", err)
	}

	// 2. Calculate freshness for this source
	now := time.Now()
	isTrading := IsMarketOpen(now) // TODO: implement market hours check

	threshold := price.GetThreshold(tick.Source, isTrading)
	staleness := price.CalculateStaleness(tick.TS, now)
	isStale := price.IsStale(tick.TS, now, threshold)
	qualityScore := price.CalculateQualityScore(tick.Source, staleness, threshold)

	// 3. Upsert freshness
	freshnessInput := price.UpsertFreshnessInput{
		Symbol:       tick.Symbol,
		Source:       tick.Source,
		LastTS:       tick.TS,
		LastPrice:    tick.LastPrice,
		IsStale:      isStale,
		StalenessMS:  staleness,
		QualityScore: qualityScore,
	}

	if err := s.repo.UpsertFreshness(ctx, freshnessInput); err != nil {
		return fmt.Errorf("upsert freshness: %w", err)
	}

	// 4. Get all freshness for this symbol to select best source
	freshnesses, err := s.repo.GetFreshnessBySymbol(ctx, tick.Symbol)
	if err != nil {
		return fmt.Errorf("get freshness by symbol: %w", err)
	}

	// 5. Select best source
	bestSource, found := price.SelectBestSource(freshnesses)
	if !found {
		// No fresh source available, mark best price as stale
		return s.markBestPriceAsStale(ctx, tick.Symbol)
	}

	// 6. Get latest tick from best source
	bestTick, err := s.repo.GetLatestTickBySource(ctx, tick.Symbol, bestSource)
	if err != nil {
		return fmt.Errorf("get latest tick by source: %w", err)
	}
	if bestTick == nil {
		return fmt.Errorf("best tick not found for symbol=%s source=%s", tick.Symbol, bestSource)
	}

	// 7. Fix ChangePrice sign if inconsistent with ChangeRate
	changePrice := bestTick.ChangePrice
	changeRate := bestTick.ChangeRate

	// Ensure ChangePrice and ChangeRate have consistent signs
	if changePrice != nil && changeRate != nil {
		if (*changeRate > 0 && *changePrice < 0) || (*changeRate < 0 && *changePrice > 0) {
			// Signs are inconsistent - fix ChangePrice to match ChangeRate sign
			correctedPrice := -*changePrice
			changePrice = &correctedPrice
		}
	}

	// 8. Update best price
	bestPriceInput := price.UpsertBestPriceInput{
		Symbol:      bestTick.Symbol,
		BestPrice:   bestTick.LastPrice,
		BestSource:  bestTick.Source,
		BestTS:      bestTick.TS,
		ChangePrice: changePrice,
		ChangeRate:  changeRate,
		Volume:      bestTick.Volume,
		BidPrice:    bestTick.BidPrice,
		AskPrice:    bestTick.AskPrice,
		IsStale:     false, // We have a fresh source
	}

	if err := s.repo.UpsertBestPrice(ctx, bestPriceInput); err != nil {
		return fmt.Errorf("upsert best price: %w", err)
	}

	return nil
}

// markBestPriceAsStale marks best price as stale when all sources are stale
func (s *Service) markBestPriceAsStale(ctx context.Context, symbol string) error {
	// Get current best price
	bp, err := s.repo.GetBestPrice(ctx, symbol)
	if err != nil {
		if err == price.ErrBestPriceNotFound {
			// No best price exists, nothing to mark as stale
			return nil
		}
		return fmt.Errorf("get best price: %w", err)
	}

	// Mark as stale
	input := price.UpsertBestPriceInput{
		Symbol:      bp.Symbol,
		BestPrice:   bp.BestPrice,
		BestSource:  bp.BestSource,
		BestTS:      bp.BestTS,
		ChangePrice: bp.ChangePrice,
		ChangeRate:  bp.ChangeRate,
		Volume:      bp.Volume,
		BidPrice:    bp.BidPrice,
		AskPrice:    bp.AskPrice,
		IsStale:     true, // Mark as stale
	}

	if err := s.repo.UpsertBestPrice(ctx, input); err != nil {
		return fmt.Errorf("upsert best price: %w", err)
	}

	return nil
}

// GetBestPrice returns best price for a symbol
func (s *Service) GetBestPrice(ctx context.Context, symbol string) (*price.BestPrice, error) {
	return s.repo.GetBestPrice(ctx, symbol)
}

// GetBestPrices returns best prices for multiple symbols
func (s *Service) GetBestPrices(ctx context.Context, symbols []string) ([]price.BestPrice, error) {
	return s.repo.GetBestPrices(ctx, symbols)
}

// GetFreshness returns freshness data for a symbol
func (s *Service) GetFreshness(ctx context.Context, symbol string) ([]price.Freshness, error) {
	return s.repo.GetFreshnessBySymbol(ctx, symbol)
}

// IsMarketOpen checks if Korean stock market is currently open
// KRX hours:
//   - 08:30-09:00: 동시호가 (Pre-market, 시간외 단일가)
//   - 09:00-15:30: 정규장 (Regular trading)
//   - 15:30-16:00: 시간외 단일가 (After-hours)
func IsMarketOpen(t time.Time) bool {
	loc, _ := time.LoadLocation("Asia/Seoul")
	seoulTime := t.In(loc)

	// Check if weekend
	weekday := seoulTime.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Check time
	hour := seoulTime.Hour()
	minute := seoulTime.Minute()

	// Market opens at 08:30 (동시호가 시작)
	if hour < 8 {
		return false
	}
	if hour == 8 && minute < 30 {
		return false
	}

	// Market closes at 16:00 (시간외 종료)
	if hour >= 16 {
		return false
	}

	return true
}
