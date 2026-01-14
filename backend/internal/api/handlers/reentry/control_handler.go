package reentry

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// ControlHandler handles reentry control endpoints
type ControlHandler struct {
	service ControlService
}

// ControlService interface for service layer
type ControlService interface {
	GetControl() (*reentry.ReentryControl, error)
	UpdateControl(control *reentry.ReentryControl) error
}

// NewControlHandler creates a new control handler
func NewControlHandler(service ControlService) *ControlHandler {
	return &ControlHandler{
		service: service,
	}
}

// UpdateControlRequest represents the request for updating control
type UpdateControlRequest struct {
	Mode      string  `json:"mode"`       // NORMAL, PAUSE_ENTRY, PAUSE_ALL
	Reason    *string `json:"reason"`     // Optional reason
	UpdatedBy string  `json:"updated_by"` // User/system identifier
}

// GetControl handles GET /api/v1/reentry/control
func (h *ControlHandler) GetControl(w http.ResponseWriter, r *http.Request) {
	control, err := h.service.GetControl()
	if err != nil {
		if err == reentry.ErrControlNotFound {
			http.Error(w, "Control not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msg("Failed to get control")
		http.Error(w, "Failed to get control", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(control); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateControl handles PUT /api/v1/reentry/control
func (h *ControlHandler) UpdateControl(w http.ResponseWriter, r *http.Request) {
	var req UpdateControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate mode
	validModes := map[string]bool{
		reentry.ControlModeRunning:    true,
		reentry.ControlModePauseEntry: true,
		reentry.ControlModePauseAll:   true,
	}

	if !validModes[req.Mode] {
		http.Error(w, "Invalid control mode", http.StatusBadRequest)
		return
	}

	// Get current control to preserve ID and updated_ts
	current, err := h.service.GetControl()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get current control")
		http.Error(w, "Failed to get current control", http.StatusInternalServerError)
		return
	}

	// Update control
	current.Mode = req.Mode
	current.Reason = req.Reason
	current.UpdatedBy = req.UpdatedBy

	if err := h.service.UpdateControl(current); err != nil {
		log.Error().Err(err).Msg("Failed to update control")
		http.Error(w, "Failed to update control", http.StatusInternalServerError)
		return
	}

	// Return updated control
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(current); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
