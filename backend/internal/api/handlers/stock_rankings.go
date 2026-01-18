package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

// StockRankingsHandler handles stock ranking requests
type StockRankingsHandler struct {
	db          *pgxpool.Pool
	rankingRepo fetcher.RankingRepository
}

// NewStockRankingsHandler creates a new handler
func NewStockRankingsHandler(db *pgxpool.Pool, rankingRepo fetcher.RankingRepository) *StockRankingsHandler {
	return &StockRankingsHandler{
		db:          db,
		rankingRepo: rankingRepo,
	}
}

// RankingStock represents a stock in a ranking
type RankingStock struct {
	Rank            int     `json:"rank"`
	StockCode       string  `json:"stock_code"`
	StockName       string  `json:"stock_name"`
	Market          string  `json:"market"`
	CurrentPrice    float64 `json:"current_price,omitempty"`
	ChangeRate      float64 `json:"change_rate,omitempty"`
	Volume          int64   `json:"volume,omitempty"`
	TradingValue    int64   `json:"trading_value,omitempty"`
	HighPrice       float64 `json:"high_price,omitempty"`
	LowPrice        float64 `json:"low_price,omitempty"`
	MarketCap       int64   `json:"market_cap,omitempty"`
	ForeignNetValue int64   `json:"foreign_net_value,omitempty"`
	InstNetValue    int64   `json:"inst_net_value,omitempty"`
	VolumeSurgeRate float64 `json:"volume_surge_rate,omitempty"`
	High52Week      float64 `json:"high_52week,omitempty"`
}

// RankingResponse represents the response for a ranking request
type RankingResponse struct {
	Category    string         `json:"category"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Stocks      []RankingStock `json:"stocks"`
	TotalCount  int            `json:"total_count"`
}

// GetTopByVolume returns stocks ranked by trading volume
func (h *StockRankingsHandler) GetTopByVolume(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "volume", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get volume rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:         ranking.Rank,
			StockCode:    ranking.StockCode,
			StockName:    ranking.StockName,
			Market:       ranking.Market,
			CurrentPrice: convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:   convertFloat64Ptr(ranking.ChangeRate),
			Volume:       convertInt64Ptr(ranking.Volume),
			TradingValue: convertInt64Ptr(ranking.TradingValue),
			HighPrice:    convertFloat64Ptr(ranking.HighPrice),
			LowPrice:     convertFloat64Ptr(ranking.LowPrice),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "volume",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopByTradingValue returns stocks ranked by trading value
func (h *StockRankingsHandler) GetTopByTradingValue(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "trading_value", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get trading value rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:         ranking.Rank,
			StockCode:    ranking.StockCode,
			StockName:    ranking.StockName,
			Market:       ranking.Market,
			CurrentPrice: convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:   convertFloat64Ptr(ranking.ChangeRate),
			Volume:       convertInt64Ptr(ranking.Volume),
			TradingValue: convertInt64Ptr(ranking.TradingValue),
			HighPrice:    convertFloat64Ptr(ranking.HighPrice),
			LowPrice:     convertFloat64Ptr(ranking.LowPrice),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "trading_value",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopGainers returns stocks with highest price increase
func (h *StockRankingsHandler) GetTopGainers(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "gainers", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get gainers rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:         ranking.Rank,
			StockCode:    ranking.StockCode,
			StockName:    ranking.StockName,
			Market:       ranking.Market,
			CurrentPrice: convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:   convertFloat64Ptr(ranking.ChangeRate),
			Volume:       convertInt64Ptr(ranking.Volume),
			TradingValue: convertInt64Ptr(ranking.TradingValue),
			HighPrice:    convertFloat64Ptr(ranking.HighPrice),
			LowPrice:     convertFloat64Ptr(ranking.LowPrice),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "gainers",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopLosers returns stocks with highest price decrease
func (h *StockRankingsHandler) GetTopLosers(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH stock_prices AS (
			SELECT
				stock_code,
				trade_date,
				close_price,
				ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
			FROM data.daily_prices
			WHERE trade_date >= NOW() - INTERVAL '30 days'
		),
		latest_two AS (
			SELECT
				stock_code,
				MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
				MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
				MAX(CASE WHEN rn = 1 THEN trade_date END) as trade_date
			FROM stock_prices
			WHERE rn <= 2
			GROUP BY stock_code
			HAVING MAX(CASE WHEN rn = 1 THEN close_price END) IS NOT NULL
			   AND MAX(CASE WHEN rn = 2 THEN close_price END) IS NOT NULL
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY ((lt.current_price - lt.prev_close) / lt.prev_close * 100) ASC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lt.current_price,
			((lt.current_price - lt.prev_close) / lt.prev_close * 100) as change_rate,
			lt.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_two lt ON s.code = lt.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND s.name NOT LIKE '%%스팩%%'
		  AND s.name NOT LIKE '%%SPAC%%'
		  AND lt.prev_close > 0
		  AND lt.current_price < lt.prev_close
		  %s
		ORDER BY change_rate ASC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top losers")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stocks := make([]RankingStock, 0)
	var updatedAt time.Time
	for rows.Next() {
		var stock RankingStock
		err := rows.Scan(
			&stock.Rank,
			&stock.StockCode,
			&stock.StockName,
			&stock.Market,
			&stock.CurrentPrice,
			&stock.ChangeRate,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
	}

	response := RankingResponse{
		Category:   "losers",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopForeignNetBuy returns stocks with highest foreign net buying
func (h *StockRankingsHandler) GetTopForeignNetBuy(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "foreign_net_buy", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get foreign net buy rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:            ranking.Rank,
			StockCode:       ranking.StockCode,
			StockName:       ranking.StockName,
			Market:          ranking.Market,
			CurrentPrice:    convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:      convertFloat64Ptr(ranking.ChangeRate),
			Volume:          convertInt64Ptr(ranking.Volume),
			TradingValue:    convertInt64Ptr(ranking.TradingValue),
			HighPrice:       convertFloat64Ptr(ranking.HighPrice),
			LowPrice:        convertFloat64Ptr(ranking.LowPrice),
			ForeignNetValue: convertInt64Ptr(ranking.ForeignNetValue),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "foreign_net_buy",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopInstNetBuy returns stocks with highest institution net buying
func (h *StockRankingsHandler) GetTopInstNetBuy(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "inst_net_buy", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get inst net buy rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:         ranking.Rank,
			StockCode:    ranking.StockCode,
			StockName:    ranking.StockName,
			Market:       ranking.Market,
			CurrentPrice: convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:   convertFloat64Ptr(ranking.ChangeRate),
			Volume:       convertInt64Ptr(ranking.Volume),
			TradingValue: convertInt64Ptr(ranking.TradingValue),
			HighPrice:    convertFloat64Ptr(ranking.HighPrice),
			LowPrice:     convertFloat64Ptr(ranking.LowPrice),
			InstNetValue: convertInt64Ptr(ranking.InstNetValue),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "inst_net_buy",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopByVolumeSurge returns stocks with highest volume surge rate
func (h *StockRankingsHandler) GetTopByVolumeSurge(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "volume_surge", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get volume surge rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:            ranking.Rank,
			StockCode:       ranking.StockCode,
			StockName:       ranking.StockName,
			Market:          ranking.Market,
			CurrentPrice:    convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:      convertFloat64Ptr(ranking.ChangeRate),
			Volume:          convertInt64Ptr(ranking.Volume),
			TradingValue:    convertInt64Ptr(ranking.TradingValue),
			HighPrice:       convertFloat64Ptr(ranking.HighPrice),
			LowPrice:        convertFloat64Ptr(ranking.LowPrice),
			VolumeSurgeRate: convertFloat64Ptr(ranking.VolumeSurgeRate),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "volume_surge",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopBy52WeekHigh returns stocks at 52-week high
func (h *StockRankingsHandler) GetTopBy52WeekHigh(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market")
	if market == "" {
		market = "ALL"
	}

	rankings, err := h.rankingRepo.GetLatest(r.Context(), "high_52week", market, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get 52-week high rankings")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// fetcher.RankingStock을 handlers.RankingStock으로 변환
	stocks := make([]RankingStock, len(rankings))
	var updatedAt time.Time
	for i, ranking := range rankings {
		stocks[i] = RankingStock{
			Rank:         ranking.Rank,
			StockCode:    ranking.StockCode,
			StockName:    ranking.StockName,
			Market:       ranking.Market,
			CurrentPrice: convertFloat64Ptr(ranking.CurrentPrice),
			ChangeRate:   convertFloat64Ptr(ranking.ChangeRate),
			Volume:       convertInt64Ptr(ranking.Volume),
			TradingValue: convertInt64Ptr(ranking.TradingValue),
			HighPrice:    convertFloat64Ptr(ranking.HighPrice),
			LowPrice:     convertFloat64Ptr(ranking.LowPrice),
			High52Week:   convertFloat64Ptr(ranking.High52Week),
		}
		if i == 0 {
			updatedAt = ranking.CollectedAt
		}
	}

	response := RankingResponse{
		Category:   "high_52week",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTopByMarketCap returns stocks ranked by market capitalization
func (h *StockRankingsHandler) GetTopByMarketCap(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 20, 100)
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH latest_market_cap AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				trade_date,
				market_cap
			FROM data.market_cap
			WHERE trade_date >= NOW() - INTERVAL '30 days'
			ORDER BY stock_code, trade_date DESC
		),
		latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				close_price as current_price
			FROM data.daily_prices
			WHERE trade_date >= NOW() - INTERVAL '30 days'
			ORDER BY stock_code, trade_date DESC
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY lmc.market_cap DESC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lp.current_price,
			lmc.market_cap,
			lmc.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_market_cap lmc ON s.code = lmc.stock_code
		LEFT JOIN latest_prices lp ON s.code = lp.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND s.name NOT LIKE '%%스팩%%'
		  AND s.name NOT LIKE '%%SPAC%%'
		  AND lmc.market_cap > 0
		  %s
		ORDER BY lmc.market_cap DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top by market cap")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stocks := make([]RankingStock, 0)
	var updatedAt time.Time
	for rows.Next() {
		var stock RankingStock
		err := rows.Scan(
			&stock.Rank,
			&stock.StockCode,
			&stock.StockName,
			&stock.Market,
			&stock.CurrentPrice,
			&stock.MarketCap,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
	}

	response := RankingResponse{
		Category:   "market_cap",
		UpdatedAt:  updatedAt,
		Stocks:     stocks,
		TotalCount: len(stocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseLimit parses the limit query parameter
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultLimit
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return defaultLimit
	}

	if limit > maxLimit {
		return maxLimit
	}

	return limit
}

// convertFloat64Ptr converts a *float64 to float64, returning 0 if nil
func convertFloat64Ptr(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// convertInt64Ptr converts a *int64 to int64, returning 0 if nil
func convertInt64Ptr(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
