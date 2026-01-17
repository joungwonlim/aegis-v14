package routes

import (
	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// RegisterChartRoutes registers chart-related routes for StockDetailSheet
func RegisterChartRoutes(router *mux.Router, dbPool *postgres.Pool) {
	// Create handler
	chartHandler := handlers.NewChartHandler(dbPool.Pool)

	// API v1 routes
	v1 := router.PathPrefix("/api/v1/fetcher").Subrouter()

	// Price history for charts
	v1.HandleFunc("/prices/{code}/history", chartHandler.GetPriceHistory).Methods("GET")

	// Flow history for charts
	v1.HandleFunc("/flows/{code}/history", chartHandler.GetFlowHistory).Methods("GET")
}
