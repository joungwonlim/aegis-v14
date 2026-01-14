package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// OrdersHandler handles orders-related API requests
type OrdersHandler struct {
	orderRepo OrderReader
}

// NewOrdersHandler creates a new OrdersHandler
func NewOrdersHandler(orderRepo OrderReader) *OrdersHandler {
	return &OrdersHandler{
		orderRepo: orderRepo,
	}
}

// GetOrders retrieves recent orders
// GET /api/orders
func (h *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get recent orders (last 100)
	orders, err := h.orderRepo.GetRecentOrders(ctx, 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get orders")
		http.Error(w, "Failed to get orders", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		log.Error().Err(err).Msg("Failed to encode orders response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
