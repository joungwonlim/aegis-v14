package reentry

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// CandidateHandler handles reentry candidate endpoints
type CandidateHandler struct {
	service CandidateService
}

// CandidateService interface for service layer
type CandidateService interface {
	GetCandidate(candidateID uuid.UUID) (*reentry.ReentryCandidate, error)
	LoadCandidatesByState(states []string) ([]*reentry.ReentryCandidate, error)
	LoadActiveCandidates() ([]*reentry.ReentryCandidate, error)
}

// NewCandidateHandler creates a new candidate handler
func NewCandidateHandler(service CandidateService) *CandidateHandler {
	return &CandidateHandler{
		service: service,
	}
}

// ListCandidatesResponse represents the response for listing candidates
type ListCandidatesResponse struct {
	Candidates []*reentry.ReentryCandidate `json:"candidates"`
	Count      int                          `json:"count"`
}

// ListCandidates handles GET /api/v1/reentry/candidates
func (h *CandidateHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	queryState := r.URL.Query().Get("state")

	var candidates []*reentry.ReentryCandidate
	var err error

	if queryState == "" || queryState == "active" {
		// Default: load active candidates (COOLDOWN, WATCH, READY)
		candidates, err = h.service.LoadActiveCandidates()
	} else {
		// Filter by specific state
		candidates, err = h.service.LoadCandidatesByState([]string{queryState})
	}

	if err != nil {
		log.Error().Err(err).Str("state", queryState).Msg("Failed to load candidates")
		http.Error(w, "Failed to load candidates", http.StatusInternalServerError)
		return
	}

	response := ListCandidatesResponse{
		Candidates: candidates,
		Count:      len(candidates),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetCandidate handles GET /api/v1/reentry/candidates/{candidateId}
func (h *CandidateHandler) GetCandidate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	candidateIDStr := vars["candidateId"]

	candidateID, err := uuid.Parse(candidateIDStr)
	if err != nil {
		http.Error(w, "Invalid candidate ID", http.StatusBadRequest)
		return
	}

	candidate, err := h.service.GetCandidate(candidateID)
	if err != nil {
		if err == reentry.ErrCandidateNotFound {
			http.Error(w, "Candidate not found", http.StatusNotFound)
			return
		}
		log.Error().Err(err).Str("candidate_id", candidateID.String()).Msg("Failed to get candidate")
		http.Error(w, "Failed to get candidate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(candidate); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
