package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// HoldingsHandler handles holdings-related API requests
type HoldingsHandler struct {
	holdingRepo  HoldingReader
	positionRepo exit.PositionRepository
	priceRepo    PriceReader
}

// NewHoldingsHandler creates a new HoldingsHandler
func NewHoldingsHandler(holdingRepo HoldingReader, positionRepo exit.PositionRepository, priceRepo PriceReader) *HoldingsHandler {
	return &HoldingsHandler{
		holdingRepo:  holdingRepo,
		positionRepo: positionRepo,
		priceRepo:    priceRepo,
	}
}

// HoldingWithPrice represents holding with enriched price info
type HoldingWithPrice struct {
	AccountID    string                 `json:"account_id"`
	Symbol       string                 `json:"symbol"`
	Qty          int64                  `json:"qty"`
	AvgPrice     string                 `json:"avg_price"`
	CurrentPrice string                 `json:"current_price"`
	Pnl          string                 `json:"pnl"`
	PnlPct       float64                `json:"pnl_pct"`
	UpdatedTS    string                 `json:"updated_ts"`
	ExitMode     string                 `json:"exit_mode"`
	Raw          map[string]interface{} `json:"raw"`
	// Price info from prices_best
	ChangePrice *int64   `json:"change_price,omitempty"` // 전일대비 (원)
	ChangeRate  *float64 `json:"change_rate,omitempty"`  // 등락률 (%)
	PriceSource string   `json:"price_source,omitempty"` // 가격 출처
}

// GetHoldings retrieves all holdings with price info
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

	// Enrich with price info
	enriched := make([]HoldingWithPrice, 0, len(holdings))
	for _, holding := range holdings {
		item := HoldingWithPrice{
			AccountID:    holding.AccountID,
			Symbol:       holding.Symbol,
			Qty:          holding.Qty,
			AvgPrice:     holding.AvgPrice.String(),
			CurrentPrice: holding.CurrentPrice.String(),
			Pnl:          holding.Pnl.String(),
			PnlPct:       holding.PnlPct,
			UpdatedTS:    holding.UpdatedTS.Format("2006-01-02T15:04:05Z07:00"),
			ExitMode:     holding.ExitMode,
			Raw:          holding.Raw,
		}

		// Get price info
		if priceInfo, err := h.priceRepo.GetBestPrice(ctx, holding.Symbol); err == nil && priceInfo != nil {
			item.ChangePrice = priceInfo.ChangePrice
			item.ChangeRate = priceInfo.ChangeRate
			item.PriceSource = string(priceInfo.BestSource)
		}

		enriched = append(enriched, item)
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(enriched); err != nil {
		log.Error().Err(err).Msg("Failed to encode holdings response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateExitMode updates exit mode for a holding
// PUT /api/holdings/{account_id}/{symbol}/exit-mode
func (h *HoldingsHandler) UpdateExitMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	accountID := vars["account_id"]
	symbol := vars["symbol"]

	log.Info().
		Str("account_id", accountID).
		Str("symbol", symbol).
		Int("symbol_len", len(symbol)).
		Msg("UpdateExitMode - URL params")

	// Parse request body
	var req struct {
		ExitMode string   `json:"exit_mode"`
		Qty      *int64   `json:"qty"`
		AvgPrice *float64 `json:"avg_price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Info().
		Str("exit_mode", req.ExitMode).
		Interface("qty", req.Qty).
		Interface("avg_price", req.AvgPrice).
		Msg("UpdateExitMode - Request body")

	// Validate exit mode
	if req.ExitMode != exit.ExitModeEnabled && req.ExitMode != exit.ExitModeDisabled && req.ExitMode != exit.ExitModeManualOnly {
		http.Error(w, "Invalid exit mode", http.StatusBadRequest)
		return
	}

	// Update exit mode (with optional holding data for upsert)
	if err := h.positionRepo.UpdateExitModeBySymbol(ctx, accountID, symbol, req.ExitMode, req.Qty, req.AvgPrice); err != nil {
		log.Error().Err(err).Str("account_id", accountID).Str("symbol", symbol).Msg("Failed to update exit mode")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("account_id", accountID).Str("symbol", symbol).Str("exit_mode", req.ExitMode).Msg("Exit mode updated")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "exit_mode": req.ExitMode})
}
