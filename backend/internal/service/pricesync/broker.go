package pricesync

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// ==============================================================================
// Broker - In-memory Pub/Sub for price updates
// ==============================================================================

// Broker distributes price updates to subscribers
// Uses Go channels for non-blocking pub/sub
type Broker struct {
	mu sync.RWMutex

	// Symbol-specific subscribers
	subscribers map[string]map[*Subscription]bool // symbol → set of subscriptions

	// All-symbol subscribers (monitoring/debugging)
	allSubs map[*Subscription]bool

	// Configuration
	channelSize int // buffer size for subscription channels

	// Metrics
	published    int64
	delivered    int64
	dropped      int64
	activeSyms   int
	activeSubs   int
}

// Subscription represents a price update subscription
type Subscription struct {
	C      chan PriceUpdate // Channel to receive updates
	Symbol string           // Specific symbol or "*" for all
}

// PriceUpdate represents a price update event
type PriceUpdate struct {
	Symbol      string       `json:"symbol"`
	Price       int64        `json:"price"`
	ChangePrice *int64       `json:"change_price,omitempty"`
	ChangeRate  *float64     `json:"change_rate,omitempty"`
	Volume      *int64       `json:"volume,omitempty"`
	BidPrice    *int64       `json:"bid_price,omitempty"`
	AskPrice    *int64       `json:"ask_price,omitempty"`
	Source      price.Source `json:"source"`
	Timestamp   time.Time    `json:"timestamp"`
	IsStale     bool         `json:"is_stale,omitempty"`
}

// BrokerConfig holds broker configuration
type BrokerConfig struct {
	ChannelSize int // buffer size for subscription channels (default: 100)
}

// DefaultBrokerConfig returns default configuration
func DefaultBrokerConfig() BrokerConfig {
	return BrokerConfig{
		ChannelSize: 100,
	}
}

// NewBroker creates a new broker
func NewBroker(config BrokerConfig) *Broker {
	if config.ChannelSize <= 0 {
		config.ChannelSize = 100
	}

	return &Broker{
		subscribers: make(map[string]map[*Subscription]bool),
		allSubs:     make(map[*Subscription]bool),
		channelSize: config.ChannelSize,
	}
}

// ==============================================================================
// Subscribe Methods
// ==============================================================================

// Subscribe creates a subscription for a specific symbol
func (b *Broker) Subscribe(symbol string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &Subscription{
		C:      make(chan PriceUpdate, b.channelSize),
		Symbol: symbol,
	}

	// Add to symbol-specific subscribers
	if _, ok := b.subscribers[symbol]; !ok {
		b.subscribers[symbol] = make(map[*Subscription]bool)
		b.activeSyms++
	}
	b.subscribers[symbol][sub] = true
	b.activeSubs++

	log.Debug().
		Str("symbol", symbol).
		Int("total_subs", b.activeSubs).
		Msg("Broker: new subscription")

	return sub
}

// SubscribeMultiple creates subscriptions for multiple symbols
// Returns a single merged channel
func (b *Broker) SubscribeMultiple(symbols []string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create merged subscription
	sub := &Subscription{
		C:      make(chan PriceUpdate, b.channelSize*len(symbols)),
		Symbol: "*multiple*",
	}

	// Add to each symbol's subscribers
	for _, symbol := range symbols {
		if _, ok := b.subscribers[symbol]; !ok {
			b.subscribers[symbol] = make(map[*Subscription]bool)
			b.activeSyms++
		}
		b.subscribers[symbol][sub] = true
	}
	b.activeSubs++

	log.Debug().
		Int("symbol_count", len(symbols)).
		Int("total_subs", b.activeSubs).
		Msg("Broker: new multi-symbol subscription")

	return sub
}

// SubscribeAll creates a subscription for all symbols
func (b *Broker) SubscribeAll() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &Subscription{
		C:      make(chan PriceUpdate, b.channelSize*10), // Larger buffer for all
		Symbol: "*",
	}

	b.allSubs[sub] = true
	b.activeSubs++

	log.Debug().
		Int("total_subs", b.activeSubs).
		Msg("Broker: new all-symbol subscription")

	return sub
}

// Unsubscribe removes a subscription
func (b *Broker) Unsubscribe(sub *Subscription) {
	if sub == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from all-symbol subscribers
	if sub.Symbol == "*" {
		delete(b.allSubs, sub)
		b.activeSubs--
		close(sub.C)
		return
	}

	// Remove from symbol-specific subscribers
	if sub.Symbol == "*multiple*" {
		// Multi-symbol subscription: need to check all symbols
		for symbol, subs := range b.subscribers {
			if subs[sub] {
				delete(subs, sub)
				if len(subs) == 0 {
					delete(b.subscribers, symbol)
					b.activeSyms--
				}
			}
		}
	} else {
		// Single symbol subscription
		if subs, ok := b.subscribers[sub.Symbol]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(b.subscribers, sub.Symbol)
				b.activeSyms--
			}
		}
	}

	b.activeSubs--
	close(sub.C)

	log.Debug().
		Str("symbol", sub.Symbol).
		Int("total_subs", b.activeSubs).
		Msg("Broker: unsubscribed")
}

// ==============================================================================
// Publish Methods
// ==============================================================================

// Publish publishes a price update to all subscribers
func (b *Broker) Publish(update PriceUpdate) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.published++

	// Send to symbol-specific subscribers
	if subs, ok := b.subscribers[update.Symbol]; ok {
		for sub := range subs {
			b.sendToSubscriber(sub, update)
		}
	}

	// Send to all-symbol subscribers
	for sub := range b.allSubs {
		b.sendToSubscriber(sub, update)
	}
}

// PublishFromTick publishes a price update from a tick
func (b *Broker) PublishFromTick(tick price.Tick) {
	update := PriceUpdate{
		Symbol:      tick.Symbol,
		Price:       tick.LastPrice,
		ChangePrice: tick.ChangePrice,
		ChangeRate:  tick.ChangeRate,
		Volume:      tick.Volume,
		BidPrice:    tick.BidPrice,
		AskPrice:    tick.AskPrice,
		Source:      tick.Source,
		Timestamp:   tick.TS,
		IsStale:     false,
	}

	b.Publish(update)
}

// PublishFromCache publishes a price update from cached price
func (b *Broker) PublishFromCache(cached *CachedPrice) {
	update := PriceUpdate{
		Symbol:      cached.Symbol,
		Price:       cached.BestPrice,
		ChangePrice: cached.ChangePrice,
		ChangeRate:  cached.ChangeRate,
		Volume:      cached.Volume,
		BidPrice:    cached.BidPrice,
		AskPrice:    cached.AskPrice,
		Source:      cached.Source,
		Timestamp:   cached.Timestamp,
		IsStale:     cached.IsStale,
	}

	b.Publish(update)
}

// sendToSubscriber sends update to a subscriber (non-blocking)
func (b *Broker) sendToSubscriber(sub *Subscription, update PriceUpdate) {
	select {
	case sub.C <- update:
		b.delivered++
	default:
		// Channel full - drop message (slow subscriber)
		b.dropped++
	}
}

// ==============================================================================
// Info Methods
// ==============================================================================

// GetStats returns broker statistics
func (b *Broker) GetStats() BrokerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	dropRate := float64(0)
	if b.published > 0 {
		dropRate = float64(b.dropped) / float64(b.published) * 100
	}

	return BrokerStats{
		ActiveSymbols:      b.activeSyms,
		ActiveSubscribers:  b.activeSubs,
		TotalPublished:     b.published,
		TotalDelivered:     b.delivered,
		TotalDropped:       b.dropped,
		DropRate:           dropRate,
	}
}

// BrokerStats holds broker statistics
type BrokerStats struct {
	ActiveSymbols     int
	ActiveSubscribers int
	TotalPublished    int64
	TotalDelivered    int64
	TotalDropped      int64
	DropRate          float64 // percentage
}

// GetSubscribedSymbols returns list of subscribed symbols
func (b *Broker) GetSubscribedSymbols() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	symbols := make([]string, 0, len(b.subscribers))
	for symbol := range b.subscribers {
		symbols = append(symbols, symbol)
	}

	return symbols
}

// HasSubscribers returns whether a symbol has any subscribers
func (b *Broker) HasSubscribers(symbol string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, ok := b.subscribers[symbol]; ok {
		return len(subs) > 0
	}

	// All-symbol subscribers count
	return len(b.allSubs) > 0
}

// Close closes all subscriptions
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all subscriptions
	for _, subs := range b.subscribers {
		for sub := range subs {
			close(sub.C)
		}
	}

	for sub := range b.allSubs {
		close(sub.C)
	}

	b.subscribers = make(map[string]map[*Subscription]bool)
	b.allSubs = make(map[*Subscription]bool)
	b.activeSyms = 0
	b.activeSubs = 0

	log.Info().Msg("Broker closed")
}
