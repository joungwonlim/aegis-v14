package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// StockRankingsHandler handles stock ranking requests
type StockRankingsHandler struct {
	db *pgxpool.Pool
}

// NewStockRankingsHandler creates a new handler
func NewStockRankingsHandler(db *pgxpool.Pool) *StockRankingsHandler {
	return &StockRankingsHandler{db: db}
}

// RankingStock represents a stock in a ranking
type RankingStock struct {
	Rank        int     `json:"rank"`
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Market      string  `json:"market"`
	CurrentPrice float64 `json:"current_price,omitempty"`
	ChangeRate  float64 `json:"change_rate,omitempty"`
	Volume      int64   `json:"volume,omitempty"`
	TradingValue int64  `json:"trading_value,omitempty"`
	MarketCap   int64   `json:"market_cap,omitempty"`
	ForeignNetValue int64 `json:"foreign_net_value,omitempty"`
	InstNetValue int64 `json:"inst_net_value,omitempty"`
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
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				trade_date,
				close_price as current_price,
				volume,
				trading_value
			FROM data.daily_prices
			WHERE trade_date >= NOW() - INTERVAL '30 days'
			ORDER BY stock_code, trade_date DESC
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY lp.volume DESC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lp.current_price,
			lp.volume,
			lp.trading_value,
			lp.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_prices lp ON s.code = lp.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND lp.volume > 0
		  %s
		ORDER BY lp.volume DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top by volume")
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
			&stock.Volume,
			&stock.TradingValue,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
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
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH latest_prices AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				trade_date,
				close_price as current_price,
				volume,
				trading_value
			FROM data.daily_prices
			WHERE trade_date >= NOW() - INTERVAL '30 days'
			ORDER BY stock_code, trade_date DESC
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY lp.trading_value DESC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lp.current_price,
			lp.volume,
			lp.trading_value,
			lp.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_prices lp ON s.code = lp.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND lp.trading_value > 0
		  %s
		ORDER BY lp.trading_value DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top by trading value")
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
			&stock.Volume,
			&stock.TradingValue,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
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
			ROW_NUMBER() OVER (ORDER BY ((lt.current_price - lt.prev_close) / lt.prev_close * 100) DESC) as rank,
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
		  AND lt.prev_close > 0
		  AND lt.current_price > lt.prev_close
		  %s
		ORDER BY change_rate DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top gainers")
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
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH latest_flow AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				trade_date,
				foreign_net_value
			FROM data.investor_flow
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
			ROW_NUMBER() OVER (ORDER BY ABS(lf.foreign_net_value) DESC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lp.current_price,
			lf.foreign_net_value,
			lf.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_flow lf ON s.code = lf.stock_code
		LEFT JOIN latest_prices lp ON s.code = lp.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND lf.foreign_net_value != 0
		  %s
		ORDER BY ABS(lf.foreign_net_value) DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top foreign net buy")
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
			&stock.ForeignNetValue,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
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
	market := r.URL.Query().Get("market") // ALL, KOSPI, KOSDAQ

	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	query := fmt.Sprintf(`
		WITH latest_flow AS (
			SELECT DISTINCT ON (stock_code)
				stock_code,
				trade_date,
				inst_net_value
			FROM data.investor_flow
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
			ROW_NUMBER() OVER (ORDER BY ABS(lf.inst_net_value) DESC) as rank,
			s.code as stock_code,
			s.name as stock_name,
			s.market,
			lp.current_price,
			lf.inst_net_value,
			lf.trade_date as updated_at
		FROM data.stocks s
		JOIN latest_flow lf ON s.code = lf.stock_code
		LEFT JOIN latest_prices lp ON s.code = lp.stock_code
		WHERE s.status = 'active'
		  AND s.market NOT IN ('ETF', 'ETN')
		  AND lf.inst_net_value != 0
		  %s
		ORDER BY ABS(lf.inst_net_value) DESC
		LIMIT $1
	`, marketFilter)

	rows, err := h.db.Query(r.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query top inst net buy")
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
			&stock.InstNetValue,
			&updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		stocks = append(stocks, stock)
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
