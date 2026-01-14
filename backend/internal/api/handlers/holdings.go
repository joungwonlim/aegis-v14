package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// HoldingsHandler handles holdings-related API requests
type HoldingsHandler struct {
	holdingRepo  HoldingReader
	positionRepo exit.PositionRepository
}

// NewHoldingsHandler creates a new HoldingsHandler
func NewHoldingsHandler(holdingRepo HoldingReader, positionRepo exit.PositionRepository) *HoldingsHandler {
	return &HoldingsHandler{
		holdingRepo:  holdingRepo,
		positionRepo: positionRepo,
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

// UpdateExitMode updates exit mode for a holding
// PUT /api/holdings/{account_id}/{symbol}/exit-mode
func (h *HoldingsHandler) UpdateExitMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := chi.URLParam(r, "account_id")
	symbol := chi.URLParam(r, "symbol")

	// Parse request body
	var req struct {
		ExitMode string `json:"exit_mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate exit mode
	if req.ExitMode != exit.ExitModeEnabled && req.ExitMode != exit.ExitModeDisabled && req.ExitMode != exit.ExitModeManualOnly {
		http.Error(w, "Invalid exit mode", http.StatusBadRequest)
		return
	}

	// Update exit mode
	if err := h.positionRepo.UpdateExitModeBySymbol(ctx, accountID, symbol, req.ExitMode); err != nil {
		log.Error().Err(err).Str("account_id", accountID).Str("symbol", symbol).Msg("Failed to update exit mode")
		http.Error(w, "Failed to update exit mode", http.StatusInternalServerError)
		return
	}

	log.Info().Str("account_id", accountID).Str("symbol", symbol).Str("exit_mode", req.ExitMode).Msg("Exit mode updated")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "exit_mode": req.ExitMode})
}
