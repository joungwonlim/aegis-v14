package routes

import (
	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// RegisterStocksRoutes registers all stocks-related routes
func RegisterStocksRoutes(router *mux.Router, dbPool *postgres.Pool) {
	// Create handler
	stocksHandler := handlers.NewStocksMuxHandler(dbPool.Pool)

	// API v1 routes
	v1 := router.PathPrefix("/api/v1/stocks").Subrouter()

	// GET /api/v1/stocks/search - Search stocks
	v1.HandleFunc("/search", stocksHandler.Search).Methods("GET")
}
