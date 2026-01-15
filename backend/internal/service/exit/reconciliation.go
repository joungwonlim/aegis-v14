package exit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ReconcileIntents reconciles Intent states with actual Holdings and Fills
// This prevents duplicate/stale intents and ensures data consistency
//
// Reconciliation checks:
// 1. Duplicate intents (same position + reason) → cancel older ones
// 2. Submitted/Filled intents without actual fills → mark as FAILED (future)
func (s *Service) ReconcileIntents(ctx context.Context) error {
	log.Debug().Msg("🔄 Starting Intent Reconciliation")

	// 1. Find and cancel duplicate intents
	if err := s.cancelDuplicateIntents(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to cancel duplicate intents (non-fatal)")
	}

	log.Debug().Msg("✅ Intent Reconciliation completed")
	return nil
}

// cancelDuplicateIntents finds and cancels duplicate intents for the same position+reason
func (s *Service) cancelDuplicateIntents(ctx context.Context) error {
	// Get recent intents (last 500)
	intents, err := s.intentRepo.GetRecentIntents(ctx, 500)
	if err != nil {
		return err
	}

	// Group by position_id + reason_code
	type intentKey struct {
		positionID uuid.UUID
		reasonCode string
	}
	intentGroups := make(map[intentKey][]struct {
		intentID  uuid.UUID
		createdTS time.Time
	})

	for _, intent := range intents {
		// Only check active statuses
		if intent.Status != "NEW" && intent.Status != "PENDING_APPROVAL" && intent.Status != "SUBMITTED" {
			continue
		}

		key := intentKey{positionID: intent.PositionID, reasonCode: intent.ReasonCode}
		intentGroups[key] = append(intentGroups[key], struct {
			intentID  uuid.UUID
			createdTS time.Time
		}{intentID: intent.IntentID, createdTS: intent.CreatedTS})
	}

	// Cancel older duplicates (keep the most recent one)
	cancelledCount := 0
	for key, intentList := range intentGroups {
		if len(intentList) <= 1 {
			continue
		}

		// Cancel all except the most recent (last one in the list)
		for i := 0; i < len(intentList)-1; i++ {
			err := s.intentRepo.UpdateIntentStatus(ctx, intentList[i].intentID, "CANCELLED")
			if err != nil {
				log.Warn().Err(err).Str("intent_id", intentList[i].intentID.String()).Msg("Failed to cancel duplicate intent")
				continue
			}

			log.Warn().
				Str("position_id", key.positionID.String()).
				Str("reason_code", key.reasonCode).
				Str("cancelled_intent_id", intentList[i].intentID.String()).
				Msg("⚠️ Cancelled duplicate intent")
			cancelledCount++
		}
	}

	if cancelledCount > 0 {
		log.Info().Int("count", cancelledCount).Msg("Cancelled duplicate intents during reconciliation")
	}

	return nil
}
