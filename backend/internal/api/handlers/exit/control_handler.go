package exit

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	exitService "github.com/wonny/aegis/v14/internal/service/exit"
)

// ControlHandler handles exit control endpoints
type ControlHandler struct {
	exitSvc *exitService.Service
}

// NewControlHandler creates a new control handler
func NewControlHandler(exitSvc *exitService.Service) *ControlHandler {
	return &ControlHandler{
		exitSvc: exitSvc,
	}
}

// GetControlRequest represents GET /api/v1/exit/control request
type GetControlResponse struct {
	Mode      string  `json:"mode"`
	Reason    *string `json:"reason"`
	UpdatedBy string  `json:"updated_by"`
	UpdatedTS string  `json:"updated_ts"`
}

// UpdateControlRequest represents POST /api/v1/exit/control request
type UpdateControlRequest struct {
	Mode      string  `json:"mode"`
	Reason    *string `json:"reason"`
	UpdatedBy string  `json:"updated_by"`
}

// GetControl handles GET /api/v1/exit/control
func (h *ControlHandler) GetControl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get control from repository (via service)
	// TODO: Service에 GetControl 메서드 추가 필요
	// For now, return placeholder
	control, err := h.exitSvc.GetControl(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get exit control")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := GetControlResponse{
		Mode:      control.Mode,
		Reason:    control.Reason,
		UpdatedBy: control.UpdatedBy,
		UpdatedTS: control.UpdatedTS.Format("2006-01-02T15:04:05-07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// UpdateControl handles POST /api/v1/exit/control
func (h *ControlHandler) UpdateControl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req UpdateControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate mode
	validModes := map[string]bool{
		"RUNNING":           true,
		"PAUSE_PROFIT":      true,
		"PAUSE_ALL":         true,
		"EMERGENCY_FLATTEN": true,
	}
	if !validModes[req.Mode] {
		log.Warn().Str("mode", req.Mode).Msg("Invalid control mode")
		http.Error(w, "Invalid control mode", http.StatusBadRequest)
		return
	}

	// Validate updated_by (required for audit trail)
	if req.UpdatedBy == "" {
		log.Warn().Msg("Missing updated_by field")
		http.Error(w, "updated_by is required", http.StatusBadRequest)
		return
	}

	// Update control
	// TODO: Service에 UpdateControl 메서드 추가 필요
	err := h.exitSvc.UpdateControl(ctx, req.Mode, req.Reason, req.UpdatedBy)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update exit control")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("mode", req.Mode).
		Str("updated_by", req.UpdatedBy).
		Msg("Exit control updated")

	w.WriteHeader(http.StatusOK)
}
