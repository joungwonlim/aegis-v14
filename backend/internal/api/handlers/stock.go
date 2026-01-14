package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/response"
	"github.com/wonny/aegis/v14/internal/domain/stock"
)

// StockHandler handles stock-related HTTP requests
type StockHandler struct {
	stockRepo stock.Repository
}

// NewStockHandler creates a new StockHandler
func NewStockHandler(stockRepo stock.Repository) *StockHandler {
	return &StockHandler{
		stockRepo: stockRepo,
	}
}

// List handles GET /api/stocks
func (h *StockHandler) List(c *gin.Context) {
	// Parse query parameters
	filter := stock.ListFilter{}

	// market (optional)
	if market := c.Query("market"); market != "" {
		filter.Market = &market
	}

	// status (default: ACTIVE)
	filter.Status = c.DefaultQuery("status", "ACTIVE")

	// is_tradable (optional)
	if tradableStr := c.Query("is_tradable"); tradableStr != "" {
		if tradableStr == "true" {
			tradable := true
			filter.IsTradable = &tradable
		} else if tradableStr == "false" {
			tradable := false
			filter.IsTradable = &tradable
		}
	}

	// search (optional)
	filter.Search = c.Query("search")

	// sort (default: symbol)
	filter.Sort = c.DefaultQuery("sort", "symbol")

	// order (default: asc)
	filter.Order = c.DefaultQuery("order", "asc")

	// page (default: 1)
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	filter.Page = page

	// limit (default: 20, max: 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	filter.Limit = limit

	// Normalize and validate filter
	if err := filter.Normalize(); err != nil {
		var message, details string
		switch err {
		case stock.ErrInvalidMarket:
			message = "Invalid market value"
			details = "market must be one of: KOSPI, KOSDAQ, KONEX"
		case stock.ErrInvalidStatus:
			message = "Invalid status value"
			details = "status must be one of: ACTIVE, SUSPENDED, DELISTED, ALL"
		case stock.ErrInvalidSort:
			message = "Invalid sort field"
			details = "sort must be one of: symbol, name, market_cap"
		case stock.ErrInvalidOrder:
			message = "Invalid order direction"
			details = "order must be one of: asc, desc"
		default:
			response.InternalError(c, err)
			return
		}
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter, message, details)
		return
	}

	// Query stocks
	result, err := h.stockRepo.List(c.Request.Context(), filter)
	if err != nil {
		response.DatabaseError(c, err)
		return
	}

	// Calculate pagination
	pagination := response.NewPagination(result.Page, result.Limit, result.TotalCount)

	// Return response
	response.SuccessWithPagination(c, result.Stocks, pagination)
}

// GetBySymbol handles GET /api/stocks/:symbol
func (h *StockHandler) GetBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")

	// Validate symbol format
	if !stock.ValidateSymbol(symbol) {
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter,
			"Invalid symbol format", "Symbol must be 6-digit number")
		return
	}

	// Query stock
	s, err := h.stockRepo.GetBySymbol(c.Request.Context(), symbol)
	if err != nil {
		if err == stock.ErrStockNotFound {
			response.ErrorWithDetails(c, http.StatusNotFound, response.ErrCodeNotFound,
				"Stock not found", fmt.Sprintf("No stock found with symbol: %s", symbol))
			return
		}
		response.DatabaseError(c, err)
		return
	}

	// Return response
	response.Success(c, s)
}
