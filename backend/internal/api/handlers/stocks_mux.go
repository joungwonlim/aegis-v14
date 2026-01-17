package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

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

// StockListItem represents a stock in list response
type StockListItem struct {
	StockCode    string   `json:"stock_code"`
	StockName    string   `json:"stock_name"`
	Market       string   `json:"market"`
	Sector       string   `json:"sector,omitempty"`
	Status       string   `json:"status"`
	IsTradable   bool     `json:"is_tradable"`
	CurrentPrice *int64   `json:"current_price,omitempty"`
	ChangeRate   *float64 `json:"change_rate,omitempty"`
}

// Pagination represents pagination metadata
type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalCount  int `json:"total_count"`
	Limit       int `json:"limit"`
}

// List handles GET /api/v1/stocks - 전체 종목 목록 조회 (페이지네이션 지원)
func (h *StocksMuxHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	market := r.URL.Query().Get("market")
	search := r.URL.Query().Get("search")

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argPos := 1

	if market != "" {
		whereConditions = append(whereConditions, "s.market = $"+strconv.Itoa(argPos))
		args = append(args, market)
		argPos++
	}

	if search != "" {
		whereConditions = append(whereConditions, "(s.symbol ILIKE '%' || $"+strconv.Itoa(argPos)+" || '%' OR s.name ILIKE '%' || $"+strconv.Itoa(argPos)+" || '%')")
		args = append(args, search)
		argPos++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM market.stocks s
		` + whereClause

	var totalCount int
	err := h.pool.QueryRow(r.Context(), countQuery, args...).Scan(&totalCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	// Query stocks with prices
	orderClause := "ORDER BY s.name"
	if search != "" {
		// 검색 시 관련도 순 정렬
		orderClause = `ORDER BY
			CASE
				WHEN s.symbol = $` + strconv.Itoa(argPos-1) + ` THEN 1
				WHEN s.name = $` + strconv.Itoa(argPos-1) + ` THEN 2
				WHEN s.symbol ILIKE $` + strconv.Itoa(argPos-1) + ` || '%' THEN 3
				WHEN s.name ILIKE $` + strconv.Itoa(argPos-1) + ` || '%' THEN 4
				ELSE 5
			END,
			s.name`
	}

	dataQuery := `
		SELECT
			s.symbol,
			s.name,
			s.market,
			COALESCE(s.sector, '') as sector,
			s.status,
			s.is_tradable,
			p.best_price,
			p.change_rate
		FROM market.stocks s
		LEFT JOIN market.prices_best p ON s.symbol = TRIM(p.symbol)
		` + whereClause + `
		` + orderClause + `
		LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)

	args = append(args, limit, offset)

	rows, err := h.pool.Query(r.Context(), dataQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stocks := []StockListItem{}
	for rows.Next() {
		var stock StockListItem
		err := rows.Scan(
			&stock.StockCode,
			&stock.StockName,
			&stock.Market,
			&stock.Sector,
			&stock.Status,
			&stock.IsTradable,
			&stock.CurrentPrice,
			&stock.ChangeRate,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stocks = append(stocks, stock)
	}

	pagination := Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
		Limit:       limit,
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"stocks":     stocks,
			"pagination": pagination,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchResult represents a stock search result (legacy endpoint)
type SearchResult struct {
	StockCode string `json:"stock_code"`
	StockName string `json:"stock_name"`
	Market    string `json:"market"`
	Sector    string `json:"sector,omitempty"`
}

// Search handles GET /api/v1/stocks/search (legacy)
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
