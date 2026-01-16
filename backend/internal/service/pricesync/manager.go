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

	// Priority management
	priorityManager *PriorityManager

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
func NewManager(service *Service, kisClient *kis.Client, priorityManager *PriorityManager) *Manager {
	return &Manager{
		service:         service,
		kisClient:       kisClient,
		priorityManager: priorityManager,
		naverClient:     naver.NewClient(),
		isRunning:       false,
	}
}

// SetPriorityManager sets or updates the priority manager
func (m *Manager) SetPriorityManager(pm *PriorityManager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.priorityManager = pm
	log.Info().Msg("PriorityManager configured")
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

	log.Info().Msg("✅ WebSocket started")

	return nil
}

// startRESTPoller initializes and starts REST poller
func (m *Manager) startRESTPoller() error {
	log.Info().Msg("Starting REST Poller...")

	// Create REST poller with Naver fallback
	m.restPoller = NewRESTPoller(m.kisClient.REST, m.naverClient, m.service)

	// Start poller
	if err := m.restPoller.Start(m.ctx); err != nil {
		return err
	}

	log.Info().Msg("✅ REST Poller started")

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

// TriggerRefresh immediately fetches price for a symbol (used for execution notifications)
func (m *Manager) TriggerRefresh(symbol string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isRunning || m.restPoller == nil {
		log.Warn().Str("symbol", symbol).Msg("TriggerRefresh: PriceSync not running")
		return
	}

	// Fetch price in background to avoid blocking
	go func() {
		if err := m.restPoller.FetchSymbolPrice(symbol); err != nil {
			log.Warn().Err(err).Str("symbol", symbol).Msg("TriggerRefresh failed")
		}
	}()
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

// RefreshSubscriptions refreshes all subscriptions based on PriorityManager
// This should be called:
// 1. On startup (after Start)
// 2. Periodically (every 5 minutes)
// 3. On events (position/order changes)
func (m *Manager) RefreshSubscriptions(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return ErrPriceSyncNotRunning
	}

	if m.priorityManager == nil {
		log.Warn().Msg("PriorityManager not configured, skipping subscription refresh")
		return nil
	}

	log.Info().Msg("Refreshing subscriptions from PriorityManager...")

	// 1. Refresh priorities
	if err := m.priorityManager.Refresh(ctx); err != nil {
		return err
	}

	// 2. Get WS symbols (top 40)
	wsSymbols := m.priorityManager.GetWSSymbols()

	// 3. Get REST tier symbols
	tier0Symbols := m.priorityManager.GetTier0Symbols()
	tier1Symbols := m.priorityManager.GetTier1Symbols()
	tier2Symbols := m.priorityManager.GetTier2Symbols()

	// 4. Update WS subscriptions
	if m.kisClient.WS != nil {
		// Get current subscriptions
		currentWS := m.kisClient.WS.GetSubscriptions()

		// Find symbols to subscribe (in new list but not in current)
		toSubscribe := difference(wsSymbols, currentWS)

		// Find symbols to unsubscribe (in current but not in new list)
		toUnsubscribe := difference(currentWS, wsSymbols)

		// Subscribe new symbols
		for _, symbol := range toSubscribe {
			if err := m.kisClient.WS.Subscribe(symbol); err != nil {
				log.Warn().Err(err).Str("symbol", symbol).Msg("Failed to subscribe WS")
			}
		}

		// Unsubscribe removed symbols
		for _, symbol := range toUnsubscribe {
			if err := m.kisClient.WS.Unsubscribe(symbol); err != nil {
				log.Warn().Err(err).Str("symbol", symbol).Msg("Failed to unsubscribe WS")
			}
		}

		log.Info().
			Int("ws_total", len(wsSymbols)).
			Int("subscribed", len(toSubscribe)).
			Int("unsubscribed", len(toUnsubscribe)).
			Msg("WS subscriptions updated")
	}

	// 5. Update REST tiers
	// Strategy:
	// - Tier0 (3s): High priority non-WS symbols only (holdings without WS, closing positions)
	// - Tier1 (10s): WS backup + watchlist (WS symbols need backup but not ultra-fast)
	// - Tier2 (30s): Universe / low priority symbols
	if m.restPoller != nil {
		// Tier0: Only non-WS high priority symbols
		if err := m.restPoller.SetTierSymbols(Tier0, tier0Symbols); err != nil {
			log.Error().Err(err).Msg("Failed to set Tier0 symbols")
		}

		// Tier1: WS symbols as backup + original tier1 symbols
		// This provides redundancy at 10s interval (acceptable for WS backup)
		allTier1Symbols := make([]string, 0, len(wsSymbols)+len(tier1Symbols))
		allTier1Symbols = append(allTier1Symbols, wsSymbols...)
		allTier1Symbols = append(allTier1Symbols, tier1Symbols...)
		if err := m.restPoller.SetTierSymbols(Tier1, allTier1Symbols); err != nil {
			log.Error().Err(err).Msg("Failed to set Tier1 symbols")
		}

		if err := m.restPoller.SetTierSymbols(Tier2, tier2Symbols); err != nil {
			log.Error().Err(err).Msg("Failed to set Tier2 symbols")
		}

		log.Info().
			Int("tier0", len(tier0Symbols)).
			Int("tier1", len(allTier1Symbols)).
			Int("tier1_ws_backup", len(wsSymbols)).
			Int("tier2", len(tier2Symbols)).
			Msg("REST tiers updated (WS symbols in Tier1 as backup)")
	}

	log.Info().Msg("✅ Subscriptions refreshed successfully")

	return nil
}

// InitializeSubscriptions initializes subscriptions on startup
// This is called by Runtime after Start()
func (m *Manager) InitializeSubscriptions(ctx context.Context) error {
	log.Info().Msg("Initializing subscriptions from positions/watchlist...")

	// First refresh
	if err := m.RefreshSubscriptions(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to initialize subscriptions")
		return err
	}

	// Start periodic refresh (every 5 minutes)
	m.wg.Add(1)
	go m.periodicRefresh()

	return nil
}

// periodicRefresh refreshes subscriptions periodically
func (m *Manager) periodicRefresh() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.RefreshSubscriptions(m.ctx); err != nil {
				log.Error().Err(err).Msg("Periodic subscription refresh failed")
			}
		}
	}
}

// difference returns elements in a that are not in b
func difference(a, b []string) []string {
	bSet := make(map[string]bool)
	for _, v := range b {
		bSet[v] = true
	}

	var diff []string
	for _, v := range a {
		if !bSet[v] {
			diff = append(diff, v)
		}
	}
	return diff
}
