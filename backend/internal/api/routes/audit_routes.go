package routes

import (
	"github.com/gorilla/mux"
	auditHandlers "github.com/wonny/aegis/v14/internal/api/handlers/audit"
)

// RegisterAuditRoutes Audit API 라우트 등록
func RegisterAuditRoutes(router *mux.Router, auditHandler *auditHandlers.Handler) {
	// Performance endpoints
	router.HandleFunc("/api/v1/audit/performance", auditHandler.GetPerformance).Methods("GET")
	router.HandleFunc("/api/v1/audit/performance/generate", auditHandler.GeneratePerformance).Methods("POST")

	// Daily PnL endpoints
	router.HandleFunc("/api/v1/audit/daily-pnl", auditHandler.GetDailyPnL).Methods("GET")

	// Attribution endpoints
	router.HandleFunc("/api/v1/audit/attribution", auditHandler.GetAttribution).Methods("GET")

	// Risk endpoints
	router.HandleFunc("/api/v1/audit/risk", auditHandler.GetRiskMetrics).Methods("GET")

	// Snapshot endpoints
	router.HandleFunc("/api/v1/audit/snapshot/{date}", auditHandler.GetSnapshot).Methods("GET")
	router.HandleFunc("/api/v1/audit/snapshots", auditHandler.GetSnapshotHistory).Methods("GET")

	// KIS Data Builder endpoint
	router.HandleFunc("/api/v1/audit/build-from-kis", auditHandler.BuildFromKIS).Methods("POST")
}
