package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// HoldingsHandler handles holdings-related API requests
type HoldingsHandler struct {
	holdingRepo HoldingReader
}

// NewHoldingsHandler creates a new HoldingsHandler
func NewHoldingsHandler(holdingRepo HoldingReader) *HoldingsHandler {
	return &HoldingsHandler{
		holdingRepo: holdingRepo,
	}
}

// GetHoldings retrieves all holdings
// GET /api/holdings
func (h *HoldingsHandler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all holdings
	holdings, err := h.holdingRepo.GetAllHoldings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get holdings")
		http.Error(w, "Failed to get holdings", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(holdings); err != nil {
		log.Error().Err(err).Msg("Failed to encode holdings response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
