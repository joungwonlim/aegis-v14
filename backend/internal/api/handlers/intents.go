package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// IntentsHandler handles order intents-related API requests
type IntentsHandler struct {
	intentRepo exit.OrderIntentRepository
}

// NewIntentsHandler creates a new IntentsHandler
func NewIntentsHandler(intentRepo exit.OrderIntentRepository) *IntentsHandler {
	return &IntentsHandler{
		intentRepo: intentRepo,
	}
}

// GetIntents retrieves recent order intents
// GET /api/intents
func (h *IntentsHandler) GetIntents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get recent intents (last 100)
	intents, err := h.intentRepo.GetRecentIntents(ctx, 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get intents")
		http.Error(w, "Failed to get intents", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(intents); err != nil {
		log.Error().Err(err).Msg("Failed to encode intents response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
