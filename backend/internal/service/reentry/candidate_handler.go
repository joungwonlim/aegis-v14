package reentry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

// handleExitEvent processes a new ExitEvent and creates a reentry candidate
func (s *Service) handleExitEvent(ctx context.Context, event *execution.ExitEvent) error {
	// 1. Check control gate
	control, err := s.controlRepo.GetControl(ctx)
	if err != nil {
		return fmt.Errorf("get control: %w", err)
	}

	if control.Mode == reentry.ControlModePauseAll {
		log.Debug().
			Str("exit_event_id", event.ExitEventID.String()).
			Msg("Reentry paused (PAUSE_ALL), skipping candidate creation")
		return nil
	}

	// 2. Check if exit reason is eligible for reentry
	if !s.isEligibleForReentry(event.ExitReasonCode) {
		log.Debug().
			Str("exit_event_id", event.ExitEventID.String()).
			Str("exit_reason", event.ExitReasonCode).
			Msg("Exit reason not eligible for reentry")
		return nil
	}

	// 3. Check if candidate already exists for this exit event (idempotency)
	existing, err := s.candidateRepo.GetCandidateByExitEvent(ctx, event.ExitEventID)
	if err != nil && err != reentry.ErrCandidateNotFound {
		return fmt.Errorf("check existing candidate: %w", err)
	}

	if existing != nil {
		log.Debug().
			Str("exit_event_id", event.ExitEventID.String()).
			Str("candidate_id", existing.CandidateID.String()).
			Msg("Candidate already exists (idempotent)")
		return nil
	}

	// 4. Calculate cooldown period
	cooldownUntil := s.calculateCooldown(event.ExitTS, event.ExitReasonCode)

	// 5. Create reentry candidate
	candidate := &reentry.ReentryCandidate{
		CandidateID:      uuid.New(),
		ExitEventID:      event.ExitEventID,
		Symbol:           event.Symbol,
		OriginPositionID: event.PositionID,
		ExitReasonCode:   event.ExitReasonCode,
		ExitTS:           event.ExitTS,
		ExitPrice:        event.ExitAvgPrice,
		ExitProfileID:    event.ExitProfileID,
		CooldownUntil:    cooldownUntil,
		State:            reentry.StateCooldown,
		MaxReentries:     s.defaultProfile.Config.MaxReentries,
		ReentryCount:     0,
		ReentryProfileID: &s.defaultProfile.ProfileID,
		LastEvalTS:       nil,
		UpdatedTS:        time.Now(),
	}

	if err := s.candidateRepo.CreateCandidate(ctx, candidate); err != nil {
		return fmt.Errorf("create candidate: %w", err)
	}

	log.Info().
		Str("candidate_id", candidate.CandidateID.String()).
		Str("exit_event_id", event.ExitEventID.String()).
		Str("symbol", event.Symbol).
		Str("exit_reason", event.ExitReasonCode).
		Str("cooldown_until", cooldownUntil.Format(time.RFC3339)).
		Msg("Reentry candidate created")

	return nil
}

// isEligibleForReentry checks if exit reason is eligible for reentry
func (s *Service) isEligibleForReentry(exitReasonCode string) bool {
	switch exitReasonCode {
	case execution.ExitReasonSL1, execution.ExitReasonSL2:
		return true // SL → Rebound strategy
	case execution.ExitReasonTP1, execution.ExitReasonTP2, execution.ExitReasonTP3:
		return true // TP → Breakout/Chase strategy
	case execution.ExitReasonTrail:
		return true // Trailing → Breakout/Chase strategy
	case execution.ExitReasonTime:
		return false // TIME → No reentry (intentional exit)
	case execution.ExitReasonManual:
		return false // MANUAL → No reentry (user decision)
	case execution.ExitReasonBroker:
		return false // BROKER → No reentry (forced)
	default:
		return false
	}
}

// calculateCooldown calculates cooldown period based on exit reason
func (s *Service) calculateCooldown(exitTS time.Time, exitReasonCode string) time.Time {
	var cooldownSeconds int

	switch exitReasonCode {
	case execution.ExitReasonSL1, execution.ExitReasonSL2:
		cooldownSeconds = s.defaultProfile.Config.CooldownSL
	case execution.ExitReasonTP1, execution.ExitReasonTP2, execution.ExitReasonTP3, execution.ExitReasonTrail:
		cooldownSeconds = s.defaultProfile.Config.CooldownTP
	case execution.ExitReasonTime:
		cooldownSeconds = s.defaultProfile.Config.CooldownTime
	default:
		cooldownSeconds = s.defaultProfile.Config.CooldownSL // Default to SL cooldown
	}

	return exitTS.Add(time.Duration(cooldownSeconds) * time.Second)
}
