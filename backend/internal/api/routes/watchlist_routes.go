package routes

import (
	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// RegisterWatchlistRoutes registers all watchlist-related routes
func RegisterWatchlistRoutes(router *mux.Router, dbPool *postgres.Pool) {
	// Create handler
	watchlistHandler := handlers.NewWatchlistMuxHandler(dbPool.Pool)

	// API v1 routes
	v1 := router.PathPrefix("/api/v1/watchlist").Subrouter()

	// GET /api/v1/watchlist - List all (categorized)
	v1.HandleFunc("", watchlistHandler.List).Methods("GET")

	// GET /api/v1/watchlist/watch - List watch category
	v1.HandleFunc("/watch", watchlistHandler.ListWatch).Methods("GET")

	// GET /api/v1/watchlist/candidate - List candidate category
	v1.HandleFunc("/candidate", watchlistHandler.ListCandidate).Methods("GET")

	// POST /api/v1/watchlist - Create new item
	v1.HandleFunc("", watchlistHandler.Create).Methods("POST")

	// PUT /api/v1/watchlist/{id} - Update item
	v1.HandleFunc("/{id}", watchlistHandler.Update).Methods("PUT")

	// DELETE /api/v1/watchlist/{id} - Delete item
	v1.HandleFunc("/{id}", watchlistHandler.Delete).Methods("DELETE")
}
