package pricesync

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/infra/naver"
)

// Tier represents polling tier with different intervals
type Tier int

const (
	Tier0 Tier = 0 // 1~3초 (WS 보완, ~40개)
	Tier1 Tier = 1 // 5~10초 (관심 종목, ~100개)
	Tier2 Tier = 2 // 30~120초 (전체 유니버스, ~1000개)
)

// TierConfig represents tier configuration
type TierConfig struct {
	Interval time.Duration
	MaxSize  int
}

// DefaultTierConfigs returns default tier configurations
func DefaultTierConfigs() map[Tier]TierConfig {
	return map[Tier]TierConfig{
		Tier0: {Interval: 2 * time.Second, MaxSize: 80}, // Increased to include WS backup
		Tier1: {Interval: 10 * time.Second, MaxSize: 100},
		Tier2: {Interval: 60 * time.Second, MaxSize: 1000},
	}
}

// RESTPoller handles REST API polling with tiering
type RESTPoller struct {
	kisClient   *kis.RESTClient
	naverClient *naver.Client // Fallback source
	service     *Service

	// Tier management
	tierConfigs map[Tier]TierConfig
	tiers       map[Tier][]string // tier -> symbols
	tierMu      sync.RWMutex

	// Fallback statistics
	kisFailed      int64 // Total KIS failures
	naverFallbacks int64 // Total Naver fallbacks
	naverSucceeded int64 // Successful Naver fallbacks
	statsMu        sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewRESTPoller creates a new REST poller
func NewRESTPoller(kisClient *kis.RESTClient, naverClient *naver.Client, service *Service) *RESTPoller {
	return &RESTPoller{
		kisClient:   kisClient,
		naverClient: naverClient,
		service:     service,
		tierConfigs: DefaultTierConfigs(),
		tiers: map[Tier][]string{
			Tier0: {},
			Tier1: {},
			Tier2: {},
		},
	}
}

// Start starts the REST poller
func (p *RESTPoller) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Start ticker for each tier
	for tier := range p.tierConfigs {
		p.wg.Add(1)
		go p.pollTier(tier)
	}

	return nil
}

// Stop stops the REST poller
func (p *RESTPoller) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

// pollTier polls symbols in a specific tier
func (p *RESTPoller) pollTier(tier Tier) {
	defer p.wg.Done()

	config := p.tierConfigs[tier]
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.fetchTierPrices(tier)
		}
	}
}

// fetchTierPrices fetches prices for all symbols in a tier
func (p *RESTPoller) fetchTierPrices(tier Tier) {
	p.tierMu.RLock()
	symbols := make([]string, len(p.tiers[tier]))
	copy(symbols, p.tiers[tier])
	p.tierMu.RUnlock()

	if len(symbols) == 0 {
		return
	}

	log.Info().
		Int("tier", int(tier)).
		Int("symbol_count", len(symbols)).
		Msg("REST Poller fetching prices...")

	// Try KIS first
	ticks, err := p.kisClient.GetCurrentPrices(p.ctx, symbols)
	if err != nil {
		log.Warn().
			Err(err).
			Int("tier", int(tier)).
			Int("symbol_count", len(symbols)).
			Msg("KIS price fetch failed, trying Naver fallback")

		// Update statistics
		p.statsMu.Lock()
		p.kisFailed++
		p.statsMu.Unlock()

		// Fallback to Naver if available
		if p.naverClient != nil {
			naverTicks, naverErr := p.naverClient.GetCurrentPrices(p.ctx, symbols)
			if naverErr != nil {
				log.Error().
					Err(naverErr).
					Int("tier", int(tier)).
					Msg("Naver fallback also failed")

				// Update statistics
				p.statsMu.Lock()
				p.naverFallbacks++
				p.statsMu.Unlock()

				return
			}

			// Naver fallback succeeded
			ticks = naverTicks

			log.Info().
				Int("tier", int(tier)).
				Int("count", len(naverTicks)).
				Msg("✅ Naver fallback succeeded")

			// Update statistics
			p.statsMu.Lock()
			p.naverFallbacks++
			p.naverSucceeded++
			p.statsMu.Unlock()
		} else {
			log.Error().Msg("Naver client not available for fallback")
			return
		}
	}

	// Process each tick
	successCount := 0
	for _, tick := range ticks {
		if err := p.service.ProcessTick(p.ctx, *tick); err != nil {
			log.Debug().
				Err(err).
				Str("symbol", tick.Symbol).
				Msg("Failed to process tick")
			continue
		}
		successCount++
	}

	log.Info().
		Int("tier", int(tier)).
		Int("total", len(ticks)).
		Int("success", successCount).
		Msg("✅ REST Tier prices processed")
}

// FetchSymbolPrice immediately fetches price for a single symbol
func (p *RESTPoller) FetchSymbolPrice(symbol string) error {
	if symbol == "" {
		return nil
	}

	symbols := []string{symbol}

	// Try KIS first
	ticks, err := p.kisClient.GetCurrentPrices(p.ctx, symbols)
	if err != nil {
		log.Warn().
			Err(err).
			Str("symbol", symbol).
			Msg("KIS immediate price fetch failed, trying Naver fallback")

		// Fallback to Naver if available
		if p.naverClient != nil {
			naverTicks, naverErr := p.naverClient.GetCurrentPrices(p.ctx, symbols)
			if naverErr != nil {
				log.Error().
					Err(naverErr).
					Str("symbol", symbol).
					Msg("Naver fallback also failed for immediate fetch")
				return naverErr
			}
			ticks = naverTicks
		} else {
			return err
		}
	}

	// Process ticks
	for _, tick := range ticks {
		if err := p.service.ProcessTick(p.ctx, *tick); err != nil {
			log.Debug().
				Err(err).
				Str("symbol", tick.Symbol).
				Msg("Failed to process immediate tick")
			continue
		}
		log.Info().
			Str("symbol", tick.Symbol).
			Int64("price", tick.LastPrice).
			Msg("⚡ Immediate price fetch processed")
	}

	return nil
}

// SetTierSymbols sets symbols for a specific tier
func (p *RESTPoller) SetTierSymbols(tier Tier, symbols []string) error {
	p.tierMu.Lock()
	defer p.tierMu.Unlock()

	config, exists := p.tierConfigs[tier]
	if !exists {
		return ErrInvalidTier
	}

	if len(symbols) > config.MaxSize {
		return ErrTierMaxSizeExceeded
	}

	p.tiers[tier] = symbols
	return nil
}

// GetTierSymbols returns symbols in a specific tier
func (p *RESTPoller) GetTierSymbols(tier Tier) []string {
	p.tierMu.RLock()
	defer p.tierMu.RUnlock()

	symbols := make([]string, len(p.tiers[tier]))
	copy(symbols, p.tiers[tier])
	return symbols
}

// AddSymbolToTier adds a symbol to a tier
func (p *RESTPoller) AddSymbolToTier(tier Tier, symbol string) error {
	p.tierMu.Lock()
	defer p.tierMu.Unlock()

	config, exists := p.tierConfigs[tier]
	if !exists {
		return ErrInvalidTier
	}

	// Check if already in tier
	for _, s := range p.tiers[tier] {
		if s == symbol {
			return nil // Already in tier
		}
	}

	// Check max size
	if len(p.tiers[tier]) >= config.MaxSize {
		return ErrTierMaxSizeExceeded
	}

	p.tiers[tier] = append(p.tiers[tier], symbol)
	return nil
}

// RemoveSymbolFromTier removes a symbol from a tier
func (p *RESTPoller) RemoveSymbolFromTier(tier Tier, symbol string) error {
	p.tierMu.Lock()
	defer p.tierMu.Unlock()

	symbols := p.tiers[tier]
	for i, s := range symbols {
		if s == symbol {
			// Remove by swapping with last element
			p.tiers[tier][i] = p.tiers[tier][len(p.tiers[tier])-1]
			p.tiers[tier] = p.tiers[tier][:len(p.tiers[tier])-1]
			return nil
		}
	}

	return ErrSymbolNotInTier
}

// UpgradeToTier0 upgrades symbol to Tier0 (used when WS disconnects)
func (p *RESTPoller) UpgradeToTier0(symbol string) error {
	// Remove from other tiers
	p.RemoveSymbolFromTier(Tier1, symbol)
	p.RemoveSymbolFromTier(Tier2, symbol)

	// Add to Tier0
	return p.AddSymbolToTier(Tier0, symbol)
}

// DowngradeFromTier0 downgrades symbol from Tier0 (used when WS reconnects)
func (p *RESTPoller) DowngradeFromTier0(symbol string, targetTier Tier) error {
	// Remove from Tier0
	if err := p.RemoveSymbolFromTier(Tier0, symbol); err != nil {
		return err
	}

	// Add to target tier
	return p.AddSymbolToTier(targetTier, symbol)
}

// SetTierInterval updates polling interval for a tier
func (p *RESTPoller) SetTierInterval(tier Tier, interval time.Duration) error {
	p.tierMu.Lock()
	defer p.tierMu.Unlock()

	config, exists := p.tierConfigs[tier]
	if !exists {
		return ErrInvalidTier
	}

	config.Interval = interval
	p.tierConfigs[tier] = config

	// Note: This doesn't restart the ticker
	// In production, you'd want to restart the ticker with new interval
	return nil
}

// GetTierStats returns statistics for all tiers
func (p *RESTPoller) GetTierStats() map[Tier]TierStats {
	p.tierMu.RLock()
	defer p.tierMu.RUnlock()

	stats := make(map[Tier]TierStats)
	for tier, config := range p.tierConfigs {
		stats[tier] = TierStats{
			Tier:         tier,
			SymbolCount:  len(p.tiers[tier]),
			MaxSize:      config.MaxSize,
			Interval:     config.Interval,
			UsagePercent: float64(len(p.tiers[tier])) / float64(config.MaxSize) * 100,
		}
	}
	return stats
}

// TierStats represents tier statistics
type TierStats struct {
	Tier         Tier
	SymbolCount  int
	MaxSize      int
	Interval     time.Duration
	UsagePercent float64
}

// GetFallbackStats returns Naver fallback statistics
func (p *RESTPoller) GetFallbackStats() FallbackStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()

	stats := FallbackStats{
		KISFailed:      p.kisFailed,
		NaverFallbacks: p.naverFallbacks,
		NaverSucceeded: p.naverSucceeded,
	}

	// Calculate success rate
	if stats.NaverFallbacks > 0 {
		stats.NaverSuccessRate = float64(stats.NaverSucceeded) / float64(stats.NaverFallbacks) * 100
	}

	return stats
}

// FallbackStats represents Naver fallback statistics
type FallbackStats struct {
	KISFailed        int64   // Total KIS failures
	NaverFallbacks   int64   // Total Naver fallback attempts
	NaverSucceeded   int64   // Successful Naver fallbacks
	NaverSuccessRate float64 // Success rate percentage
}
