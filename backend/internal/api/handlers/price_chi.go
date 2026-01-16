package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// PriceHandlerChi handles price-related HTTP requests using chi router
type PriceHandlerChi struct {
	priceRepo *postgres.PriceRepository
}

// NewPriceHandlerChi creates a new PriceHandlerChi
func NewPriceHandlerChi(priceRepo *postgres.PriceRepository) *PriceHandlerChi {
	return &PriceHandlerChi{
		priceRepo: priceRepo,
	}
}

// GetPrice returns the latest price for a symbol
// GET /api/prices/{symbol}
func (h *PriceHandlerChi) GetPrice(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "Symbol is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get best price from DB
	price, err := h.priceRepo.GetBestPrice(ctx, symbol)
	if err != nil {
		log.Debug().Err(err).Str("symbol", symbol).Msg("Price not found")
		http.Error(w, "Price not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(price)
}

// GetPrices returns prices for multiple symbols
// POST /api/prices/batch
func (h *PriceHandlerChi) GetPrices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbols []string `json:"symbols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Symbols) == 0 {
		http.Error(w, "Symbols array is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get prices for each symbol
	type PriceResult struct {
		Symbol      string  `json:"symbol"`
		LastPrice   int64   `json:"last_price"`
		ChangePrice int64   `json:"change_price"`
		ChangeRate  float64 `json:"change_rate"`
		Source      string  `json:"source"`
		Found       bool    `json:"found"`
	}

	results := make([]PriceResult, 0, len(req.Symbols))
	for _, symbol := range req.Symbols {
		price, err := h.priceRepo.GetBestPrice(ctx, symbol)
		if err != nil {
			results = append(results, PriceResult{
				Symbol: symbol,
				Found:  false,
			})
			continue
		}

		changePrice := int64(0)
		if price.ChangePrice != nil {
			changePrice = *price.ChangePrice
		}
		changeRate := float64(0)
		if price.ChangeRate != nil {
			changeRate = *price.ChangeRate
		}

		results = append(results, PriceResult{
			Symbol:      symbol,
			LastPrice:   price.BestPrice,
			ChangePrice: changePrice,
			ChangeRate:  changeRate,
			Source:      string(price.BestSource),
			Found:       true,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
