package reentry

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// evaluateAllCandidates evaluates all active candidates
func (s *Service) evaluateAllCandidates(ctx context.Context) error {
	// 1. Check control gate
	control, err := s.controlRepo.GetControl(ctx)
	if err != nil {
		return fmt.Errorf("get control: %w", err)
	}

	log.Debug().Str("mode", control.Mode).Msg("Reentry control mode")

	// 2. Load active candidates (COOLDOWN, WATCH, READY)
	candidates, err := s.candidateRepo.LoadActiveCandidates(ctx)
	if err != nil {
		return fmt.Errorf("load active candidates: %w", err)
	}

	if len(candidates) == 0 {
		return nil
	}

	log.Debug().Int("count", len(candidates)).Msg("Evaluating candidates")

	// 3. Evaluate each candidate
	for _, candidate := range candidates {
		if err := s.evaluateCandidate(ctx, candidate, control.Mode); err != nil {
			log.Error().
				Err(err).
				Str("candidate_id", candidate.CandidateID.String()).
				Str("symbol", candidate.Symbol).
				Msg("Candidate evaluation failed")
		}
	}

	return nil
}

// evaluateCandidate evaluates a single candidate and transitions FSM
func (s *Service) evaluateCandidate(ctx context.Context, candidate *reentry.ReentryCandidate, controlMode string) error {
	now := time.Now()

	// Update last eval timestamp
	if err := s.candidateRepo.UpdateLastEvalTS(ctx, candidate.CandidateID, now); err != nil {
		log.Warn().Err(err).Str("candidate_id", candidate.CandidateID.String()).Msg("Failed to update last eval ts")
	}

	switch candidate.State {
	case reentry.StateCooldown:
		return s.handleCooldownState(ctx, candidate, now)

	case reentry.StateWatch:
		return s.handleWatchState(ctx, candidate, now, controlMode)

	case reentry.StateReady:
		return s.handleReadyState(ctx, candidate, now, controlMode)

	default:
		// Terminal states (ENTERED, EXPIRED, BLOCKED) - do nothing
		return nil
	}
}

// handleCooldownState handles candidate in COOLDOWN state
func (s *Service) handleCooldownState(ctx context.Context, candidate *reentry.ReentryCandidate, now time.Time) error {
	// Check if cooldown period has passed
	if now.Before(candidate.CooldownUntil) {
		// Still in cooldown
		return nil
	}

	// Transition to WATCH
	if err := s.candidateRepo.UpdateCandidateState(ctx, candidate.CandidateID, reentry.StateWatch); err != nil {
		return fmt.Errorf("update state to WATCH: %w", err)
	}

	log.Info().
		Str("candidate_id", candidate.CandidateID.String()).
		Str("symbol", candidate.Symbol).
		Msg("Candidate → WATCH")

	return nil
}

// handleWatchState handles candidate in WATCH state
func (s *Service) handleWatchState(ctx context.Context, candidate *reentry.ReentryCandidate, now time.Time, controlMode string) error {
	// Check if max watch time exceeded
	maxWatchDuration := time.Duration(s.defaultProfile.Config.MaxWatchHours) * time.Hour
	if now.Sub(candidate.CooldownUntil) > maxWatchDuration {
		// Expired
		if err := s.candidateRepo.UpdateCandidateState(ctx, candidate.CandidateID, reentry.StateExpired); err != nil {
			return fmt.Errorf("update state to EXPIRED: %w", err)
		}

		log.Info().
			Str("candidate_id", candidate.CandidateID.String()).
			Str("symbol", candidate.Symbol).
			Msg("Candidate → EXPIRED (max watch time exceeded)")

		return nil
	}

	// TODO: Check reentry triggers (Rebound/Breakout/Chase)
	// For now, just log
	log.Debug().
		Str("candidate_id", candidate.CandidateID.String()).
		Str("symbol", candidate.Symbol).
		Msg("Candidate in WATCH state (trigger evaluation not implemented)")

	return nil
}

// handleReadyState handles candidate in READY state
func (s *Service) handleReadyState(ctx context.Context, candidate *reentry.ReentryCandidate, now time.Time, controlMode string) error {
	// Check control gate
	if controlMode == reentry.ControlModePauseEntry || controlMode == reentry.ControlModePauseAll {
		log.Debug().
			Str("candidate_id", candidate.CandidateID.String()).
			Str("control_mode", controlMode).
			Msg("Entry paused by control gate")
		return nil
	}

	// TODO: Create ENTRY intent
	// For now, just log
	log.Info().
		Str("candidate_id", candidate.CandidateID.String()).
		Str("symbol", candidate.Symbol).
		Msg("Candidate READY for entry (intent creation not implemented)")

	return nil
}
