package execution

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// ExitEventHandler handles exit event-related HTTP requests
type ExitEventHandler struct {
	exitEventRepo execution.ExitEventRepository
}

// NewExitEventHandler creates a new exit event handler
func NewExitEventHandler(exitEventRepo execution.ExitEventRepository) *ExitEventHandler {
	return &ExitEventHandler{
		exitEventRepo: exitEventRepo,
	}
}

// GetExitEvent handles GET /api/v1/execution/exit-events/{exitEventId}
func (h *ExitEventHandler) GetExitEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	exitEventIDStr := vars["exitEventId"]

	if exitEventIDStr == "" {
		http.Error(w, "exit_event_id required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	exitEventID, err := uuid.Parse(exitEventIDStr)
	if err != nil {
		http.Error(w, "invalid exit_event_id", http.StatusBadRequest)
		return
	}

	// Get exit event
	exitEvent, err := h.exitEventRepo.GetExitEvent(r.Context(), exitEventID)
	if err != nil {
		if err == execution.ErrExitEventNotFound {
			http.Error(w, "exit event not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("exit_event_id", exitEventIDStr).Msg("Failed to get exit event")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return exit event
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exitEvent)
}

// ListExitEvents handles GET /api/v1/execution/exit-events
func (h *ExitEventHandler) ListExitEvents(w http.ResponseWriter, r *http.Request) {
	// Get since parameter (default: last 24 hours)
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().Add(-24 * time.Hour)

	if sinceStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			since = parsedTime
		}
	}

	// Load exit events
	exitEvents, err := h.exitEventRepo.LoadExitEventsSince(r.Context(), since)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load exit events")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return exit events
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exit_events": exitEvents,
		"count":       len(exitEvents),
		"since":       since.Format(time.RFC3339),
	})
}
