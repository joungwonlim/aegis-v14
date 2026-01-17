package exit

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	exitService "github.com/wonny/aegis/v14/internal/service/exit"
)

// ProfileHandler handles exit profile endpoints
type ProfileHandler struct {
	exitSvc *exitService.Service
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(exitSvc *exitService.Service) *ProfileHandler {
	return &ProfileHandler{
		exitSvc: exitSvc,
	}
}

// ProfileResponse represents exit profile API response
type ProfileResponse struct {
	ProfileID   string                   `json:"profile_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Config      exit.ExitProfileConfig   `json:"config"`
	IsActive    bool                     `json:"is_active"`
	CreatedBy   string                   `json:"created_by"`
	CreatedTS   string                   `json:"created_ts"`
}

// ProfilesResponse represents list of profiles response
type ProfilesResponse struct {
	Profiles []ProfileResponse `json:"profiles"`
}

// CreateProfileRequest represents POST /api/v1/exit/profiles request
type CreateProfileRequest struct {
	ProfileID   string                   `json:"profile_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Config      exit.ExitProfileConfig   `json:"config"`
	CreatedBy   string                   `json:"created_by"`
}

// GetProfiles handles GET /api/v1/exit/profiles
func (h *ProfileHandler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check query param for active_only filter
	activeOnly := r.URL.Query().Get("active_only") == "true"

	// Get profiles
	var profiles []*exit.ExitProfile
	var err error

	if activeOnly {
		profiles, err = h.exitSvc.GetActiveProfiles(ctx)
	} else {
		profiles, err = h.exitSvc.GetAllProfiles(ctx)
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to get profiles")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response
	resp := ProfilesResponse{
		Profiles: make([]ProfileResponse, len(profiles)),
	}

	for i, profile := range profiles {
		resp.Profiles[i] = ProfileResponse{
			ProfileID:   profile.ProfileID,
			Name:        profile.Name,
			Description: profile.Description,
			Config:      profile.Config,
			IsActive:    profile.IsActive,
			CreatedBy:   profile.CreatedBy,
			CreatedTS:   profile.CreatedTS.Format("2006-01-02T15:04:05-07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// CreateProfile handles POST /api/v1/exit/profiles
func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ProfileID == "" {
		http.Error(w, "profile_id is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}

	// Create profile
	profile := &exit.ExitProfile{
		ProfileID:   req.ProfileID,
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
		IsActive:    true,
		CreatedBy:   req.CreatedBy,
	}

	err := h.exitSvc.CreateOrUpdateProfile(ctx, profile)
	if err != nil {
		log.Error().Err(err).Str("profile_id", req.ProfileID).Msg("Failed to create profile")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("profile_id", req.ProfileID).
		Str("name", req.Name).
		Str("created_by", req.CreatedBy).
		Msg("Exit profile created")

	w.WriteHeader(http.StatusCreated)
}
