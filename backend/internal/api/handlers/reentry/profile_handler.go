package reentry

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// ProfileHandler handles reentry profile endpoints
type ProfileHandler struct {
	service ProfileService
}

// ProfileService interface for service layer
type ProfileService interface {
	GetProfile(profileID string) (*reentry.ReentryProfile, error)
	GetDefaultProfile() (*reentry.ReentryProfile, error)
	ListProfiles() ([]*reentry.ReentryProfile, error)
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(service ProfileService) *ProfileHandler {
	return &ProfileHandler{
		service: service,
	}
}

// ListProfilesResponse represents the response for listing profiles
type ListProfilesResponse struct {
	Profiles []*reentry.ReentryProfile `json:"profiles"`
	Count    int                        `json:"count"`
}

// ListProfiles handles GET /api/v1/reentry/profiles
func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.ListProfiles()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list profiles")
		http.Error(w, "Failed to list profiles", http.StatusInternalServerError)
		return
	}

	response := ListProfilesResponse{
		Profiles: profiles,
		Count:    len(profiles),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetProfile handles GET /api/v1/reentry/profiles/{profileId}
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profileId"]

	profile, err := h.service.GetProfile(profileID)
	if err != nil {
		if err == reentry.ErrProfileNotFound {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("profile_id", profileID).Msg("Failed to get profile")
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetDefaultProfile handles GET /api/v1/reentry/profiles/default
func (h *ProfileHandler) GetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetDefaultProfile()
	if err != nil {
		if err == reentry.ErrProfileNotFound {
			http.Error(w, "Default profile not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Msg("Failed to get default profile")
		http.Error(w, "Failed to get default profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
