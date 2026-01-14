package routes

import (
	"github.com/gorilla/mux"
	exitHandlers "github.com/wonny/aegis/v14/internal/api/handlers/exit"
	exitService "github.com/wonny/aegis/v14/internal/service/exit"
)

// RegisterExitRoutes registers all exit-related routes
func RegisterExitRoutes(router *mux.Router, exitSvc *exitService.Service) {
	// Create handlers
	controlHandler := exitHandlers.NewControlHandler(exitSvc)
	positionHandler := exitHandlers.NewPositionHandler(exitSvc)

	// Control endpoints
	router.HandleFunc("/api/v1/exit/control", controlHandler.GetControl).Methods("GET")
	router.HandleFunc("/api/v1/exit/control", controlHandler.UpdateControl).Methods("POST")

	// Position endpoints
	router.HandleFunc("/api/v1/exit/positions/{positionId}/manual", positionHandler.CreateManualExit).Methods("POST")
	router.HandleFunc("/api/v1/exit/positions/{positionId}/state", positionHandler.GetPositionState).Methods("GET")
}
