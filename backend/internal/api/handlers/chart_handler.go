package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChartHandler handles chart data requests
type ChartHandler struct {
	pool *pgxpool.Pool
}

// NewChartHandler creates a new ChartHandler
func NewChartHandler(pool *pgxpool.Pool) *ChartHandler {
	return &ChartHandler{
		pool: pool,
	}
}

// DailyPriceResponse represents daily price data for charts
type DailyPriceResponse struct {
	Date   string  `json:"date"`
	Open   int64   `json:"open"`
	High   int64   `json:"high"`
	Low    int64   `json:"low"`
	Close  int64   `json:"close"`
	Volume int64   `json:"volume"`
}

// InvestorFlowResponse represents investor flow data for charts
type InvestorFlowResponse struct {
	Date        string  `json:"date"`
	ForeignNet  int64   `json:"foreign_net"`
	InstNet     int64   `json:"inst_net"`
	RetailNet   int64   `json:"retail_net"`
	ClosePrice  int64   `json:"close_price"`
	PriceChange int64   `json:"price_change"`
	ChangeRate  float64 `json:"change_rate"`
	Volume      int64   `json:"volume"`
}

// GetPriceHistory handles GET /api/v1/fetcher/prices/{code}/history
func (h *ChartHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["code"]

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 3 months if not specified
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	if startDate == "" {
		start := time.Now().AddDate(0, -3, 0)
		startDate = start.Format("2006-01-02")
	}

	// Query database
	query := `
		SELECT
			trade_date,
			open_price,
			high_price,
			low_price,
			close_price,
			volume
		FROM data.daily_prices
		WHERE stock_code = $1
			AND trade_date >= $2
			AND trade_date <= $3
		ORDER BY trade_date ASC
	`

	rows, err := h.pool.Query(r.Context(), query, symbol, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var prices []DailyPriceResponse
	for rows.Next() {
		var p DailyPriceResponse
		var tradeDate time.Time
		var open, high, low, close float64
		err := rows.Scan(&tradeDate, &open, &high, &low, &close, &p.Volume)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p.Date = tradeDate.Format("2006-01-02")
		p.Open = int64(open)
		p.High = int64(high)
		p.Low = int64(low)
		p.Close = int64(close)
		prices = append(prices, p)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    prices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFlowHistory handles GET /api/v1/fetcher/flows/{code}/history
func (h *ChartHandler) GetFlowHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["code"]

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 1 month if not specified
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	if startDate == "" {
		start := time.Now().AddDate(0, -1, 0)
		startDate = start.Format("2006-01-02")
	}

	// Query database (join with daily_prices for price data)
	query := `
		SELECT
			f.trade_date,
			f.foreign_net_qty,
			f.inst_net_qty,
			f.indiv_net_qty,
			COALESCE(p.close_price, 0) as close_price,
			COALESCE(p.close_price - p.open_price, 0) as price_change,
			COALESCE((p.close_price - p.open_price) / NULLIF(p.open_price, 0) * 100, 0) as change_rate,
			COALESCE(p.volume, 0) as volume
		FROM data.investor_flow f
		LEFT JOIN data.daily_prices p ON f.stock_code = p.stock_code AND f.trade_date = p.trade_date
		WHERE f.stock_code = $1
			AND f.trade_date >= $2
			AND f.trade_date <= $3
		ORDER BY f.trade_date ASC
	`

	rows, err := h.pool.Query(r.Context(), query, symbol, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var flows []InvestorFlowResponse
	for rows.Next() {
		var f InvestorFlowResponse
		var tradeDate time.Time
		err := rows.Scan(
			&tradeDate,
			&f.ForeignNet,
			&f.InstNet,
			&f.RetailNet,
			&f.ClosePrice,
			&f.PriceChange,
			&f.ChangeRate,
			&f.Volume,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.Date = tradeDate.Format("2006-01-02")
		flows = append(flows, f)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    flows,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
