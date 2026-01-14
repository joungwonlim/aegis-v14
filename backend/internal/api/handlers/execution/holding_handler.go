package execution

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// HoldingHandler handles holding-related HTTP requests
type HoldingHandler struct {
	holdingRepo execution.HoldingRepository
}

// NewHoldingHandler creates a new holding handler
func NewHoldingHandler(holdingRepo execution.HoldingRepository) *HoldingHandler {
	return &HoldingHandler{
		holdingRepo: holdingRepo,
	}
}

// ListHoldings handles GET /api/v1/execution/holdings
func (h *HoldingHandler) ListHoldings(w http.ResponseWriter, r *http.Request) {
	// Get account_id from query parameter (or use default)
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		accountID = "default" // TODO: get from config or auth
	}

	// Load holdings
	holdings, err := h.holdingRepo.LoadHoldings(r.Context(), accountID)
	if err != nil {
		log.Error().Err(err).Str("account_id", accountID).Msg("Failed to load holdings")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return holdings
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"holdings": holdings,
		"count":    len(holdings),
	})
}
