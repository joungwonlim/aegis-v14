package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// FillsHandler handles fills-related API requests
type FillsHandler struct {
	fillRepo FillReader
}

// NewFillsHandler creates a new FillsHandler
func NewFillsHandler(fillRepo FillReader) *FillsHandler {
	return &FillsHandler{
		fillRepo: fillRepo,
	}
}

// GetFills retrieves recent fills
// GET /api/fills
func (h *FillsHandler) GetFills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get recent fills (last 100)
	fills, err := h.fillRepo.GetRecentFills(ctx, 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get fills")
		http.Error(w, "Failed to get fills", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fills); err != nil {
		log.Error().Err(err).Msg("Failed to encode fills response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
