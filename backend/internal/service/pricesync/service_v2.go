package pricesync

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// ==============================================================================
// ServiceV2 - DB 보호 기능이 통합된 PriceSync Service
// ==============================================================================

// ServiceV2 handles price synchronization with DB protection
// Components:
// - Cache: In-memory price cache (조회 시 DB 안 감)
// - Coalescer: DB 쓰기 debounce (1초, 가격 변화 없으면 스킵)
// - Broker: Pub/Sub for real-time updates (UI에 푸시)
type ServiceV2 struct {
	repo      price.PriceRepository
	cache     *PriceCache
	coalescer *Coalescer
	broker    *Broker
}

// ServiceV2Config holds configuration for ServiceV2
type ServiceV2Config struct {
	CoalescerConfig CoalescerConfig
	BrokerConfig    BrokerConfig
}

// DefaultServiceV2Config returns default configuration
func DefaultServiceV2Config() ServiceV2Config {
	return ServiceV2Config{
		CoalescerConfig: DefaultCoalescerConfig(),
		BrokerConfig:    DefaultBrokerConfig(),
	}
}

// NewServiceV2 creates a new ServiceV2 with DB protection
func NewServiceV2(repo price.PriceRepository, config ServiceV2Config) *ServiceV2 {
	cache := NewPriceCache(repo)
	coalescer := NewCoalescer(repo, config.CoalescerConfig)
	broker := NewBroker(config.BrokerConfig)

	return &ServiceV2{
		repo:      repo,
		cache:     cache,
		coalescer: coalescer,
		broker:    broker,
	}
}

// ==============================================================================
// Lifecycle
// ==============================================================================

// Start starts the service components
func (s *ServiceV2) Start(ctx context.Context) error {
	log.Info().Msg("Starting ServiceV2...")

	// 1. Load cache from DB
	if err := s.cache.LoadFromDB(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to load cache from DB")
		// Continue anyway - cache will be populated on first tick
	}

	// 2. Start coalescer flush loop
	s.coalescer.Start(ctx)

	log.Info().Msg("✅ ServiceV2 started")
	return nil
}

// Stop stops the service components
func (s *ServiceV2) Stop() {
	log.Info().Msg("Stopping ServiceV2...")

	// 1. Stop coalescer (will flush pending ticks)
	s.coalescer.Stop()

	// 2. Close broker
	s.broker.Close()

	log.Info().Msg("✅ ServiceV2 stopped")
}

// ==============================================================================
// ProcessTick - DB 보호 적용된 새로운 흐름
// ==============================================================================

// ProcessTick processes a new price tick with DB protection
// Flow:
// 1. Update in-memory cache (즉시, 빠름)
// 2. Publish to broker (구독자에게 즉시 푸시)
// 3. Enqueue to coalescer (1초 debounce 후 DB 쓰기)
//
// DB 쓰기는 Coalescer가 담당하므로 이 함수는 빠르게 반환됨
func (s *ServiceV2) ProcessTick(ctx context.Context, tick price.Tick) error {
	// 1. Update cache immediately
	s.cache.Update(tick)

	// 2. Publish to subscribers
	s.broker.PublishFromTick(tick)

	// 3. Enqueue for DB write (coalesced)
	s.coalescer.Enqueue(tick)

	return nil
}

// ==============================================================================
// Query Methods (Cache-first)
// ==============================================================================

// GetBestPrice returns best price for a symbol (cache-first)
func (s *ServiceV2) GetBestPrice(ctx context.Context, symbol string) (*CachedPrice, error) {
	return s.cache.GetOrLoad(ctx, symbol)
}

// GetBestPrices returns best prices for multiple symbols (cache-first)
func (s *ServiceV2) GetBestPrices(ctx context.Context, symbols []string) (map[string]*CachedPrice, error) {
	return s.cache.GetMultipleOrLoad(ctx, symbols)
}

// GetAllCachedPrices returns all cached prices
func (s *ServiceV2) GetAllCachedPrices() map[string]*CachedPrice {
	return s.cache.GetAll()
}

// GetFreshness returns freshness data for a symbol (DB direct)
func (s *ServiceV2) GetFreshness(ctx context.Context, symbol string) ([]price.Freshness, error) {
	return s.repo.GetFreshnessBySymbol(ctx, symbol)
}

// ==============================================================================
// Subscription Methods
// ==============================================================================

// Subscribe creates a subscription for a specific symbol
func (s *ServiceV2) Subscribe(symbol string) *Subscription {
	return s.broker.Subscribe(symbol)
}

// SubscribeMultiple creates subscriptions for multiple symbols
func (s *ServiceV2) SubscribeMultiple(symbols []string) *Subscription {
	return s.broker.SubscribeMultiple(symbols)
}

// SubscribeAll creates a subscription for all symbols
func (s *ServiceV2) SubscribeAll() *Subscription {
	return s.broker.SubscribeAll()
}

// Unsubscribe removes a subscription
func (s *ServiceV2) Unsubscribe(sub *Subscription) {
	s.broker.Unsubscribe(sub)
}

// ==============================================================================
// Component Access
// ==============================================================================

// Cache returns the price cache
func (s *ServiceV2) Cache() *PriceCache {
	return s.cache
}

// Broker returns the price broker
func (s *ServiceV2) Broker() *Broker {
	return s.broker
}

// Coalescer returns the coalescer
func (s *ServiceV2) Coalescer() *Coalescer {
	return s.coalescer
}

// ==============================================================================
// Statistics
// ==============================================================================

// GetStats returns all statistics
func (s *ServiceV2) GetStats() ServiceV2Stats {
	return ServiceV2Stats{
		Cache:     s.cache.GetStats(),
		Coalescer: s.coalescer.GetStats(),
		Broker:    s.broker.GetStats(),
	}
}

// ServiceV2Stats holds all statistics
type ServiceV2Stats struct {
	Cache     CacheStats
	Coalescer CoalescerStats
	Broker    BrokerStats
}
