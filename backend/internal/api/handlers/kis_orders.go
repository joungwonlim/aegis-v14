package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
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

	// Store stale cache for rate limit fallback
	var staleCache []*execution.KISUnfilledOrder
	if h.unfilledCache != nil {
		staleCache = h.unfilledCache.data.([]*execution.KISUnfilledOrder)
	}
	h.cacheMu.RUnlock()

	// Get unfilled orders from KIS
	orders, err := h.kisAdapter.GetUnfilledOrders(ctx, h.accountID)
	if err != nil {
		// Check if it's a rate limit error (holdUntil period)
		errMsg := err.Error()
		if isRateLimitError(errMsg) {
			// Rate limit - use stale cache if available
			if staleCache != nil {
				log.Debug().Str("account_id", h.accountID).Msg("KIS rate limit, returning stale cache")
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "STALE")
				json.NewEncoder(w).Encode(staleCache)
				return
			}

			// No cache available - return rate limit error
			log.Warn().Str("account_id", h.accountID).Msg("KIS rate limit, no cache available")
			http.Error(w, "KIS rate limit, please retry later", http.StatusTooManyRequests)
			return
		}

		log.Error().Err(err).Str("account_id", h.accountID).Msg("Failed to get unfilled orders from KIS")
		// Return more specific error message for debugging
		http.Error(w, fmt.Sprintf("Failed to get unfilled orders: %s", errMsg), http.StatusInternalServerError)
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

	// Store stale cache for rate limit fallback
	var staleCache []*execution.KISFill
	if h.filledCache != nil {
		staleCache = h.filledCache.data.([]*execution.KISFill)
	}
	h.cacheMu.RUnlock()

	// Get filled orders since midnight
	since := time.Now().Truncate(24 * time.Hour)
	fills, err := h.kisAdapter.GetFills(ctx, h.accountID, since)
	if err != nil {
		// Check if it's a rate limit error (holdUntil period)
		errMsg := err.Error()
		if isRateLimitError(errMsg) {
			// Rate limit - use stale cache if available
			if staleCache != nil {
				log.Debug().Str("account_id", h.accountID).Msg("KIS rate limit, returning stale cache")
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "STALE")
				json.NewEncoder(w).Encode(staleCache)
				return
			}

			// No cache available - return rate limit error
			log.Warn().Str("account_id", h.accountID).Msg("KIS rate limit, no cache available")
			http.Error(w, "KIS rate limit, please retry later", http.StatusTooManyRequests)
			return
		}

		log.Error().Err(err).Str("account_id", h.accountID).Msg("Failed to get filled orders from KIS")
		http.Error(w, fmt.Sprintf("Failed to get filled orders: %s", errMsg), http.StatusInternalServerError)
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
	var limitPrice *decimal.Decimal
	if req.OrderType == "limit" && req.Price > 0 {
		price := decimal.NewFromInt(int64(req.Price))
		limitPrice = &price
	}

	kisReq := execution.KISOrderRequest{
		AccountID:  h.accountID,
		Symbol:     req.Symbol,
		Side:       req.Side,
		OrderType:  req.OrderType,
		Qty:        int64(req.Qty),
		LimitPrice: limitPrice,
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

// CancelOrderRequest is the request body for cancelling an order
type CancelOrderRequest struct {
	OrderNo string `json:"order_no"` // 주문번호
}

// CancelOrderResponse is the response for cancelling an order
type CancelOrderResponse struct {
	Success  bool   `json:"success"`
	CancelNo string `json:"cancel_no,omitempty"` // 취소주문번호
	Error    string `json:"error,omitempty"`
}

// CancelOrder cancels an order in KIS
// DELETE /api/kis/orders/{order_no}
func (h *KISOrdersHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get order_no from URL path
	orderNo := chi.URLParam(r, "order_no")
	if orderNo == "" {
		log.Warn().Msg("Order number is required for cancel")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CancelOrderResponse{
			Success: false,
			Error:   "Order number is required",
		})
		return
	}

	// Cancel order in KIS
	resp, err := h.kisAdapter.CancelOrder(ctx, h.accountID, orderNo)
	if err != nil {
		log.Error().Err(err).Str("order_no", orderNo).Msg("Failed to cancel order in KIS")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CancelOrderResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Invalidate unfilled orders cache
	h.cacheMu.Lock()
	h.unfilledCache = nil
	h.cacheMu.Unlock()

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CancelOrderResponse{
		Success:  true,
		CancelNo: resp.CancelNo,
	})

	log.Info().
		Str("order_no", orderNo).
		Str("cancel_no", resp.CancelNo).
		Msg("Order cancelled successfully")
}

// KISAdapterProvider interface for getting KIS adapter
type KISAdapterProvider interface {
	GetKISAdapter(ctx context.Context) (execution.KISAdapter, error)
}

// isRateLimitError checks if the error is a rate limit error
func isRateLimitError(errMsg string) bool {
	return strings.Contains(errMsg, "token refresh on hold") ||
		strings.Contains(errMsg, "KIS rate limit") ||
		strings.Contains(errMsg, "EGW00133")
}
