package routes

import (
	"github.com/gorilla/mux"
	reentryHandlers "github.com/wonny/aegis/v14/internal/api/handlers/reentry"
)

// RegisterReentryRoutes registers all reentry-related routes
func RegisterReentryRoutes(
	router *mux.Router,
	candidateHandler *reentryHandlers.CandidateHandler,
	controlHandler *reentryHandlers.ControlHandler,
	profileHandler *reentryHandlers.ProfileHandler,
) {
	// Candidate endpoints
	router.HandleFunc("/api/v1/reentry/candidates", candidateHandler.ListCandidates).Methods("GET")
	router.HandleFunc("/api/v1/reentry/candidates/{candidateId}", candidateHandler.GetCandidate).Methods("GET")

	// Control endpoints
	router.HandleFunc("/api/v1/reentry/control", controlHandler.GetControl).Methods("GET")
	router.HandleFunc("/api/v1/reentry/control", controlHandler.UpdateControl).Methods("PUT")

	// Profile endpoints
	router.HandleFunc("/api/v1/reentry/profiles", profileHandler.ListProfiles).Methods("GET")
	router.HandleFunc("/api/v1/reentry/profiles/default", profileHandler.GetDefaultProfile).Methods("GET")
	router.HandleFunc("/api/v1/reentry/profiles/{profileId}", profileHandler.GetProfile).Methods("GET")
}
