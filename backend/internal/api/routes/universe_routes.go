package routes

import (
	"github.com/gorilla/mux"
	universeHandlers "github.com/wonny/aegis/v14/internal/api/handlers/universe"
)

// RegisterUniverseRoutes registers all universe-related routes
func RegisterUniverseRoutes(
	router *mux.Router,
	universeHandler *universeHandlers.UniverseHandler,
) {
	// Universe endpoints
	router.HandleFunc("/api/v1/universe/latest", universeHandler.GetLatestSnapshot).Methods("GET")
	router.HandleFunc("/api/v1/universe/snapshots", universeHandler.ListSnapshots).Methods("GET")
	router.HandleFunc("/api/v1/universe/snapshots/{snapshotId}", universeHandler.GetSnapshot).Methods("GET")
	router.HandleFunc("/api/v1/universe/symbols", universeHandler.GetUniverseSymbols).Methods("GET")
}
