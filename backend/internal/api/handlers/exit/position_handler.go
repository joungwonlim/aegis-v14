package exit

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	exitService "github.com/wonny/aegis/v14/internal/service/exit"
)

// PositionHandler handles position exit endpoints
type PositionHandler struct {
	exitSvc *exitService.Service
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(exitSvc *exitService.Service) *PositionHandler {
	return &PositionHandler{
		exitSvc: exitSvc,
	}
}

// ManualExitRequest represents POST /api/v1/exit/positions/{positionId}/manual request
type ManualExitRequest struct {
	Qty       int64  `json:"qty"`
	OrderType string `json:"order_type"`
}

// PositionStateResponse represents GET /api/v1/exit/positions/{positionId}/state response
type PositionStateResponse struct {
	PositionID     string  `json:"position_id"`
	Phase          string  `json:"phase"`
	HWMPrice       *string `json:"hwm_price"`        // High-Water Mark (decimal string)
	StopFloorPrice *string `json:"stop_floor_price"` // Stop Floor (decimal string)
	ATR            *string `json:"atr"`              // ATR (decimal string)
	CooldownUntil  *string `json:"cooldown_until"`   // ISO8601 timestamp
	LastEvalTS     *string `json:"last_eval_ts"`     // ISO8601 timestamp
	UpdatedTS      string  `json:"updated_ts"`       // ISO8601 timestamp
}

// CreateManualExit handles POST /api/v1/exit/positions/{positionId}/manual
func (h *PositionHandler) CreateManualExit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse positionId from URL
	vars := mux.Vars(r)
	positionIDStr := vars["positionId"]
	positionID, err := uuid.Parse(positionIDStr)
	if err != nil {
		log.Warn().Str("position_id", positionIDStr).Msg("Invalid position ID")
		http.Error(w, "Invalid position ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req ManualExitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate qty
	if req.Qty <= 0 {
		log.Warn().Int64("qty", req.Qty).Msg("Invalid qty (must be > 0)")
		http.Error(w, "qty must be greater than 0", http.StatusBadRequest)
		return
	}

	// Validate order type
	validOrderTypes := map[string]bool{
		"MKT": true,
		"LMT": true,
	}
	if !validOrderTypes[req.OrderType] {
		log.Warn().Str("order_type", req.OrderType).Msg("Invalid order type")
		http.Error(w, "Invalid order type (must be MKT or LMT)", http.StatusBadRequest)
		return
	}

	// Create manual intent
	err = h.exitSvc.CreateManualIntent(ctx, positionID, req.Qty, req.OrderType)
	if err != nil {
		// Handle specific errors
		switch err.Error() {
		case "exit disabled":
			log.Warn().Str("position_id", positionIDStr).Msg("Exit disabled for position")
			http.Error(w, "Exit is disabled for this position", http.StatusForbidden)
			return
		case "no available quantity (already locked)":
			log.Warn().Str("position_id", positionIDStr).Msg("No available qty")
			http.Error(w, "No available quantity (all qty locked in pending orders)", http.StatusConflict)
			return
		default:
			log.Error().Err(err).Str("position_id", positionIDStr).Msg("Failed to create manual intent")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	log.Info().
		Str("position_id", positionIDStr).
		Int64("qty", req.Qty).
		Str("order_type", req.OrderType).
		Msg("Manual exit intent created")

	w.WriteHeader(http.StatusCreated)
}

// GetPositionState handles GET /api/v1/exit/positions/{positionId}/state
func (h *PositionHandler) GetPositionState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse positionId from URL
	vars := mux.Vars(r)
	positionIDStr := vars["positionId"]
	positionID, err := uuid.Parse(positionIDStr)
	if err != nil {
		log.Warn().Str("position_id", positionIDStr).Msg("Invalid position ID")
		http.Error(w, "Invalid position ID", http.StatusBadRequest)
		return
	}

	// Get position state
	state, err := h.exitSvc.GetPositionState(ctx, positionID)
	if err != nil {
		log.Error().Err(err).Str("position_id", positionIDStr).Msg("Failed to get position state")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert optional decimal fields to strings
	var hwmPriceStr, stopFloorPriceStr, atrStr *string
	if state.HWMPrice != nil {
		s := state.HWMPrice.String()
		hwmPriceStr = &s
	}
	if state.StopFloorPrice != nil {
		s := state.StopFloorPrice.String()
		stopFloorPriceStr = &s
	}
	if state.ATR != nil {
		s := state.ATR.String()
		atrStr = &s
	}

	// Convert optional time fields to ISO8601 strings
	var cooldownUntilStr, lastEvalTSStr *string
	if state.CooldownUntil != nil {
		s := state.CooldownUntil.Format("2006-01-02T15:04:05-07:00")
		cooldownUntilStr = &s
	}
	if state.LastEvalTS != nil {
		s := state.LastEvalTS.Format("2006-01-02T15:04:05-07:00")
		lastEvalTSStr = &s
	}

	resp := PositionStateResponse{
		PositionID:     state.PositionID.String(),
		Phase:          state.Phase,
		HWMPrice:       hwmPriceStr,
		StopFloorPrice: stopFloorPriceStr,
		ATR:            atrStr,
		CooldownUntil:  cooldownUntilStr,
		LastEvalTS:     lastEvalTSStr,
		UpdatedTS:      state.UpdatedTS.Format("2006-01-02T15:04:05-07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
