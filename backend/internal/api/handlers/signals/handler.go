package signals

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// SignalService 신호 서비스 인터페이스
type SignalService interface {
	// 신호 생성
	GenerateSignals(ctx context.Context) (*signals.SignalSnapshot, error)

	// 스냅샷 조회
	GetLatestSnapshot(ctx context.Context) (*signals.SignalSnapshot, error)
	GetSnapshotByID(ctx context.Context, snapshotID string) (*signals.SignalSnapshot, error)

	// 신호 조회
	GetSignalBySymbol(ctx context.Context, snapshotID, symbol string) (*signals.Signal, error)
}

// FactorService 팩터 서비스 인터페이스
type FactorService interface {
	GetMomentumFactors(ctx context.Context, symbol string) (*signals.MomentumFactors, error)
	GetQualityFactors(ctx context.Context, symbol string) (*signals.QualityFactors, error)
	GetValueFactors(ctx context.Context, symbol string) (*signals.ValueFactors, error)
	GetTechnicalFactors(ctx context.Context, symbol string) (*signals.TechnicalFactors, error)
	GetFlowFactors(ctx context.Context, symbol string) (*signals.FlowFactors, error)
	GetEventFactors(ctx context.Context, symbol string) (*signals.EventFactors, error)
}

// Handler Signals API 핸들러
type Handler struct {
	signalService SignalService
	factorService FactorService
}

// NewHandler 핸들러 생성
func NewHandler(signalService SignalService, factorService FactorService) *Handler {
	return &Handler{
		signalService: signalService,
		factorService: factorService,
	}
}

// =============================================================================
// Response Types
// =============================================================================

// SnapshotResponse 스냅샷 응답
type SnapshotResponse struct {
	SnapshotID  string              `json:"snapshot_id"`
	UniverseID  string              `json:"universe_id"`
	GeneratedAt time.Time           `json:"generated_at"`
	TotalCount  int                 `json:"total_count"`
	BuyCount    int                 `json:"buy_count"`
	SellCount   int                 `json:"sell_count"`
	Stats       signals.SignalStats `json:"stats"`
}

// SignalListResponse 신호 목록 응답
type SignalListResponse struct {
	Signals []signals.Signal `json:"signals"`
	Count   int              `json:"count"`
}

// FactorsResponse 6팩터 응답
type FactorsResponse struct {
	Symbol    string                    `json:"symbol"`
	CalcDate  string                    `json:"calc_date"`
	Momentum  *signals.MomentumFactors  `json:"momentum,omitempty"`
	Technical *signals.TechnicalFactors `json:"technical,omitempty"`
	Value     *signals.ValueFactors     `json:"value,omitempty"`
	Quality   *signals.QualityFactors   `json:"quality,omitempty"`
	Flow      *signals.FlowFactors      `json:"flow,omitempty"`
	Event     *signals.EventFactors     `json:"event,omitempty"`
}

// GenerateResponse 신호 생성 응답
type GenerateResponse struct {
	Success    bool   `json:"success"`
	SnapshotID string `json:"snapshot_id"`
	TotalCount int    `json:"total_count"`
	BuyCount   int    `json:"buy_count"`
	SellCount  int    `json:"sell_count"`
}

// =============================================================================
// Snapshot Handlers
// =============================================================================

// GetLatestSnapshot handles GET /api/v1/signals/snapshot/latest
func (h *Handler) GetLatestSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.signalService.GetLatestSnapshot(r.Context())
	if err != nil {
		if err == signals.ErrSnapshotNotFound {
			http.Error(w, "No snapshot found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msg("Failed to get latest snapshot")
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	response := SnapshotResponse{
		SnapshotID:  snapshot.SnapshotID,
		UniverseID:  snapshot.UniverseID,
		GeneratedAt: snapshot.GeneratedAt,
		TotalCount:  snapshot.TotalCount,
		BuyCount:    len(snapshot.BuySignals),
		SellCount:   len(snapshot.SellSignals),
		Stats:       snapshot.Stats,
	}

	h.writeJSON(w, response)
}

// GetSnapshotByID handles GET /api/v1/signals/snapshot/{id}
func (h *Handler) GetSnapshotByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snapshotID := vars["id"]

	snapshot, err := h.signalService.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		if err == signals.ErrSnapshotNotFound {
			http.Error(w, "Snapshot not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("snapshot_id", snapshotID).Msg("Failed to get snapshot")
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, snapshot)
}

// GetBuySignals handles GET /api/v1/signals/snapshot/{id}/buy
func (h *Handler) GetBuySignals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snapshotID := vars["id"]

	snapshot, err := h.signalService.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		if err == signals.ErrSnapshotNotFound {
			http.Error(w, "Snapshot not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("snapshot_id", snapshotID).Msg("Failed to get snapshot")
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	response := SignalListResponse{
		Signals: snapshot.BuySignals,
		Count:   len(snapshot.BuySignals),
	}

	h.writeJSON(w, response)
}

// GetSellSignals handles GET /api/v1/signals/snapshot/{id}/sell
func (h *Handler) GetSellSignals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snapshotID := vars["id"]

	snapshot, err := h.signalService.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		if err == signals.ErrSnapshotNotFound {
			http.Error(w, "Snapshot not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("snapshot_id", snapshotID).Msg("Failed to get snapshot")
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	response := SignalListResponse{
		Signals: snapshot.SellSignals,
		Count:   len(snapshot.SellSignals),
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Signal Handlers
// =============================================================================

// GetSignalBySymbol handles GET /api/v1/signals/{snapshot_id}/{symbol}
func (h *Handler) GetSignalBySymbol(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snapshotID := vars["snapshot_id"]
	symbol := vars["symbol"]

	signal, err := h.signalService.GetSignalBySymbol(r.Context(), snapshotID, symbol)
	if err != nil {
		if err == signals.ErrSignalNotFound {
			http.Error(w, "Signal not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).
			Str("snapshot_id", snapshotID).
			Str("symbol", symbol).
			Msg("Failed to get signal")
		http.Error(w, "Failed to get signal", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, signal)
}

// GenerateSignals handles POST /api/v1/signals/generate
func (h *Handler) GenerateSignals(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.signalService.GenerateSignals(r.Context())
	if err != nil {
		if err == signals.ErrUniverseNotReady {
			http.Error(w, "Universe not ready", http.StatusServiceUnavailable)
			return
		}
		log.Error().Err(err).Msg("Failed to generate signals")
		http.Error(w, "Failed to generate signals", http.StatusInternalServerError)
		return
	}

	response := GenerateResponse{
		Success:    true,
		SnapshotID: snapshot.SnapshotID,
		TotalCount: snapshot.TotalCount,
		BuyCount:   len(snapshot.BuySignals),
		SellCount:  len(snapshot.SellSignals),
	}

	h.writeJSON(w, response)
}

// =============================================================================
// Factor Handlers
// =============================================================================

// GetFactors handles GET /api/v1/signals/factors/{symbol}
func (h *Handler) GetFactors(w http.ResponseWriter, r *http.Request) {
	if h.factorService == nil {
		http.Error(w, "Factor service not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	symbol := vars["symbol"]
	ctx := r.Context()

	response := FactorsResponse{
		Symbol:   symbol,
		CalcDate: time.Now().Format("2006-01-02"),
	}

	// 각 팩터 조회 (실패해도 계속 진행)
	if momentum, err := h.factorService.GetMomentumFactors(ctx, symbol); err == nil {
		response.Momentum = momentum
	}
	if technical, err := h.factorService.GetTechnicalFactors(ctx, symbol); err == nil {
		response.Technical = technical
	}
	if value, err := h.factorService.GetValueFactors(ctx, symbol); err == nil {
		response.Value = value
	}
	if quality, err := h.factorService.GetQualityFactors(ctx, symbol); err == nil {
		response.Quality = quality
	}
	if flow, err := h.factorService.GetFlowFactors(ctx, symbol); err == nil {
		response.Flow = flow
	}
	if event, err := h.factorService.GetEventFactors(ctx, symbol); err == nil {
		response.Event = event
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
