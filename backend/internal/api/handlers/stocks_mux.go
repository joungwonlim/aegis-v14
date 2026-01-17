package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StocksMuxHandler handles stocks-related HTTP requests
type StocksMuxHandler struct {
	pool *pgxpool.Pool
}

// NewStocksMuxHandler creates a new StocksMuxHandler
func NewStocksMuxHandler(pool *pgxpool.Pool) *StocksMuxHandler {
	return &StocksMuxHandler{
		pool: pool,
	}
}

// SearchResult represents a stock search result
type SearchResult struct {
	StockCode string `json:"stock_code"`
	StockName string `json:"stock_name"`
	Market    string `json:"market"`
	Sector    string `json:"sector,omitempty"`
}

// List handles GET /api/v1/stocks - 전체 종목 목록 조회
func (h *StocksMuxHandler) List(w http.ResponseWriter, r *http.Request) {
	market := r.URL.Query().Get("market")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := json.Number(limitStr).Int64(); err == nil && l > 0 {
			limit = int(l)
		}
	}

	var sqlQuery string
	var args []interface{}

	if market != "" {
		sqlQuery = `
			SELECT symbol, name, market, COALESCE(sector, '') as sector
			FROM market.stocks
			WHERE market = $1
			ORDER BY name
			LIMIT $2
		`
		args = []interface{}{market, limit}
	} else {
		sqlQuery = `
			SELECT symbol, name, market, COALESCE(sector, '') as sector
			FROM market.stocks
			ORDER BY name
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	rows, err := h.pool.Query(r.Context(), sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.StockCode, &result.StockName, &result.Market, &result.Sector); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    results,
		"count":   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Search handles GET /api/v1/stocks/search
func (h *StocksMuxHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response := map[string]interface{}{
			"success": true,
			"data":    []SearchResult{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// SQL 쿼리: symbol 또는 name에서 검색 (ILIKE는 대소문자 무시)
	sqlQuery := `
		SELECT symbol, name, market, COALESCE(sector, '') as sector
		FROM market.stocks
		WHERE symbol ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%'
		ORDER BY
			CASE
				WHEN symbol = $1 THEN 1
				WHEN name = $1 THEN 2
				WHEN symbol ILIKE $1 || '%' THEN 3
				WHEN name ILIKE $1 || '%' THEN 4
				ELSE 5
			END,
			name
		LIMIT 20
	`

	rows, err := h.pool.Query(r.Context(), sqlQuery, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var result SearchResult
		err := rows.Scan(&result.StockCode, &result.StockName, &result.Market, &result.Sector)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
