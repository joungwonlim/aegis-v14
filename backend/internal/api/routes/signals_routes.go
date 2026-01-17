package routes

import (
	"github.com/gorilla/mux"
	signalsHandlers "github.com/wonny/aegis/v14/internal/api/handlers/signals"
)

// RegisterSignalsRoutes Signals API 라우트 등록
func RegisterSignalsRoutes(router *mux.Router, signalsHandler *signalsHandlers.Handler) {
	// Snapshot endpoints
	router.HandleFunc("/api/v1/signals/snapshot/latest", signalsHandler.GetLatestSnapshot).Methods("GET")
	router.HandleFunc("/api/v1/signals/snapshot/{id}", signalsHandler.GetSnapshotByID).Methods("GET")
	router.HandleFunc("/api/v1/signals/snapshot/{id}/buy", signalsHandler.GetBuySignals).Methods("GET")
	router.HandleFunc("/api/v1/signals/snapshot/{id}/sell", signalsHandler.GetSellSignals).Methods("GET")

	// Signal endpoints
	router.HandleFunc("/api/v1/signals/{snapshot_id}/{symbol}", signalsHandler.GetSignalBySymbol).Methods("GET")

	// Factor endpoints
	router.HandleFunc("/api/v1/signals/factors/{symbol}", signalsHandler.GetFactors).Methods("GET")

	// Generation endpoints
	router.HandleFunc("/api/v1/signals/generate", signalsHandler.GenerateSignals).Methods("POST")
}
