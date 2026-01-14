package universe

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// UniverseHandler handles universe endpoints
type UniverseHandler struct {
	service UniverseService
}

// UniverseService interface for service layer
type UniverseService interface {
	GetLatestSnapshot() (*universe.UniverseSnapshot, error)
	GetSnapshot(snapshotID string) (*universe.UniverseSnapshot, error)
	ListSnapshots(from, to time.Time) ([]*universe.UniverseSnapshot, error)
	GetUniverseSymbols() ([]string, error)
}

// NewUniverseHandler creates a new universe handler
func NewUniverseHandler(service UniverseService) *UniverseHandler {
	return &UniverseHandler{
		service: service,
	}
}

// GetLatestSnapshot handles GET /api/v1/universe/latest
func (h *UniverseHandler) GetLatestSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.GetLatestSnapshot()
	if err != nil {
		if err == universe.ErrSnapshotNotFound {
			http.Error(w, "Universe snapshot not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msg("Failed to get latest snapshot")
		http.Error(w, "Failed to get latest snapshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetSnapshot handles GET /api/v1/universe/snapshots/{snapshotId}
func (h *UniverseHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	snapshotID := vars["snapshotId"]

	snapshot, err := h.service.GetSnapshot(snapshotID)
	if err != nil {
		if err == universe.ErrSnapshotNotFound {
			http.Error(w, "Universe snapshot not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("snapshot_id", snapshotID).Msg("Failed to get snapshot")
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListSnapshotsResponse represents the response for listing snapshots
type ListSnapshotsResponse struct {
	Snapshots []*universe.UniverseSnapshot `json:"snapshots"`
	Count     int                           `json:"count"`
}

// ListSnapshots handles GET /api/v1/universe/snapshots
func (h *UniverseHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters (from, to)
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "Invalid from parameter", http.StatusBadRequest)
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, "Invalid to parameter", http.StatusBadRequest)
			return
		}
	} else {
		to = time.Now()
	}

	snapshots, err := h.service.ListSnapshots(from, to)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list snapshots")
		http.Error(w, "Failed to list snapshots", http.StatusInternalServerError)
		return
	}

	response := ListSnapshotsResponse{
		Snapshots: snapshots,
		Count:     len(snapshots),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetUniverseSymbolsResponse represents the response for universe symbols
type GetUniverseSymbolsResponse struct {
	Symbols []string `json:"symbols"`
	Count   int      `json:"count"`
}

// GetUniverseSymbols handles GET /api/v1/universe/symbols
func (h *UniverseHandler) GetUniverseSymbols(w http.ResponseWriter, r *http.Request) {
	symbols, err := h.service.GetUniverseSymbols()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get universe symbols")
		http.Error(w, "Failed to get universe symbols", http.StatusInternalServerError)
		return
	}

	response := GetUniverseSymbolsResponse{
		Symbols: symbols,
		Count:   len(symbols),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
