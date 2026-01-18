package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// ==============================================================================
// PriceStreamHandler - SSE endpoint for real-time price updates
// ==============================================================================

// PriceStreamHandler handles Server-Sent Events for price streaming
type PriceStreamHandler struct {
	broker *pricesync.Broker
	cache  *pricesync.PriceCache
}

// NewPriceStreamHandler creates a new price stream handler
func NewPriceStreamHandler(broker *pricesync.Broker, cache *pricesync.PriceCache) *PriceStreamHandler {
	return &PriceStreamHandler{
		broker: broker,
		cache:  cache,
	}
}

// ==============================================================================
// SSE Endpoints
// ==============================================================================

// StreamPrices streams prices for specified symbols via SSE
// GET /api/v1/prices/stream?symbols=005930,000660,035720
func (h *PriceStreamHandler) StreamPrices(w http.ResponseWriter, r *http.Request) {
	// Parse symbols from query
	symbolsParam := r.URL.Query().Get("symbols")
	if symbolsParam == "" {
		http.Error(w, "symbols parameter required", http.StatusBadRequest)
		return
	}

	symbols := parseSymbols(symbolsParam)
	if len(symbols) == 0 {
		http.Error(w, "at least one symbol required", http.StatusBadRequest)
		return
	}

	// Limit max symbols per connection
	const maxSymbols = 100
	if len(symbols) > maxSymbols {
		http.Error(w, fmt.Sprintf("max %d symbols allowed", maxSymbols), http.StatusBadRequest)
		return
	}

	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial prices from cache
	if h.cache != nil {
		initialPrices := h.cache.GetMultiple(symbols)
		for _, cached := range initialPrices {
			update := pricesync.PriceUpdate{
				Symbol:      cached.Symbol,
				Price:       cached.BestPrice,
				ChangePrice: cached.ChangePrice,
				ChangeRate:  cached.ChangeRate,
				Volume:      cached.Volume,
				Source:      cached.Source,
				Timestamp:   cached.Timestamp,
				IsStale:     cached.IsStale,
			}
			h.sendEvent(w, "price", update)
		}
		flusher.Flush()
	}

	// Subscribe to updates
	sub := h.broker.SubscribeMultiple(symbols)
	defer h.broker.Unsubscribe(sub)

	log.Info().
		Strs("symbols", symbols).
		Str("remote", r.RemoteAddr).
		Msg("SSE: client connected")

	// Keep-alive ticker (send comment every 30s to prevent timeout)
	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	// Stream updates
	for {
		select {
		case <-r.Context().Done():
			log.Info().
				Str("remote", r.RemoteAddr).
				Msg("SSE: client disconnected")
			return

		case update, ok := <-sub.C:
			if !ok {
				// Channel closed
				return
			}
			h.sendEvent(w, "price", update)
			flusher.Flush()

		case <-keepAlive.C:
			// Send keep-alive comment
			fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}

// StreamAllPrices streams all price updates via SSE
// GET /api/v1/prices/stream/all
// WARNING: High traffic - use for monitoring only
func (h *PriceStreamHandler) StreamAllPrices(w http.ResponseWriter, r *http.Request) {
	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to all updates
	sub := h.broker.SubscribeAll()
	defer h.broker.Unsubscribe(sub)

	log.Info().
		Str("remote", r.RemoteAddr).
		Msg("SSE: all-prices client connected")

	keepAlive := time.NewTicker(30 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Info().
				Str("remote", r.RemoteAddr).
				Msg("SSE: all-prices client disconnected")
			return

		case update, ok := <-sub.C:
			if !ok {
				return
			}
			h.sendEvent(w, "price", update)
			flusher.Flush()

		case <-keepAlive.C:
			fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}

// ==============================================================================
// REST Endpoints (for compatibility)
// ==============================================================================

// GetBestPrices returns best prices for multiple symbols
// GET /api/v1/prices/best?symbols=005930,000660
func (h *PriceStreamHandler) GetBestPrices(w http.ResponseWriter, r *http.Request) {
	symbolsParam := r.URL.Query().Get("symbols")
	if symbolsParam == "" {
		h.jsonError(w, "symbols parameter required", http.StatusBadRequest)
		return
	}

	symbols := parseSymbols(symbolsParam)
	if len(symbols) == 0 {
		h.jsonError(w, "at least one symbol required", http.StatusBadRequest)
		return
	}

	// Get from cache (or load from DB)
	prices, err := h.cache.GetMultipleOrLoad(r.Context(), symbols)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	result := make(map[string]interface{})
	for symbol, cached := range prices {
		result[symbol] = map[string]interface{}{
			"price":        cached.BestPrice,
			"change_price": cached.ChangePrice,
			"change_rate":  cached.ChangeRate,
			"volume":       cached.Volume,
			"source":       cached.Source,
			"timestamp":    cached.Timestamp,
			"is_stale":     cached.IsStale,
		}
	}

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// GetBrokerStats returns broker statistics
// GET /api/v1/prices/stats
func (h *PriceStreamHandler) GetBrokerStats(w http.ResponseWriter, r *http.Request) {
	brokerStats := h.broker.GetStats()
	cacheStats := h.cache.GetStats()

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"broker": brokerStats,
			"cache":  cacheStats,
		},
	})
}

// ==============================================================================
// Helper Methods
// ==============================================================================

// sendEvent sends an SSE event
func (h *PriceStreamHandler) sendEvent(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("SSE: failed to marshal event")
		return
	}

	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// jsonResponse writes JSON response
func (h *PriceStreamHandler) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes JSON error response
func (h *PriceStreamHandler) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// parseSymbols parses comma-separated symbols
func parseSymbols(s string) []string {
	parts := strings.Split(s, ",")
	symbols := make([]string, 0, len(parts))

	for _, part := range parts {
		symbol := strings.TrimSpace(part)
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}
