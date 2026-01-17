package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WatchlistMuxHandler handles watchlist-related HTTP requests (gorilla/mux version)
type WatchlistMuxHandler struct {
	pool *pgxpool.Pool
}

// NewWatchlistMuxHandler creates a new WatchlistMuxHandler
func NewWatchlistMuxHandler(pool *pgxpool.Pool) *WatchlistMuxHandler {
	return &WatchlistMuxHandler{
		pool: pool,
	}
}

// List handles GET /api/v1/watchlist
func (h *WatchlistMuxHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT
			w.id, w.stock_code, w.category, w.memo, w.target_price,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis, w.claude_analysis,
			w.alert_enabled, w.alert_price, w.alert_condition,
			w.created_at, w.updated_at,
			COALESCE(s.name, w.stock_code) as stock_name,
			COALESCE(s.market, '') as market,
			p.best_price as current_price,
			CAST(p.change_rate AS TEXT) as change_rate
		FROM portfolio.watchlist w
		LEFT JOIN market.stocks s ON w.stock_code = s.symbol
		LEFT JOIN market.prices_best p ON w.stock_code = p.symbol
		ORDER BY w.created_at DESC
	`

	rows, err := h.pool.Query(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	watchItems := []WatchlistItem{}
	candidateItems := []WatchlistItem{}

	for rows.Next() {
		var item WatchlistItem
		err := rows.Scan(
			&item.ID, &item.StockCode, &item.Category, &item.Memo, &item.TargetPrice,
			&item.GrokAnalysis, &item.GeminiAnalysis, &item.ChatGPTAnalysis, &item.ClaudeAnalysis,
			&item.AlertEnabled, &item.AlertPrice, &item.AlertCondition,
			&item.CreatedAt, &item.UpdatedAt,
			&item.StockName, &item.Market, &item.CurrentPrice, &item.ChangeRate,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if item.Category == "watch" {
			watchItems = append(watchItems, item)
		} else if item.Category == "candidate" {
			candidateItems = append(candidateItems, item)
		}
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"watch":     watchItems,
			"candidate": candidateItems,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListWatch handles GET /api/v1/watchlist/watch
func (h *WatchlistMuxHandler) ListWatch(w http.ResponseWriter, r *http.Request) {
	h.listByCategory(w, r, "watch")
}

// ListCandidate handles GET /api/v1/watchlist/candidate
func (h *WatchlistMuxHandler) ListCandidate(w http.ResponseWriter, r *http.Request) {
	h.listByCategory(w, r, "candidate")
}

func (h *WatchlistMuxHandler) listByCategory(w http.ResponseWriter, r *http.Request, category string) {
	query := `
		SELECT
			w.id, w.stock_code, w.category, w.memo, w.target_price,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis, w.claude_analysis,
			w.alert_enabled, w.alert_price, w.alert_condition,
			w.created_at, w.updated_at,
			COALESCE(s.name, w.stock_code) as stock_name,
			COALESCE(s.market, '') as market,
			p.best_price as current_price,
			CAST(p.change_rate AS TEXT) as change_rate
		FROM portfolio.watchlist w
		LEFT JOIN market.stocks s ON w.stock_code = s.symbol
		LEFT JOIN market.prices_best p ON w.stock_code = p.symbol
		WHERE w.category = $1
		ORDER BY w.created_at DESC
	`

	rows, err := h.pool.Query(r.Context(), query, category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []WatchlistItem{}
	for rows.Next() {
		var item WatchlistItem
		err := rows.Scan(
			&item.ID, &item.StockCode, &item.Category, &item.Memo, &item.TargetPrice,
			&item.GrokAnalysis, &item.GeminiAnalysis, &item.ChatGPTAnalysis, &item.ClaudeAnalysis,
			&item.AlertEnabled, &item.AlertPrice, &item.AlertCondition,
			&item.CreatedAt, &item.UpdatedAt,
			&item.StockName, &item.Market, &item.CurrentPrice, &item.ChangeRate,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create handles POST /api/v1/watchlist
func (h *WatchlistMuxHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 중복 체크
	var count int
	checkQuery := `SELECT COUNT(*) FROM portfolio.watchlist WHERE stock_code = $1 AND category = $2`
	err := h.pool.QueryRow(r.Context(), checkQuery, req.StockCode, req.Category).Scan(&count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		response := map[string]interface{}{
			"success": false,
			"error":   "이미 등록된 종목입니다",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 삽입
	query := `
		INSERT INTO portfolio.watchlist (stock_code, category, memo, target_price)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	var id int
	var createdAt, updatedAt time.Time
	err = h.pool.QueryRow(r.Context(), query,
		req.StockCode, req.Category, req.Memo, req.TargetPrice,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":         id,
			"stock_code": req.StockCode,
			"category":   req.Category,
			"created_at": createdAt,
			"updated_at": updatedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Update handles PUT /api/v1/watchlist/{id}
func (h *WatchlistMuxHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req UpdateWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 존재 여부 확인
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM portfolio.watchlist WHERE id = $1)`
	err = h.pool.QueryRow(r.Context(), checkQuery, id).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "종목을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// 업데이트
	query := `
		UPDATE portfolio.watchlist
		SET memo = COALESCE($2, memo),
		    target_price = COALESCE($3, target_price),
		    alert_enabled = COALESCE($4, alert_enabled),
		    alert_price = COALESCE($5, alert_price),
		    alert_condition = COALESCE($6, alert_condition)
		WHERE id = $1
	`

	_, err = h.pool.Exec(r.Context(), query,
		id, req.Memo, req.TargetPrice, req.AlertEnabled, req.AlertPrice, req.AlertCondition,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"id": id},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /api/v1/watchlist/{id}
func (h *WatchlistMuxHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 삭제
	query := `DELETE FROM portfolio.watchlist WHERE id = $1`
	result, err := h.pool.Exec(r.Context(), query, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "종목을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"id": id},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
