package pricesync

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// ==============================================================================
// Coalescer - DB 쓰기 부하를 줄이기 위한 debounce + dedup 레이어
// ==============================================================================

// Coalescer collects ticks and flushes to DB with debouncing
// Rules:
// 1. 심볼별 최대 1초에 1번만 DB에 쓰기
// 2. 가격이 이전과 동일하면 스킵
// 3. 변화율이 너무 작으면 스킵 (노이즈 필터링)
type Coalescer struct {
	mu           sync.RWMutex
	pending      map[string]*coalescedTick // symbol → 대기 중인 최신 틱
	lastWritten  map[string]*writtenState  // symbol → 마지막 DB 기록 상태

	flushInterval   time.Duration // 기본 1초
	minPriceChange  int64         // 최소 가격 변화 (원 단위)
	tickRetention   bool          // prices_ticks 저장 여부

	// Dependencies
	repo price.PriceRepository

	// Metrics
	totalReceived  int64
	totalFlushed   int64
	totalSkipped   int64
	skippedNoChange int64
	skippedTooSoon  int64

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type coalescedTick struct {
	Tick       price.Tick
	ReceivedAt time.Time
}

type writtenState struct {
	Price     int64
	Timestamp time.Time
}

// CoalescerConfig holds configuration for Coalescer
type CoalescerConfig struct {
	FlushInterval  time.Duration // DB flush 주기 (기본: 1초)
	MinPriceChange int64         // 최소 가격 변화 (기본: 0 = 모든 변화 기록)
	TickRetention  bool          // prices_ticks 저장 여부 (기본: true)
}

// DefaultCoalescerConfig returns default configuration
func DefaultCoalescerConfig() CoalescerConfig {
	return CoalescerConfig{
		FlushInterval:  1 * time.Second,
		MinPriceChange: 0,     // 모든 변화 기록
		TickRetention:  true,  // 틱 저장 활성화
	}
}

// NewCoalescer creates a new Coalescer
func NewCoalescer(repo price.PriceRepository, config CoalescerConfig) *Coalescer {
	return &Coalescer{
		pending:        make(map[string]*coalescedTick),
		lastWritten:    make(map[string]*writtenState),
		flushInterval:  config.FlushInterval,
		minPriceChange: config.MinPriceChange,
		tickRetention:  config.TickRetention,
		repo:           repo,
	}
}

// Start starts the coalescer flush loop
func (c *Coalescer) Start(ctx context.Context) {
	c.ctx, c.cancel = context.WithCancel(ctx)

	c.wg.Add(1)
	go c.flushLoop()

	log.Info().
		Dur("flush_interval", c.flushInterval).
		Int64("min_price_change", c.minPriceChange).
		Bool("tick_retention", c.tickRetention).
		Msg("Coalescer started")
}

// Stop stops the coalescer
func (c *Coalescer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	// Final flush
	c.flush()

	log.Info().
		Int64("total_received", c.totalReceived).
		Int64("total_flushed", c.totalFlushed).
		Int64("total_skipped", c.totalSkipped).
		Msg("Coalescer stopped")
}

// Enqueue adds a tick to be coalesced
// Returns immediately (non-blocking)
func (c *Coalescer) Enqueue(tick price.Tick) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalReceived++

	// 항상 최신 틱으로 덮어씀 (같은 심볼이면)
	c.pending[tick.Symbol] = &coalescedTick{
		Tick:       tick,
		ReceivedAt: time.Now(),
	}
}

// flushLoop runs the periodic flush
func (c *Coalescer) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.flush()
		}
	}
}

// flush writes pending ticks to DB
func (c *Coalescer) flush() {
	c.mu.Lock()

	// Collect ticks to flush
	toFlush := make(map[string]*coalescedTick)
	now := time.Now()

	for symbol, pending := range c.pending {
		last, exists := c.lastWritten[symbol]

		// Rule 1: 가격 변화 없으면 스킵
		if exists && last.Price == pending.Tick.LastPrice {
			c.skippedNoChange++
			c.totalSkipped++
			delete(c.pending, symbol)
			continue
		}

		// Rule 2: 최소 가격 변화 체크
		if exists && c.minPriceChange > 0 {
			diff := abs(pending.Tick.LastPrice - last.Price)
			if diff < c.minPriceChange {
				c.skippedNoChange++
				c.totalSkipped++
				delete(c.pending, symbol)
				continue
			}
		}

		// Rule 3: 너무 빈번한 쓰기 방지 (1초 미만이면 대기)
		if exists && now.Sub(last.Timestamp) < c.flushInterval {
			c.skippedTooSoon++
			// 삭제하지 않고 다음 flush에서 처리
			continue
		}

		toFlush[symbol] = pending
		delete(c.pending, symbol)
	}

	c.mu.Unlock()

	if len(toFlush) == 0 {
		return
	}

	// Write to DB (outside lock)
	successCount := 0
	for symbol, pending := range toFlush {
		if err := c.writeToDB(pending.Tick); err != nil {
			log.Error().Err(err).Str("symbol", symbol).Msg("Coalescer flush failed")
			continue
		}

		c.mu.Lock()
		c.lastWritten[symbol] = &writtenState{
			Price:     pending.Tick.LastPrice,
			Timestamp: time.Now(),
		}
		c.totalFlushed++
		c.mu.Unlock()

		successCount++
	}

	if successCount > 0 {
		log.Debug().
			Int("flushed", successCount).
			Int("total_pending", len(c.pending)).
			Msg("Coalescer flushed")
	}
}

// writeToDB writes a single tick to database
func (c *Coalescer) writeToDB(tick price.Tick) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	// 1. Save tick to prices_ticks (if retention enabled)
	if c.tickRetention {
		if err := c.repo.InsertTick(ctx, tick); err != nil {
			// 틱 저장 실패는 경고만 (best price 업데이트는 계속)
			log.Warn().Err(err).Str("symbol", tick.Symbol).Msg("Failed to insert tick")
		}
	}

	// 2. Calculate freshness
	now := time.Now()
	isTrading := IsMarketOpen(now)
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

	if err := c.repo.UpsertFreshness(ctx, freshnessInput); err != nil {
		return err
	}

	// 4. Get all freshness to select best
	freshnesses, err := c.repo.GetFreshnessBySymbol(ctx, tick.Symbol)
	if err != nil {
		return err
	}

	// 5. Select best source
	bestSource, found := price.SelectBestSource(freshnesses)
	if !found {
		// 모든 소스가 stale - best를 stale로 마킹
		return c.markStale(ctx, tick.Symbol)
	}

	// 6. Get best tick
	bestTick, err := c.repo.GetLatestTickBySource(ctx, tick.Symbol, bestSource)
	if err != nil {
		return err
	}
	if bestTick == nil {
		return nil // No tick found, skip
	}

	// 7. Fix sign consistency
	changePrice := bestTick.ChangePrice
	changeRate := bestTick.ChangeRate
	if changePrice != nil && changeRate != nil {
		if (*changeRate > 0 && *changePrice < 0) || (*changeRate < 0 && *changePrice > 0) {
			correctedPrice := -*changePrice
			changePrice = &correctedPrice
		}
	}

	// 8. Upsert best price
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
		IsStale:     false,
	}

	return c.repo.UpsertBestPrice(ctx, bestPriceInput)
}

// markStale marks best price as stale
func (c *Coalescer) markStale(ctx context.Context, symbol string) error {
	bp, err := c.repo.GetBestPrice(ctx, symbol)
	if err != nil {
		if err == price.ErrBestPriceNotFound {
			return nil
		}
		return err
	}

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
		IsStale:     true,
	}

	return c.repo.UpsertBestPrice(ctx, input)
}

// GetStats returns coalescer statistics
func (c *Coalescer) GetStats() CoalescerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CoalescerStats{
		TotalReceived:   c.totalReceived,
		TotalFlushed:    c.totalFlushed,
		TotalSkipped:    c.totalSkipped,
		SkippedNoChange: c.skippedNoChange,
		SkippedTooSoon:  c.skippedTooSoon,
		PendingCount:    len(c.pending),
		TrackedSymbols:  len(c.lastWritten),
	}
}

// CoalescerStats holds statistics
type CoalescerStats struct {
	TotalReceived   int64
	TotalFlushed    int64
	TotalSkipped    int64
	SkippedNoChange int64
	SkippedTooSoon  int64
	PendingCount    int
	TrackedSymbols  int
}

// abs returns absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
