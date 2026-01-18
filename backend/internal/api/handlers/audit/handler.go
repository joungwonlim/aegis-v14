package audit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/audit"
	auditService "github.com/wonny/aegis/v14/internal/service/audit"
)

// =============================================================================
// Handler
// =============================================================================

// Handler Audit API 핸들러
type Handler struct {
	service    *auditService.Service
	kisBuilder *auditService.KISAuditBuilder
}

// NewHandler 새 핸들러 생성
func NewHandler(service *auditService.Service) *Handler {
	return &Handler{service: service}
}

// SetKISBuilder KIS Builder 설정
func (h *Handler) SetKISBuilder(builder *auditService.KISAuditBuilder) {
	h.kisBuilder = builder
}

// =============================================================================
// Response Types
// =============================================================================

// PerformanceResponse 성과 응답
type PerformanceResponse struct {
	Success bool                      `json:"success"`
	Data    *audit.PerformanceReport  `json:"data,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

// DailyPnLResponse 일별 손익 응답
type DailyPnLResponse struct {
	Success bool            `json:"success"`
	Data    []audit.DailyPnL `json:"data"`
	Count   int             `json:"count"`
	Error   string          `json:"error,omitempty"`
}

// AttributionResponse 귀속 분석 응답
type AttributionResponse struct {
	Success bool                       `json:"success"`
	Data    *audit.AttributionAnalysis `json:"data,omitempty"`
	Error   string                     `json:"error,omitempty"`
}

// RiskMetricsResponse 리스크 지표 응답
type RiskMetricsResponse struct {
	Success bool               `json:"success"`
	Data    *audit.RiskMetrics `json:"data,omitempty"`
	Error   string             `json:"error,omitempty"`
}

// =============================================================================
// Performance Handlers
// =============================================================================

// GetPerformance handles GET /api/v1/audit/performance
func (h *Handler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	periodStr := r.URL.Query().Get("period")
	if periodStr == "" {
		periodStr = "1M"
	}

	period := audit.Period(periodStr)
	if !period.IsValid() {
		h.writeError(w, "Invalid period", http.StatusBadRequest)
		return
	}

	report, err := h.service.GetPerformanceReport(r.Context(), period)
	if err != nil {
		if err == audit.ErrInsufficientData {
			h.writeError(w, "Insufficient data for analysis", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("period", periodStr).Msg("Failed to get performance report")
		h.writeError(w, "Failed to get performance report", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, PerformanceResponse{
		Success: true,
		Data:    report,
	})
}

// GeneratePerformance handles POST /api/v1/audit/performance/generate
func (h *Handler) GeneratePerformance(w http.ResponseWriter, r *http.Request) {
	periodStr := r.URL.Query().Get("period")
	if periodStr == "" {
		periodStr = "1M"
	}

	period := audit.Period(periodStr)
	if !period.IsValid() {
		h.writeError(w, "Invalid period", http.StatusBadRequest)
		return
	}

	report, err := h.service.GeneratePerformanceReport(r.Context(), period)
	if err != nil {
		if err == audit.ErrInsufficientData {
			h.writeError(w, "Insufficient data for analysis", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("period", periodStr).Msg("Failed to generate performance report")
		h.writeError(w, "Failed to generate performance report", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, PerformanceResponse{
		Success: true,
		Data:    report,
	})
}

// =============================================================================
// Daily PnL Handlers
// =============================================================================

// GetDailyPnL handles GET /api/v1/audit/daily-pnl
func (h *Handler) GetDailyPnL(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	pnls, err := h.service.GetDailyPnLHistory(r.Context(), startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get daily PnL history")
		h.writeError(w, "Failed to get daily PnL", http.StatusInternalServerError)
		return
	}

	// Ensure data field is always present (empty array instead of null)
	if pnls == nil {
		pnls = make([]audit.DailyPnL, 0)
	}

	h.writeJSON(w, DailyPnLResponse{
		Success: true,
		Data:    pnls,
		Count:   len(pnls),
	})
}

// =============================================================================
// Attribution Handlers
// =============================================================================

// GetAttribution handles GET /api/v1/audit/attribution
func (h *Handler) GetAttribution(w http.ResponseWriter, r *http.Request) {
	periodStr := r.URL.Query().Get("period")
	if periodStr == "" {
		periodStr = "1M"
	}

	period := audit.Period(periodStr)
	if !period.IsValid() {
		h.writeError(w, "Invalid period", http.StatusBadRequest)
		return
	}

	analysis, err := h.service.GetAttributionAnalysis(r.Context(), period)
	if err != nil {
		if err == audit.ErrInsufficientData {
			h.writeError(w, "Insufficient data for analysis", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("period", periodStr).Msg("Failed to get attribution analysis")
		h.writeError(w, "Failed to get attribution analysis", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, AttributionResponse{
		Success: true,
		Data:    analysis,
	})
}

// =============================================================================
// Risk Handlers
// =============================================================================

// GetRiskMetrics handles GET /api/v1/audit/risk
func (h *Handler) GetRiskMetrics(w http.ResponseWriter, r *http.Request) {
	periodStr := r.URL.Query().Get("period")
	if periodStr == "" {
		periodStr = "1M"
	}

	period := audit.Period(periodStr)
	if !period.IsValid() {
		h.writeError(w, "Invalid period", http.StatusBadRequest)
		return
	}

	metrics, err := h.service.CalculateRiskMetrics(r.Context(), period)
	if err != nil {
		if err == audit.ErrInsufficientData {
			h.writeError(w, "Insufficient data for analysis", http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("period", periodStr).Msg("Failed to calculate risk metrics")
		h.writeError(w, "Failed to calculate risk metrics", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, RiskMetricsResponse{
		Success: true,
		Data:    metrics,
	})
}

// =============================================================================
// Snapshot Handlers
// =============================================================================

// GetSnapshot handles GET /api/v1/audit/snapshot/{date}
func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"]

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.writeError(w, "Invalid date format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	snapshots, err := h.service.GetSnapshotHistory(r.Context(), date, date)
	if err != nil || len(snapshots) == 0 {
		h.writeError(w, "Snapshot not found", http.StatusNotFound)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    snapshots[0],
	})
}

// GetSnapshotHistory handles GET /api/v1/audit/snapshots
func (h *Handler) GetSnapshotHistory(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	snapshots, err := h.service.GetSnapshotHistory(r.Context(), startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get snapshot history")
		h.writeError(w, "Failed to get snapshots", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    snapshots,
		"count":   len(snapshots),
	})
}

// =============================================================================
// KIS Data Builder Handlers
// =============================================================================

// BuildFromKIS handles POST /api/v1/audit/build-from-kis
func (h *Handler) BuildFromKIS(w http.ResponseWriter, r *http.Request) {
	if h.kisBuilder == nil {
		h.writeError(w, "KIS builder not configured", http.StatusInternalServerError)
		return
	}

	// Query parameters
	accountNo := r.URL.Query().Get("account_no")
	accountProductCode := r.URL.Query().Get("account_product_code")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// Defaults
	if accountProductCode == "" {
		accountProductCode = "01" // 기본값
	}

	// Parse dates
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			h.writeError(w, "Invalid start_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	} else {
		startDate = time.Now().AddDate(0, -1, 0) // 1개월 전
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			h.writeError(w, "Invalid end_date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	} else {
		endDate = time.Now()
	}

	// Build audit data from KIS
	err = h.kisBuilder.BuildFromKIS(r.Context(), accountNo, accountProductCode, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build audit data from KIS")
		h.writeError(w, fmt.Sprintf("Failed to build audit data: %v", err), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Audit data successfully built from KIS",
	})
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

func (h *Handler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
