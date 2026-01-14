package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/response"
	"github.com/wonny/aegis/v14/internal/domain/stock"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// PriceHandler handles price-related HTTP requests
type PriceHandler struct {
	priceService *pricesync.Service
}

// NewPriceHandler creates a new PriceHandler
func NewPriceHandler(priceService *pricesync.Service) *PriceHandler {
	return &PriceHandler{
		priceService: priceService,
	}
}

// GetBestPrice returns best price for a symbol
// GET /api/prices/:symbol
func (h *PriceHandler) GetBestPrice(c *gin.Context) {
	symbol := c.Param("symbol")

	// Validate symbol format
	if !stock.ValidateSymbol(symbol) {
		response.ErrorWithDetails(c, http.StatusBadRequest,
			response.ErrCodeInvalidParameter,
			"Invalid symbol format",
			"Symbol must be 6-digit number")
		return
	}

	// Get best price
	bp, err := h.priceService.GetBestPrice(c.Request.Context(), symbol)
	if err != nil {
		response.NotFound(c, "Price not found")
		return
	}

	response.Success(c, bp)
}

// BatchGetBestPrices returns best prices for multiple symbols
// POST /api/prices/batch
func (h *PriceHandler) BatchGetBestPrices(c *gin.Context) {
	var req struct {
		Symbols []string `json:"symbols" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// Validate symbols
	if len(req.Symbols) == 0 {
		response.ErrorWithDetails(c, http.StatusBadRequest,
			response.ErrCodeInvalidParameter,
			"Invalid request",
			"symbols array cannot be empty")
		return
	}

	if len(req.Symbols) > 100 {
		response.ErrorWithDetails(c, http.StatusBadRequest,
			response.ErrCodeInvalidParameter,
			"Too many symbols",
			"Maximum 100 symbols per request")
		return
	}

	// Validate each symbol
	for _, symbol := range req.Symbols {
		if !stock.ValidateSymbol(symbol) {
			response.ErrorWithDetails(c, http.StatusBadRequest,
				response.ErrCodeInvalidParameter,
				"Invalid symbol format",
				"Symbol must be 6-digit number: "+symbol)
			return
		}
	}

	// Get best prices
	prices, err := h.priceService.GetBestPrices(c.Request.Context(), req.Symbols)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, prices)
}

// GetFreshness returns freshness data for a symbol
// GET /api/prices/:symbol/freshness
func (h *PriceHandler) GetFreshness(c *gin.Context) {
	symbol := c.Param("symbol")

	// Validate symbol format
	if !stock.ValidateSymbol(symbol) {
		response.ErrorWithDetails(c, http.StatusBadRequest,
			response.ErrCodeInvalidParameter,
			"Invalid symbol format",
			"Symbol must be 6-digit number")
		return
	}

	// Get freshness
	freshnesses, err := h.priceService.GetFreshness(c.Request.Context(), symbol)
	if err != nil {
		response.NotFound(c, "Freshness data not found")
		return
	}

	response.Success(c, freshnesses)
}
