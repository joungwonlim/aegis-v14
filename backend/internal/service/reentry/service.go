package reentry

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/reentry"
)

const (
	evaluationInterval = 5 * time.Second  // Evaluation loop 주기 (5초)
	exitEventCheckInterval = 3 * time.Second // ExitEvent polling 주기 (3초)
)

// Service is the Reentry Engine service
type Service struct {
	// Context
	ctx context.Context

	// Repositories
	candidateRepo reentry.CandidateRepository
	controlRepo   reentry.ControlRepository
	profileRepo   reentry.ProfileRepository

	// External dependencies (read-only)
	exitEventRepo execution.ExitEventRepository
	intentWriter  IntentWriter // For creating ENTRY intents

	// Config
	defaultProfile *reentry.ReentryProfile
}

// IntentWriter is an interface for creating order intents
type IntentWriter interface {
	// CreateEntryIntent creates a new ENTRY intent
	CreateEntryIntent(ctx context.Context, intent *EntryIntent) error
}

// EntryIntent represents an ENTRY order intent (to be created)
type EntryIntent struct {
	CandidateID  uuid.UUID
	Symbol       string
	Qty          int64
	OrderType    string // MKT, LMT
	LimitPrice   *decimal.Decimal
	ReasonCode   string // REENTRY_REBOUND, REENTRY_BREAKOUT, REENTRY_CHASE
}

// NewService creates a new Reentry service
func NewService(
	ctx context.Context,
	candidateRepo reentry.CandidateRepository,
	controlRepo reentry.ControlRepository,
	profileRepo reentry.ProfileRepository,
	exitEventRepo execution.ExitEventRepository,
	intentWriter IntentWriter,
) *Service {
	return &Service{
		ctx:           ctx,
		candidateRepo: candidateRepo,
		controlRepo:   controlRepo,
		profileRepo:   profileRepo,
		exitEventRepo: exitEventRepo,
		intentWriter:  intentWriter,
		defaultProfile: nil, // Will be loaded on Start()
	}
}

// Start starts the Reentry Engine
func (s *Service) Start() error {
	log.Info().Msg("Starting Reentry Engine")

	// Load default profile
	profile, err := s.profileRepo.GetDefaultProfile(s.ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load default profile, using built-in defaults")
		s.defaultProfile = s.getBuiltInDefaultProfile()
	} else {
		s.defaultProfile = profile
	}

	// Start background loops
	go s.exitEventMonitorLoop()
	go s.evaluationLoop()

	log.Info().Msg("Reentry Engine started")
	return nil
}

// exitEventMonitorLoop monitors for new ExitEvents
func (s *Service) exitEventMonitorLoop() {
	ticker := time.NewTicker(exitEventCheckInterval)
	defer ticker.Stop()

	lastCheck := time.Now().Add(-24 * time.Hour) // Start from 24h ago

	for {
		select {
		case <-ticker.C:
			// Load new exit events since last check
			exitEvents, err := s.exitEventRepo.LoadExitEventsSince(s.ctx, lastCheck)
			if err != nil {
				log.Error().Err(err).Msg("Failed to load exit events")
				continue
			}

			if len(exitEvents) > 0 {
				log.Debug().Int("count", len(exitEvents)).Msg("Processing new exit events")

				for _, event := range exitEvents {
					if err := s.handleExitEvent(s.ctx, event); err != nil {
						log.Error().
							Err(err).
							Str("exit_event_id", event.ExitEventID.String()).
							Msg("Failed to handle exit event")
					}
				}

				// Update last check time to the latest event
				lastCheck = exitEvents[0].CreatedTS
			}

		case <-s.ctx.Done():
			log.Info().Msg("ExitEvent monitor loop stopped")
			return
		}
	}
}

// evaluationLoop evaluates active candidates
func (s *Service) evaluationLoop() {
	ticker := time.NewTicker(evaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.evaluateAllCandidates(s.ctx); err != nil {
				log.Error().Err(err).Msg("Candidate evaluation failed")
			}

		case <-s.ctx.Done():
			log.Info().Msg("Evaluation loop stopped")
			return
		}
	}
}

// getBuiltInDefaultProfile returns a built-in default profile
func (s *Service) getBuiltInDefaultProfile() *reentry.ReentryProfile {
	return &reentry.ReentryProfile{
		ProfileID:   "default",
		Name:        "Default Reentry Profile",
		Description: "Built-in default configuration",
		Config: reentry.ReentryProfileConfig{
			CooldownSL:    300,  // 5 minutes
			CooldownTP:    600,  // 10 minutes
			CooldownTime:  1800, // 30 minutes
			MaxReentries:  3,
			MaxWatchHours: 24,
			TriggerRebound: reentry.ReboundConfig{
				Enabled:       true,
				BouncePercent: 0.02, // 2%
				MinVolume:     10000,
			},
			TriggerBreakout: reentry.BreakoutConfig{
				Enabled:       true,
				BreakPercent:  0.03, // 3%
				MinVolume:     10000,
			},
			TriggerChase: reentry.ChaseConfig{
				Enabled:       false,
				ChasePercent:  0.05, // 5%
				MinVolume:     10000,
			},
			SizingMode:    reentry.SizingModePercent,
			SizingPercent: 0.02, // 2% of portfolio
			SizingMax:     100,
		},
		IsActive:  true,
		CreatedBy: "system",
		CreatedTS: time.Now(),
	}
}
