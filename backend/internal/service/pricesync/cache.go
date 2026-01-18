package pricesync

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// ==============================================================================
// PriceCache - In-memory 가격 캐시로 DB 조회 부하 제거
// ==============================================================================

// PriceCache provides fast in-memory access to best prices
// All reads go through cache first (cache-aside pattern)
// Cache is updated on every ProcessTick (write-through)
type PriceCache struct {
	mu     sync.RWMutex
	prices map[string]*CachedPrice // symbol → cached price

	// Metrics
	hits   int64
	misses int64

	// Optional: DB fallback for cache miss
	repo price.PriceRepository
}

// CachedPrice represents a cached best price
type CachedPrice struct {
	Symbol      string
	BestPrice   int64
	ChangePrice *int64
	ChangeRate  *float64
	Volume      *int64
	BidPrice    *int64
	AskPrice    *int64
	Source      price.Source
	Timestamp   time.Time // 가격 시각
	UpdatedAt   time.Time // 캐시 갱신 시각
	IsStale     bool
}

// NewPriceCache creates a new price cache
func NewPriceCache(repo price.PriceRepository) *PriceCache {
	return &PriceCache{
		prices: make(map[string]*CachedPrice),
		repo:   repo,
	}
}

// ==============================================================================
// Public API
// ==============================================================================

// Get returns cached price for a symbol
// Returns nil if not found (cache miss)
func (c *PriceCache) Get(symbol string) *CachedPrice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, ok := c.prices[symbol]; ok {
		c.hits++
		// Return copy to prevent external modification
		return &CachedPrice{
			Symbol:      cached.Symbol,
			BestPrice:   cached.BestPrice,
			ChangePrice: cached.ChangePrice,
			ChangeRate:  cached.ChangeRate,
			Volume:      cached.Volume,
			BidPrice:    cached.BidPrice,
			AskPrice:    cached.AskPrice,
			Source:      cached.Source,
			Timestamp:   cached.Timestamp,
			UpdatedAt:   cached.UpdatedAt,
			IsStale:     cached.IsStale,
		}
	}

	c.misses++
	return nil
}

// GetOrLoad returns cached price, loading from DB if not cached
func (c *PriceCache) GetOrLoad(ctx context.Context, symbol string) (*CachedPrice, error) {
	// Try cache first
	if cached := c.Get(symbol); cached != nil {
		return cached, nil
	}

	// Cache miss - load from DB
	if c.repo == nil {
		return nil, nil
	}

	bp, err := c.repo.GetBestPrice(ctx, symbol)
	if err != nil {
		if err == price.ErrBestPriceNotFound {
			return nil, nil
		}
		return nil, err
	}

	// Update cache
	cached := c.updateFromBestPrice(bp)
	return cached, nil
}

// GetMultiple returns cached prices for multiple symbols
// Returns only found symbols (no error for missing)
func (c *PriceCache) GetMultiple(symbols []string) map[string]*CachedPrice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CachedPrice, len(symbols))

	for _, symbol := range symbols {
		if cached, ok := c.prices[symbol]; ok {
			c.hits++
			result[symbol] = &CachedPrice{
				Symbol:      cached.Symbol,
				BestPrice:   cached.BestPrice,
				ChangePrice: cached.ChangePrice,
				ChangeRate:  cached.ChangeRate,
				Volume:      cached.Volume,
				BidPrice:    cached.BidPrice,
				AskPrice:    cached.AskPrice,
				Source:      cached.Source,
				Timestamp:   cached.Timestamp,
				UpdatedAt:   cached.UpdatedAt,
				IsStale:     cached.IsStale,
			}
		} else {
			c.misses++
		}
	}

	return result
}

// GetMultipleOrLoad returns cached prices, loading missing from DB
func (c *PriceCache) GetMultipleOrLoad(ctx context.Context, symbols []string) (map[string]*CachedPrice, error) {
	// Get from cache first
	result := c.GetMultiple(symbols)

	// Find missing symbols
	var missing []string
	for _, symbol := range symbols {
		if _, ok := result[symbol]; !ok {
			missing = append(missing, symbol)
		}
	}

	// Load missing from DB
	if len(missing) > 0 && c.repo != nil {
		bps, err := c.repo.GetBestPrices(ctx, missing)
		if err != nil {
			return nil, err
		}

		// Update cache and result
		for _, bp := range bps {
			cached := c.updateFromBestPrice(&bp)
			result[bp.Symbol] = cached
		}
	}

	return result, nil
}

// GetAll returns all cached prices
func (c *PriceCache) GetAll() map[string]*CachedPrice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CachedPrice, len(c.prices))
	for symbol, cached := range c.prices {
		result[symbol] = &CachedPrice{
			Symbol:      cached.Symbol,
			BestPrice:   cached.BestPrice,
			ChangePrice: cached.ChangePrice,
			ChangeRate:  cached.ChangeRate,
			Volume:      cached.Volume,
			BidPrice:    cached.BidPrice,
			AskPrice:    cached.AskPrice,
			Source:      cached.Source,
			Timestamp:   cached.Timestamp,
			UpdatedAt:   cached.UpdatedAt,
			IsStale:     cached.IsStale,
		}
	}

	return result
}

// Update updates cache from a tick (called on every ProcessTick)
func (c *PriceCache) Update(tick price.Tick) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Get existing or create new
	cached, exists := c.prices[tick.Symbol]
	if !exists {
		cached = &CachedPrice{Symbol: tick.Symbol}
		c.prices[tick.Symbol] = cached
	}

	// Only update if newer or higher priority source
	if exists && !shouldUpdateCache(cached, tick) {
		return
	}

	cached.BestPrice = tick.LastPrice
	cached.ChangePrice = tick.ChangePrice
	cached.ChangeRate = tick.ChangeRate
	cached.Volume = tick.Volume
	cached.BidPrice = tick.BidPrice
	cached.AskPrice = tick.AskPrice
	cached.Source = tick.Source
	cached.Timestamp = tick.TS
	cached.UpdatedAt = now
	cached.IsStale = false
}

// UpdateFromBestPrice updates cache from BestPrice (used in LoadFromDB)
func (c *PriceCache) UpdateFromBestPrice(bp *price.BestPrice) {
	c.updateFromBestPrice(bp)
}

// MarkStale marks a symbol as stale in cache
func (c *PriceCache) MarkStale(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.prices[symbol]; ok {
		cached.IsStale = true
		cached.UpdatedAt = time.Now()
	}
}

// Remove removes a symbol from cache
func (c *PriceCache) Remove(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.prices, symbol)
}

// Clear clears all cached prices
func (c *PriceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prices = make(map[string]*CachedPrice)
	c.hits = 0
	c.misses = 0
}

// LoadFromDB loads all best prices from DB into cache
// Should be called on startup
func (c *PriceCache) LoadFromDB(ctx context.Context) error {
	if c.repo == nil {
		return nil
	}

	bps, err := c.repo.GetAllBestPrices(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, bp := range bps {
		c.prices[bp.Symbol] = &CachedPrice{
			Symbol:      bp.Symbol,
			BestPrice:   bp.BestPrice,
			ChangePrice: bp.ChangePrice,
			ChangeRate:  bp.ChangeRate,
			Volume:      bp.Volume,
			BidPrice:    bp.BidPrice,
			AskPrice:    bp.AskPrice,
			Source:      bp.BestSource,
			Timestamp:   bp.BestTS,
			UpdatedAt:   bp.UpdatedTS,
			IsStale:     bp.IsStale,
		}
	}

	log.Info().
		Int("loaded", len(bps)).
		Msg("PriceCache loaded from DB")

	return nil
}

// GetStats returns cache statistics
func (c *PriceCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	total := c.hits + c.misses
	if total > 0 {
		hitRate = float64(c.hits) / float64(total) * 100
	}

	return CacheStats{
		Size:    len(c.prices),
		Hits:    c.hits,
		Misses:  c.misses,
		HitRate: hitRate,
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	Size    int
	Hits    int64
	Misses  int64
	HitRate float64 // percentage
}

// ==============================================================================
// Internal Methods
// ==============================================================================

// updateFromBestPrice updates cache from BestPrice
func (c *PriceCache) updateFromBestPrice(bp *price.BestPrice) *CachedPrice {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached := &CachedPrice{
		Symbol:      bp.Symbol,
		BestPrice:   bp.BestPrice,
		ChangePrice: bp.ChangePrice,
		ChangeRate:  bp.ChangeRate,
		Volume:      bp.Volume,
		BidPrice:    bp.BidPrice,
		AskPrice:    bp.AskPrice,
		Source:      bp.BestSource,
		Timestamp:   bp.BestTS,
		UpdatedAt:   bp.UpdatedTS,
		IsStale:     bp.IsStale,
	}

	c.prices[bp.Symbol] = cached

	// Return copy
	return &CachedPrice{
		Symbol:      cached.Symbol,
		BestPrice:   cached.BestPrice,
		ChangePrice: cached.ChangePrice,
		ChangeRate:  cached.ChangeRate,
		Volume:      cached.Volume,
		BidPrice:    cached.BidPrice,
		AskPrice:    cached.AskPrice,
		Source:      cached.Source,
		Timestamp:   cached.Timestamp,
		UpdatedAt:   cached.UpdatedAt,
		IsStale:     cached.IsStale,
	}
}

// shouldUpdateCache determines if cache should be updated
// Returns true if:
// 1. New tick is newer, OR
// 2. New tick is from higher priority source
func shouldUpdateCache(cached *CachedPrice, tick price.Tick) bool {
	// Always update if tick is newer
	if tick.TS.After(cached.Timestamp) {
		return true
	}

	// Update if higher priority source (even if same timestamp)
	// Priority: KIS_WS > KIS_REST > NAVER
	tickPriority := getSourcePriority(tick.Source)
	cachedPriority := getSourcePriority(cached.Source)

	return tickPriority > cachedPriority
}

// getSourcePriority returns priority for a source (higher = better)
func getSourcePriority(source price.Source) int {
	switch source {
	case price.SourceKISWebSocket:
		return 3
	case price.SourceKISREST:
		return 2
	case price.SourceNaver:
		return 1
	default:
		return 0
	}
}
