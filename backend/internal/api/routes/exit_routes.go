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
	profileHandler := exitHandlers.NewProfileHandler(exitSvc)
	overrideHandler := exitHandlers.NewOverrideHandler(exitSvc)

	// Control endpoints
	router.HandleFunc("/api/v1/exit/control", controlHandler.GetControl).Methods("GET")
	router.HandleFunc("/api/v1/exit/control", controlHandler.UpdateControl).Methods("POST")

	// Position endpoints
	router.HandleFunc("/api/v1/exit/positions/{positionId}/manual", positionHandler.CreateManualExit).Methods("POST")
	router.HandleFunc("/api/v1/exit/positions/{positionId}/state", positionHandler.GetPositionState).Methods("GET")

	// Profile endpoints
	router.HandleFunc("/api/v1/exit/profiles", profileHandler.GetProfiles).Methods("GET")
	router.HandleFunc("/api/v1/exit/profiles", profileHandler.CreateProfile).Methods("POST")

	// Symbol override endpoints
	router.HandleFunc("/api/v1/exit/overrides/{symbol}", overrideHandler.GetOverride).Methods("GET")
	router.HandleFunc("/api/v1/exit/overrides/{symbol}", overrideHandler.SetOverride).Methods("POST")
	router.HandleFunc("/api/v1/exit/overrides/{symbol}", overrideHandler.DeleteOverride).Methods("DELETE")
}
