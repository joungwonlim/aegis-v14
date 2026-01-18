package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/infra/external/naver"
	"github.com/wonny/aegis/v14/internal/infrastructure/postgres/universe"
)

// StockInfoHandler handles stock info API requests
type StockInfoHandler struct {
	repo        *universe.StockInfoRepository
	naverClient *naver.Client
}

// NewStockInfoHandler creates a new StockInfoHandler
func NewStockInfoHandler(repo *universe.StockInfoRepository, naverClient *naver.Client) *StockInfoHandler {
	return &StockInfoHandler{
		repo:        repo,
		naverClient: naverClient,
	}
}

// StockInfoResponse API 응답 구조
type StockInfoResponse struct {
	Symbol          string  `json:"symbol"`
	SymbolName      *string `json:"symbol_name,omitempty"`
	CompanyOverview *string `json:"company_overview,omitempty"`
	OverviewSource  *string `json:"overview_source,omitempty"`
}

// GetCompanyOverview handles GET /api/stocks/:symbol/info
// DB에 있으면 반환, 없으면 네이버에서 가져와서 저장 후 반환
func (h *StockInfoHandler) GetCompanyOverview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	ctx := r.Context()

	// 1. DB에서 먼저 조회
	info, err := h.repo.GetBySymbol(ctx, symbol)
	if err != nil {
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to get stock info from DB")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 2. DB에 있고 company_overview가 있으면 반환
	if info != nil && info.CompanyOverview != nil && *info.CompanyOverview != "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StockInfoResponse{
			Symbol:          info.Symbol,
			SymbolName:      info.SymbolName,
			CompanyOverview: info.CompanyOverview,
			OverviewSource:  info.OverviewSource,
		})
		return
	}

	// 3. DB에 없으면 네이버에서 가져오기
	overview, err := h.naverClient.FetchCompanyOverview(ctx, symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", symbol).Msg("Failed to fetch company overview from Naver")
		// 에러가 발생해도 빈 응답 반환
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StockInfoResponse{
			Symbol: symbol,
		})
		return
	}

	// 4. DB에 저장
	if err := h.repo.UpsertCompanyOverview(ctx, symbol, overview.SymbolName, overview.Overview, overview.FetchedFrom); err != nil {
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to save company overview to DB")
		// 저장 실패해도 응답은 반환
	}

	// 5. 응답 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StockInfoResponse{
		Symbol:          symbol,
		SymbolName:      &overview.SymbolName,
		CompanyOverview: &overview.Overview,
		OverviewSource:  &overview.FetchedFrom,
	})
}

// RefreshCompanyOverview handles POST /api/stocks/:symbol/info/refresh
// 강제로 네이버에서 새로 가져와서 DB 업데이트
func (h *StockInfoHandler) RefreshCompanyOverview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	ctx := r.Context()

	// 네이버에서 가져오기
	overview, err := h.naverClient.FetchCompanyOverview(ctx, symbol)
	if err != nil {
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch company overview from Naver")
		http.Error(w, "Failed to fetch from Naver: "+err.Error(), http.StatusBadGateway)
		return
	}

	// DB에 저장
	if err := h.repo.UpsertCompanyOverview(ctx, symbol, overview.SymbolName, overview.Overview, overview.FetchedFrom); err != nil {
		log.Error().Err(err).Str("symbol", symbol).Msg("Failed to save company overview to DB")
		http.Error(w, "Failed to save to DB", http.StatusInternalServerError)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StockInfoResponse{
		Symbol:          symbol,
		SymbolName:      &overview.SymbolName,
		CompanyOverview: &overview.Overview,
		OverviewSource:  &overview.FetchedFrom,
	})
}
