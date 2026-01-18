package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	fetcherService "github.com/wonny/aegis/v14/internal/service/fetcher"
)

// FetcherService Fetcher 서비스 인터페이스
type FetcherService interface {
	// Stock
	GetStock(ctx context.Context, code string) (*fetcher.Stock, error)
	ListStocks(ctx context.Context, filter *fetcher.StockFilter) ([]*fetcher.Stock, error)
	RefreshStockMaster(ctx context.Context) error

	// Price
	GetLatestPrice(ctx context.Context, stockCode string) (*fetcher.DailyPrice, error)
	GetPriceRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.DailyPrice, error)

	// Flow
	GetLatestFlow(ctx context.Context, stockCode string) (*fetcher.InvestorFlow, error)
	GetFlowRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.InvestorFlow, error)

	// Fundamentals
	GetLatestFundamentals(ctx context.Context, stockCode string) (*fetcher.Fundamentals, error)

	// MarketCap
	GetLatestMarketCap(ctx context.Context, stockCode string) (*fetcher.MarketCap, error)

	// Disclosure
	GetRecentDisclosures(ctx context.Context, limit int) ([]*fetcher.Disclosure, error)
	GetStockDisclosures(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.Disclosure, error)

	// Collection
	CollectNow(ctx context.Context, collectorType fetcherService.CollectorType) error
	CollectStock(ctx context.Context, stockCode string) (*fetcher.FetchResult, error)

	// Schedule
	GetSchedules() []fetcherService.ScheduleInfo
}

// Handler Fetcher API 핸들러
type Handler struct {
	service FetcherService
}

// NewHandler 핸들러 생성
func NewHandler(service FetcherService) *Handler {
	return &Handler{service: service}
}

// =============================================================================
// Response Types
// =============================================================================

// ListStocksResponse 종목 목록 응답
type ListStocksResponse struct {
	Stocks []*fetcher.Stock `json:"stocks"`
	Count  int              `json:"count"`
}

// PriceHistoryResponse 가격 이력 응답
type PriceHistoryResponse struct {
	StockCode string               `json:"stock_code"`
	Prices    []*fetcher.DailyPrice `json:"prices"`
	Count     int                   `json:"count"`
}

// FlowHistoryResponse 수급 이력 응답
type FlowHistoryResponse struct {
	StockCode string                 `json:"stock_code"`
	Flows     []*fetcher.InvestorFlow `json:"flows"`
	Count     int                     `json:"count"`
}

// DisclosureListResponse 공시 목록 응답
type DisclosureListResponse struct {
	Disclosures []*fetcher.Disclosure `json:"disclosures"`
	Count       int                   `json:"count"`
}

// StockDataResponse 종목 데이터 종합 응답
type StockDataResponse struct {
	Stock        *fetcher.Stock       `json:"stock"`
	LatestPrice  *fetcher.DailyPrice  `json:"latest_price,omitempty"`
	LatestFlow   *fetcher.InvestorFlow `json:"latest_flow,omitempty"`
	Fundamentals *fetcher.Fundamentals `json:"fundamentals,omitempty"`
	MarketCap    *fetcher.MarketCap   `json:"market_cap,omitempty"`
}

// CollectRequest 수집 요청
type CollectRequest struct {
	CollectorType string `json:"collector_type"` // price, flow, fundamental, marketcap, disclosure
}

// CollectResponse 수집 응답
type CollectResponse struct {
	Success       bool   `json:"success"`
	CollectorType string `json:"collector_type"`
	Message       string `json:"message"`
}

// =============================================================================
// Stock Handlers
// =============================================================================

// ListStocks handles GET /api/v1/fetcher/stocks
func (h *Handler) ListStocks(w http.ResponseWriter, r *http.Request) {
	filter := &fetcher.StockFilter{}

	// Parse query parameters
	if market := r.URL.Query().Get("market"); market != "" {
		filter.Market = market
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	} else {
		filter.Status = "active"
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	stocks, err := h.service.ListStocks(r.Context(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list stocks")
		http.Error(w, "Failed to list stocks", http.StatusInternalServerError)
		return
	}

	response := ListStocksResponse{
		Stocks: stocks,
		Count:  len(stocks),
	}

	h.writeJSON(w, response)
}

// GetStock handles GET /api/v1/fetcher/stocks/{code}
func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	stock, err := h.service.GetStock(r.Context(), code)
	if err != nil {
		if err == fetcher.ErrStockNotFound {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to get stock")
		http.Error(w, "Failed to get stock", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, stock)
}

// GetStockData handles GET /api/v1/fetcher/stocks/{code}/data
func (h *Handler) GetStockData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	ctx := r.Context()

	stock, err := h.service.GetStock(ctx, code)
	if err != nil {
		if err == fetcher.ErrStockNotFound {
			http.Error(w, "Stock not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to get stock")
		http.Error(w, "Failed to get stock", http.StatusInternalServerError)
		return
	}

	response := StockDataResponse{Stock: stock}

	// Optional data - don't fail on errors
	if price, err := h.service.GetLatestPrice(ctx, code); err == nil {
		response.LatestPrice = price
	}
	if flow, err := h.service.GetLatestFlow(ctx, code); err == nil {
		response.LatestFlow = flow
	}
	if fund, err := h.service.GetLatestFundamentals(ctx, code); err == nil {
		response.Fundamentals = fund
	}
	if mc, err := h.service.GetLatestMarketCap(ctx, code); err == nil {
		response.MarketCap = mc
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Price Handlers
// =============================================================================

// GetLatestPrice handles GET /api/v1/fetcher/prices/{code}
func (h *Handler) GetLatestPrice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	price, err := h.service.GetLatestPrice(r.Context(), code)
	if err != nil {
		if err == fetcher.ErrPriceNotFound {
			http.Error(w, "Price not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to get price")
		http.Error(w, "Failed to get price", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, price)
}

// GetPriceHistory handles GET /api/v1/fetcher/prices/{code}/history
func (h *Handler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	from, to := h.parseDateRange(r, 30) // Default: last 30 days

	prices, err := h.service.GetPriceRange(r.Context(), code, from, to)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to get price history")
		http.Error(w, "Failed to get price history", http.StatusInternalServerError)
		return
	}

	response := PriceHistoryResponse{
		StockCode: code,
		Prices:    prices,
		Count:     len(prices),
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Flow Handlers
// =============================================================================

// GetLatestFlow handles GET /api/v1/fetcher/flows/{code}
func (h *Handler) GetLatestFlow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	flow, err := h.service.GetLatestFlow(r.Context(), code)
	if err != nil {
		if err == fetcher.ErrFlowNotFound {
			http.Error(w, "Flow not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to get flow")
		http.Error(w, "Failed to get flow", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, flow)
}

// GetFlowHistory handles GET /api/v1/fetcher/flows/{code}/history
func (h *Handler) GetFlowHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	from, to := h.parseDateRange(r, 30)

	flows, err := h.service.GetFlowRange(r.Context(), code, from, to)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to get flow history")
		http.Error(w, "Failed to get flow history", http.StatusInternalServerError)
		return
	}

	response := FlowHistoryResponse{
		StockCode: code,
		Flows:     flows,
		Count:     len(flows),
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Disclosure Handlers
// =============================================================================

// GetRecentDisclosures handles GET /api/v1/fetcher/disclosures
func (h *Handler) GetRecentDisclosures(w http.ResponseWriter, r *http.Request) {
	limit := 50 // Default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	disclosures, err := h.service.GetRecentDisclosures(r.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get recent disclosures")
		http.Error(w, "Failed to get disclosures", http.StatusInternalServerError)
		return
	}

	response := DisclosureListResponse{
		Disclosures: disclosures,
		Count:       len(disclosures),
	}

	h.writeJSON(w, response)
}

// GetStockDisclosures handles GET /api/v1/fetcher/disclosures/{code}
func (h *Handler) GetStockDisclosures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	from, to := h.parseDateRange(r, 90) // Default: last 90 days

	disclosures, err := h.service.GetStockDisclosures(r.Context(), code, from, to)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to get stock disclosures")
		http.Error(w, "Failed to get disclosures", http.StatusInternalServerError)
		return
	}

	response := DisclosureListResponse{
		Disclosures: disclosures,
		Count:       len(disclosures),
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Collection Handlers
// =============================================================================

// TriggerCollection handles POST /api/v1/fetcher/collect
func (h *Handler) TriggerCollection(w http.ResponseWriter, r *http.Request) {
	var req CollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	typeMap := map[string]fetcherService.CollectorType{
		"price":       fetcherService.CollectorPrice,
		"flow":        fetcherService.CollectorFlow,
		"fundamental": fetcherService.CollectorFundament,
		"marketcap":   fetcherService.CollectorMarketCap,
		"disclosure":  fetcherService.CollectorDisclosure,
	}

	collectorType, ok := typeMap[req.CollectorType]
	if !ok {
		http.Error(w, "Invalid collector type", http.StatusBadRequest)
		return
	}

	if err := h.service.CollectNow(r.Context(), collectorType); err != nil {
		log.Error().Err(err).Str("type", req.CollectorType).Msg("Failed to trigger collection")
		http.Error(w, "Failed to trigger collection", http.StatusInternalServerError)
		return
	}

	response := CollectResponse{
		Success:       true,
		CollectorType: req.CollectorType,
		Message:       "Collection triggered successfully",
	}

	h.writeJSON(w, response)
}

// CollectStock handles POST /api/v1/fetcher/collect/{code}
func (h *Handler) CollectStock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	result, err := h.service.CollectStock(r.Context(), code)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to collect stock data")
		http.Error(w, "Failed to collect stock data", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, result)
}

// RefreshStockMaster handles POST /api/v1/fetcher/refresh-stocks
func (h *Handler) RefreshStockMaster(w http.ResponseWriter, r *http.Request) {
	if err := h.service.RefreshStockMaster(r.Context()); err != nil {
		log.Error().Err(err).Msg("Failed to refresh stock master")
		http.Error(w, "Failed to refresh stock master", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Stock master refreshed successfully",
	}

	h.writeJSON(w, response)
}

// GetSchedules handles GET /api/v1/fetcher/schedules
func (h *Handler) GetSchedules(w http.ResponseWriter, r *http.Request) {
	schedules := h.service.GetSchedules()

	response := map[string]interface{}{
		"schedules": schedules,
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Helpers
// =============================================================================

func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) parseDateRange(r *http.Request, defaultDays int) (from, to time.Time) {
	to = time.Now()
	from = to.AddDate(0, 0, -defaultDays)

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t
		}
	}

	return from, to
}
