package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/response"
)

// WatchlistItem represents a watchlist item
type WatchlistItem struct {
	ID           int        `json:"id" db:"id"`
	StockCode    string     `json:"stock_code" db:"stock_code"`
	Category     string     `json:"category" db:"category"`
	Memo         *string    `json:"memo,omitempty" db:"memo"`
	TargetPrice  *int64     `json:"target_price,omitempty" db:"target_price"`
	AlertEnabled bool       `json:"alert_enabled" db:"alert_enabled"`
	AlertPrice   *int64     `json:"alert_price,omitempty" db:"alert_price"`
	AlertCondition *string  `json:"alert_condition,omitempty" db:"alert_condition"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`

	// AI 분석 (candidate용)
	GrokAnalysis     *string `json:"grok_analysis,omitempty" db:"grok_analysis"`
	GeminiAnalysis   *string `json:"gemini_analysis,omitempty" db:"gemini_analysis"`
	ChatGPTAnalysis  *string `json:"chatgpt_analysis,omitempty" db:"chatgpt_analysis"`
	ClaudeAnalysis   *string `json:"claude_analysis,omitempty" db:"claude_analysis"`

	// JOIN된 종목 정보
	StockName    string  `json:"stock_name" db:"stock_name"`
	Market       string  `json:"market" db:"market"`
	CurrentPrice *int64  `json:"current_price,omitempty" db:"current_price"`
	ChangeRate   *string `json:"change_rate,omitempty" db:"change_rate"`
}

// CreateWatchlistRequest represents create request
type CreateWatchlistRequest struct {
	StockCode   string  `json:"stock_code" binding:"required"`
	Category    string  `json:"category" binding:"required,oneof=watch candidate"`
	Memo        *string `json:"memo"`
	TargetPrice *int64  `json:"target_price"`
}

// UpdateWatchlistRequest represents update request
type UpdateWatchlistRequest struct {
	Memo           *string `json:"memo"`
	TargetPrice    *int64  `json:"target_price"`
	AlertEnabled   *bool   `json:"alert_enabled"`
	AlertPrice     *int64  `json:"alert_price"`
	AlertCondition *string `json:"alert_condition"`
}

// WatchlistHandler handles watchlist-related HTTP requests
type WatchlistHandler struct {
	db *sql.DB
}

// NewWatchlistHandler creates a new WatchlistHandler
func NewWatchlistHandler(db *sql.DB) *WatchlistHandler {
	return &WatchlistHandler{
		db: db,
	}
}

// List handles GET /api/v1/watchlist
func (h *WatchlistHandler) List(c *gin.Context) {
	query := `
		SELECT
			w.id, w.stock_code, w.category, w.memo, w.target_price,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis, w.claude_analysis,
			w.alert_enabled, w.alert_price, w.alert_condition,
			w.created_at, w.updated_at,
			COALESCE(s.stock_name, w.stock_code) as stock_name,
			COALESCE(s.market, '') as market,
			p.price as current_price,
			CAST(p.change_rate AS TEXT) as change_rate
		FROM portfolio.watchlist w
		LEFT JOIN market.stocks s ON w.stock_code = s.stock_code
		LEFT JOIN market.prices p ON w.stock_code = p.symbol
		ORDER BY w.created_at DESC
	`

	rows, err := h.db.QueryContext(c.Request.Context(), query)
	if err != nil {
		response.DatabaseError(c, err)
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
			response.DatabaseError(c, err)
			return
		}
		items = append(items, item)
	}

	// 카테고리별 분리
	watchItems := []WatchlistItem{}
	candidateItems := []WatchlistItem{}
	for _, item := range items {
		if item.Category == "watch" {
			watchItems = append(watchItems, item)
		} else if item.Category == "candidate" {
			candidateItems = append(candidateItems, item)
		}
	}

	result := gin.H{
		"watch":     watchItems,
		"candidate": candidateItems,
	}

	response.Success(c, result)
}

// ListWatch handles GET /api/v1/watchlist/watch
func (h *WatchlistHandler) ListWatch(c *gin.Context) {
	h.listByCategory(c, "watch")
}

// ListCandidate handles GET /api/v1/watchlist/candidate
func (h *WatchlistHandler) ListCandidate(c *gin.Context) {
	h.listByCategory(c, "candidate")
}

func (h *WatchlistHandler) listByCategory(c *gin.Context, category string) {
	query := `
		SELECT
			w.id, w.stock_code, w.category, w.memo, w.target_price,
			w.grok_analysis, w.gemini_analysis, w.chatgpt_analysis, w.claude_analysis,
			w.alert_enabled, w.alert_price, w.alert_condition,
			w.created_at, w.updated_at,
			COALESCE(s.stock_name, w.stock_code) as stock_name,
			COALESCE(s.market, '') as market,
			p.price as current_price,
			CAST(p.change_rate AS TEXT) as change_rate
		FROM portfolio.watchlist w
		LEFT JOIN market.stocks s ON w.stock_code = s.stock_code
		LEFT JOIN market.prices p ON w.stock_code = p.symbol
		WHERE w.category = $1
		ORDER BY w.created_at DESC
	`

	rows, err := h.db.QueryContext(c.Request.Context(), query, category)
	if err != nil {
		response.DatabaseError(c, err)
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
			response.DatabaseError(c, err)
			return
		}
		items = append(items, item)
	}

	response.Success(c, items)
}

// Create handles POST /api/v1/watchlist
func (h *WatchlistHandler) Create(c *gin.Context) {
	var req CreateWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter,
			"Invalid request body", err.Error())
		return
	}

	// 중복 체크
	var count int
	checkQuery := `SELECT COUNT(*) FROM portfolio.watchlist WHERE stock_code = $1 AND category = $2`
	err := h.db.QueryRowContext(c.Request.Context(), checkQuery, req.StockCode, req.Category).Scan(&count)
	if err != nil {
		response.DatabaseError(c, err)
		return
	}
	if count > 0 {
		response.ErrorWithDetails(c, http.StatusConflict, response.ErrCodeDuplicateEntry,
			"Already exists", "이미 등록된 종목입니다")
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
	err = h.db.QueryRowContext(c.Request.Context(), query,
		req.StockCode, req.Category, req.Memo, req.TargetPrice,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		response.DatabaseError(c, err)
		return
	}

	result := gin.H{
		"id":         id,
		"stock_code": req.StockCode,
		"category":   req.Category,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	response.Success(c, result)
}

// Update handles PUT /api/v1/watchlist/:id
func (h *WatchlistHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter,
			"Invalid ID", "ID must be a number")
		return
	}

	var req UpdateWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter,
			"Invalid request body", err.Error())
		return
	}

	// 존재 여부 확인
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM portfolio.watchlist WHERE id = $1)`
	err = h.db.QueryRowContext(c.Request.Context(), checkQuery, id).Scan(&exists)
	if err != nil {
		response.DatabaseError(c, err)
		return
	}
	if !exists {
		response.ErrorWithDetails(c, http.StatusNotFound, response.ErrCodeNotFound,
			"Not found", "종목을 찾을 수 없습니다")
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

	_, err = h.db.ExecContext(c.Request.Context(), query,
		id, req.Memo, req.TargetPrice, req.AlertEnabled, req.AlertPrice, req.AlertCondition,
	)

	if err != nil {
		response.DatabaseError(c, err)
		return
	}

	response.Success(c, gin.H{"id": id})
}

// Delete handles DELETE /api/v1/watchlist/:id
func (h *WatchlistHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, response.ErrCodeInvalidParameter,
			"Invalid ID", "ID must be a number")
		return
	}

	// 삭제
	query := `DELETE FROM portfolio.watchlist WHERE id = $1`
	result, err := h.db.ExecContext(c.Request.Context(), query, id)
	if err != nil {
		response.DatabaseError(c, err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		response.ErrorWithDetails(c, http.StatusNotFound, response.ErrCodeNotFound,
			"Not found", "종목을 찾을 수 없습니다")
		return
	}

	response.Success(c, gin.H{"id": id})
}
