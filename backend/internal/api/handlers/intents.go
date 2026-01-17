package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// IntentsHandler handles order intents-related API requests
type IntentsHandler struct {
	intentRepo       IntentReader
	intentRepoWriter exit.OrderIntentRepository
}

// NewIntentsHandler creates a new IntentsHandler
func NewIntentsHandler(intentRepo IntentReader, intentRepoWriter exit.OrderIntentRepository) *IntentsHandler {
	return &IntentsHandler{
		intentRepo:       intentRepo,
		intentRepoWriter: intentRepoWriter,
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

// ApproveIntent approves an intent (PENDING_APPROVAL → NEW)
// POST /api/intents/{intent_id}/approve
func (h *IntentsHandler) ApproveIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	intentIDStr := mux.Vars(r)["intent_id"]

	// Parse intent ID
	intentID, err := uuid.Parse(intentIDStr)
	if err != nil {
		http.Error(w, "Invalid intent ID", http.StatusBadRequest)
		return
	}

	// Approve intent
	if err := h.intentRepoWriter.ApproveIntent(ctx, intentID); err != nil {
		log.Error().Err(err).Str("intent_id", intentIDStr).Msg("Failed to approve intent")
		http.Error(w, "Failed to approve intent", http.StatusInternalServerError)
		return
	}

	log.Info().Str("intent_id", intentIDStr).Msg("Intent approved")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// RejectIntent rejects an intent (PENDING_APPROVAL → CANCELLED)
// POST /api/intents/{intent_id}/reject
func (h *IntentsHandler) RejectIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	intentIDStr := mux.Vars(r)["intent_id"]

	// Parse intent ID
	intentID, err := uuid.Parse(intentIDStr)
	if err != nil {
		http.Error(w, "Invalid intent ID", http.StatusBadRequest)
		return
	}

	// Reject intent
	if err := h.intentRepoWriter.RejectIntent(ctx, intentID); err != nil {
		log.Error().Err(err).Str("intent_id", intentIDStr).Msg("Failed to reject intent")
		http.Error(w, "Failed to reject intent", http.StatusInternalServerError)
		return
	}

	log.Info().Str("intent_id", intentIDStr).Msg("Intent rejected")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}
