package pricesync

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/infra/naver"
)

// Manager manages all price sync components
type Manager struct {
	// Core service
	service *Service

	// Data sources
	restPoller  *RESTPoller
	naverClient *naver.Client

	// KIS client
	kisClient *kis.Client

	// State
	isRunning bool
	mu        sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new PriceSync manager
func NewManager(service *Service, kisClient *kis.Client) *Manager {
	return &Manager{
		service:     service,
		kisClient:   kisClient,
		naverClient: naver.NewClient(),
		isRunning:   false,
	}
}

// Start starts all price sync components
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		log.Warn().Msg("PriceSync already running")
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	log.Info().Msg("Starting PriceSync Manager...")

	// 1. Start WebSocket client
	if err := m.startWebSocket(); err != nil {
		log.Error().Err(err).Msg("Failed to start WebSocket")
		// Continue with REST only
	}

	// 2. Start REST poller
	if err := m.startRESTPoller(); err != nil {
		log.Error().Err(err).Msg("Failed to start REST poller")
		return err
	}

	// 3. Naver client is ready (on-demand)
	log.Info().Msg("Naver client ready for fallback")

	m.isRunning = true

	log.Info().Msg("✅ PriceSync Manager started")

	return nil
}

// Stop stops all price sync components
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	log.Info().Msg("Stopping PriceSync Manager...")

	// Cancel context
	if m.cancel != nil {
		m.cancel()
	}

	// Stop WebSocket
	if m.kisClient != nil && m.kisClient.WS != nil {
		m.kisClient.WS.Stop()
	}

	// Stop REST poller
	if m.restPoller != nil {
		m.restPoller.Stop()
	}

	// Wait for all goroutines
	m.wg.Wait()

	m.isRunning = false

	log.Info().Msg("✅ PriceSync Manager stopped")
}

// startWebSocket initializes and starts WebSocket client
func (m *Manager) startWebSocket() error {
	log.Info().Msg("Starting KIS WebSocket...")

	// Set tick handler
	m.kisClient.WS.SetTickHandler(func(tick price.Tick) {
		// Process tick through service
		if err := m.service.ProcessTick(m.ctx, tick); err != nil {
			log.Error().Err(err).Str("symbol", tick.Symbol).Msg("Failed to process WS tick")
		} else {
			log.Debug().Str("symbol", tick.Symbol).Int64("price", tick.LastPrice).Msg("Processed WS tick")
		}
	})

	// Start WebSocket
	if err := m.kisClient.WS.Start(m.ctx); err != nil {
		return err
	}

	// Subscribe to initial symbols (empty for now)
	// TODO: Load from positions/watchlist
	log.Info().Msg("✅ WebSocket started (no initial subscriptions)")

	return nil
}

// startRESTPoller initializes and starts REST poller
func (m *Manager) startRESTPoller() error {
	log.Info().Msg("Starting REST Poller...")

	// Create REST poller
	m.restPoller = NewRESTPoller(m.kisClient.REST, m.service)

	// Start poller
	if err := m.restPoller.Start(m.ctx); err != nil {
		return err
	}

	// Set initial symbols (empty for now)
	// TODO: Load from positions/watchlist
	log.Info().Msg("✅ REST Poller started (no initial symbols)")

	return nil
}

// SubscribeSymbol subscribes to a symbol (WS or REST based on availability)
func (m *Manager) SubscribeSymbol(symbol string, tier Tier) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning {
		return ErrPriceSyncNotRunning
	}

	// Try WebSocket first (if available and space left)
	if m.kisClient.WS != nil && m.kisClient.WS.CanSubscribe() {
		if err := m.kisClient.WS.Subscribe(symbol); err != nil {
			log.Warn().Err(err).Str("symbol", symbol).Msg("WS subscription failed, using REST")
			// Fallback to REST
			return m.restPoller.AddSymbolToTier(tier, symbol)
		}
		log.Info().Str("symbol", symbol).Msg("Subscribed via WebSocket")
		return nil
	}

	// Use REST poller
	if err := m.restPoller.AddSymbolToTier(tier, symbol); err != nil {
		return err
	}

	log.Info().Str("symbol", symbol).Int("tier", int(tier)).Msg("Added to REST poller")
	return nil
}

// UnsubscribeSymbol unsubscribes from a symbol
func (m *Manager) UnsubscribeSymbol(symbol string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning {
		return ErrPriceSyncNotRunning
	}

	// Try to unsubscribe from WebSocket
	if m.kisClient.WS != nil {
		if err := m.kisClient.WS.Unsubscribe(symbol); err == nil {
			log.Info().Str("symbol", symbol).Msg("Unsubscribed from WebSocket")
			return nil
		}
	}

	// Remove from all REST tiers
	m.restPoller.RemoveSymbolFromTier(Tier0, symbol)
	m.restPoller.RemoveSymbolFromTier(Tier1, symbol)
	m.restPoller.RemoveSymbolFromTier(Tier2, symbol)

	log.Info().Str("symbol", symbol).Msg("Removed from REST poller")
	return nil
}

// GetTierStats returns statistics for all tiers
func (m *Manager) GetTierStats() map[Tier]TierStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.restPoller == nil {
		return nil
	}

	return m.restPoller.GetTierStats()
}

// GetWSSubscriptionCount returns current WebSocket subscription count
func (m *Manager) GetWSSubscriptionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.kisClient == nil || m.kisClient.WS == nil {
		return 0
	}

	return m.kisClient.WS.GetSubscriptionCount()
}

// IsRunning returns whether PriceSync is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// HealthCheck performs health check
func (m *Manager) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning {
		return ErrPriceSyncNotRunning
	}

	// TODO: Implement detailed health checks
	// - Check WS connection state
	// - Check freshness stats
	// - Check REST poller status

	return nil
}

// monitorHealth monitors overall PriceSync health (background goroutine)
func (m *Manager) monitorHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// TODO: Implement health monitoring
			// - Check freshness
			// - Check stale symbols
			// - Trigger alerts if needed
			log.Debug().Msg("Health check (placeholder)")
		}
	}
}
