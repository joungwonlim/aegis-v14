package execution

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderRepo execution.OrderRepository
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderRepo execution.OrderRepository) *OrderHandler {
	return &OrderHandler{
		orderRepo: orderRepo,
	}
}

// GetOrder handles GET /api/v1/execution/orders/{orderId}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderId"]

	if orderID == "" {
		http.Error(w, "order_id required", http.StatusBadRequest)
		return
	}

	// Get order
	order, err := h.orderRepo.GetOrder(r.Context(), orderID)
	if err != nil {
		if err == execution.ErrOrderNotFound {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to get order")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return order
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// ListOpenOrders handles GET /api/v1/execution/orders/open
func (h *OrderHandler) ListOpenOrders(w http.ResponseWriter, r *http.Request) {
	// Load open orders
	orders, err := h.orderRepo.LoadOpenOrders(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to load open orders")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return orders
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"count":  len(orders),
	})
}
