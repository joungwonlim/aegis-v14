package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// Cache entry for KIS API responses
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// KISOrdersHandler handles KIS orders API requests
type KISOrdersHandler struct {
	kisAdapter       execution.KISAdapter
	accountID        string
	unfilledCache    *cacheEntry
	filledCache      *cacheEntry
	cacheMu          sync.RWMutex
	cacheDuration    time.Duration
}

// NewKISOrdersHandler creates a new KISOrdersHandler
func NewKISOrdersHandler(kisAdapter execution.KISAdapter, accountID string) *KISOrdersHandler {
	return &KISOrdersHandler{
		kisAdapter:    kisAdapter,
		accountID:     accountID,
		cacheDuration: 5 * time.Second, // Cache for 5 seconds to avoid rate limit
	}
}

// GetUnfilledOrders retrieves unfilled orders from KIS
// GET /api/kis/unfilled-orders
func (h *KISOrdersHandler) GetUnfilledOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check cache first
	h.cacheMu.RLock()
	if h.unfilledCache != nil && time.Now().Before(h.unfilledCache.expiresAt) {
		cached := h.unfilledCache.data.([]*execution.KISUnfilledOrder)
		h.cacheMu.RUnlock()

		// Return cached response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}
	h.cacheMu.RUnlock()

	// Get unfilled orders from KIS
	orders, err := h.kisAdapter.GetUnfilledOrders(ctx, h.accountID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get unfilled orders from KIS")
		http.Error(w, "Failed to get unfilled orders", http.StatusInternalServerError)
		return
	}

	// Update cache
	h.cacheMu.Lock()
	h.unfilledCache = &cacheEntry{
		data:      orders,
		expiresAt: time.Now().Add(h.cacheDuration),
	}
	h.cacheMu.Unlock()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		log.Error().Err(err).Msg("Failed to encode unfilled orders response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetFilledOrders retrieves filled orders from KIS
// GET /api/kis/filled-orders
func (h *KISOrdersHandler) GetFilledOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check cache first
	h.cacheMu.RLock()
	if h.filledCache != nil && time.Now().Before(h.filledCache.expiresAt) {
		cached := h.filledCache.data.([]*execution.KISFill)
		h.cacheMu.RUnlock()

		// Return cached response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}
	h.cacheMu.RUnlock()

	// Get filled orders since midnight
	since := time.Now().Truncate(24 * time.Hour)
	fills, err := h.kisAdapter.GetFills(ctx, h.accountID, since)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get filled orders from KIS")
		http.Error(w, "Failed to get filled orders", http.StatusInternalServerError)
		return
	}

	// Update cache
	h.cacheMu.Lock()
	h.filledCache = &cacheEntry{
		data:      fills,
		expiresAt: time.Now().Add(h.cacheDuration),
	}
	h.cacheMu.Unlock()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(fills); err != nil {
		log.Error().Err(err).Msg("Failed to encode filled orders response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PlaceOrderRequest is the request body for placing an order
type PlaceOrderRequest struct {
	Symbol    string `json:"symbol"`     // 종목코드 (6자리)
	Side      string `json:"side"`       // buy 또는 sell
	OrderType string `json:"order_type"` // limit 또는 market
	Qty       int    `json:"qty"`        // 주문수량
	Price     int    `json:"price"`      // 주문가격 (시장가일 경우 0)
}

// PlaceOrderResponse is the response for placing an order
type PlaceOrderResponse struct {
	Success bool   `json:"success"`
	OrderID string `json:"order_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PlaceOrder places an order to KIS
// POST /api/kis/orders
func (h *KISOrdersHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Failed to decode place order request")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlaceOrderResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Validate request
	if req.Symbol == "" || req.Side == "" || req.OrderType == "" || req.Qty <= 0 {
		log.Warn().Interface("req", req).Msg("Invalid order request parameters")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlaceOrderResponse{
			Success: false,
			Error:   "Invalid parameters: symbol, side, order_type, and qty are required",
		})
		return
	}

	if req.OrderType == "limit" && req.Price <= 0 {
		log.Warn().Interface("req", req).Msg("Invalid limit order: price must be > 0")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlaceOrderResponse{
			Success: false,
			Error:   "Limit order requires price > 0",
		})
		return
	}

	// Submit order to KIS
	kisReq := execution.KISOrderRequest{
		Symbol:    req.Symbol,
		Side:      req.Side,
		OrderType: req.OrderType,
		Qty:       req.Qty,
		Price:     req.Price,
	}

	resp, err := h.kisAdapter.SubmitOrder(ctx, kisReq)
	if err != nil {
		log.Error().Err(err).Interface("req", req).Msg("Failed to submit order to KIS")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlaceOrderResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PlaceOrderResponse{
		Success: true,
		OrderID: resp.OrderID,
	})

	log.Info().
		Str("symbol", req.Symbol).
		Str("side", req.Side).
		Str("order_type", req.OrderType).
		Int("qty", req.Qty).
		Int("price", req.Price).
		Str("order_id", resp.OrderID).
		Msg("Order placed successfully")
}

// KISAdapterProvider interface for getting KIS adapter
type KISAdapterProvider interface {
	GetKISAdapter(ctx context.Context) (execution.KISAdapter, error)
}
