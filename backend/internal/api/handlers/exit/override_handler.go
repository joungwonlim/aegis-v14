package exit

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	exitService "github.com/wonny/aegis/v14/internal/service/exit"
)

// OverrideHandler handles symbol exit override endpoints
type OverrideHandler struct {
	exitSvc *exitService.Service
}

// NewOverrideHandler creates a new override handler
func NewOverrideHandler(exitSvc *exitService.Service) *OverrideHandler {
	return &OverrideHandler{
		exitSvc: exitSvc,
	}
}

// OverrideResponse represents symbol override API response
type OverrideResponse struct {
	Symbol        string  `json:"symbol"`
	ProfileID     string  `json:"profile_id"`
	Enabled       bool    `json:"enabled"`
	EffectiveFrom *string `json:"effective_from"`
	Reason        string  `json:"reason"`
}

// SetOverrideRequest represents POST /api/v1/exit/overrides/{symbol} request
type SetOverrideRequest struct {
	ProfileID string `json:"profile_id"`
	Reason    string `json:"reason"`
	CreatedBy string `json:"created_by"`
}

// GetOverride handles GET /api/v1/exit/overrides/{symbol}
func (h *OverrideHandler) GetOverride(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse symbol from URL
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	// Get override
	override, err := h.exitSvc.GetSymbolOverride(ctx, symbol)
	if err != nil {
		if err.Error() == "override not found for symbol: "+symbol {
			http.Error(w, "Override not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to get override")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert effective_from to string
	var effectiveFromStr *string
	if override.EffectiveFrom != nil {
		s := override.EffectiveFrom.Format("2006-01-02")
		effectiveFromStr = &s
	}

	resp := OverrideResponse{
		Symbol:        override.Symbol,
		ProfileID:     override.ProfileID,
		Enabled:       override.Enabled,
		EffectiveFrom: effectiveFromStr,
		Reason:        override.Reason,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// SetOverride handles POST /api/v1/exit/overrides/{symbol}
func (h *OverrideHandler) SetOverride(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse symbol from URL
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req SetOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ProfileID == "" {
		http.Error(w, "profile_id is required", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}

	// Create override (effective immediately)
	now := time.Now()
	override := &exit.SymbolExitOverride{
		Symbol:        symbol,
		ProfileID:     req.ProfileID,
		Enabled:       true,
		EffectiveFrom: &now,
		Reason:        req.Reason,
		CreatedBy:     req.CreatedBy,
	}

	err := h.exitSvc.SetSymbolOverride(ctx, override)
	if err != nil {
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to set override")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("symbol", symbol).
		Str("profile_id", req.ProfileID).
		Str("created_by", req.CreatedBy).
		Msg("Symbol override set")

	w.WriteHeader(http.StatusCreated)
}

// DeleteOverride handles DELETE /api/v1/exit/overrides/{symbol}
func (h *OverrideHandler) DeleteOverride(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse symbol from URL
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	// Delete override
	err := h.exitSvc.DeleteSymbolOverride(ctx, symbol)
	if err != nil {
		if err.Error() == "override not found for symbol: "+symbol {
			http.Error(w, "Override not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to delete override")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("symbol", symbol).
		Msg("Symbol override deleted")

	w.WriteHeader(http.StatusNoContent)
}
