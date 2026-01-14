package execution

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// FillHandler handles fill-related HTTP requests
type FillHandler struct {
	fillRepo execution.FillRepository
}

// NewFillHandler creates a new fill handler
func NewFillHandler(fillRepo execution.FillRepository) *FillHandler {
	return &FillHandler{
		fillRepo: fillRepo,
	}
}

// GetFillsForOrder handles GET /api/v1/execution/orders/{orderId}/fills
func (h *FillHandler) GetFillsForOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["orderId"]

	if orderID == "" {
		http.Error(w, "order_id required", http.StatusBadRequest)
		return
	}

	// Load fills
	fills, err := h.fillRepo.LoadFills(r.Context(), orderID)
	if err != nil {
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to load fills")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return fills
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fills": fills,
		"count": len(fills),
	})
}
