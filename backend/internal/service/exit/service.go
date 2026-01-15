package exit

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// Service implements Exit Engine business logic
type Service struct {
	// Repositories
	posRepo       exit.PositionRepository
	stateRepo     exit.PositionStateRepository
	controlRepo   exit.ExitControlRepository
	intentRepo    exit.OrderIntentRepository
	profileRepo   exit.ExitProfileRepository
	overrideRepo  exit.SymbolExitOverrideRepository
	signalRepo    exit.ExitSignalRepository

	// Dependencies
	priceSync     *pricesync.Service

	// Default profile (loaded from config)
	defaultProfile *exit.ExitProfile

	// State
	mu        sync.RWMutex
	isRunning bool

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// NewService creates a new Exit service
func NewService(
	posRepo exit.PositionRepository,
	stateRepo exit.PositionStateRepository,
	controlRepo exit.ExitControlRepository,
	intentRepo exit.OrderIntentRepository,
	profileRepo exit.ExitProfileRepository,
	overrideRepo exit.SymbolExitOverrideRepository,
	signalRepo exit.ExitSignalRepository,
	priceSync *pricesync.Service,
	defaultProfile *exit.ExitProfile,
) *Service {
	return &Service{
		posRepo:        posRepo,
		stateRepo:      stateRepo,
		controlRepo:    controlRepo,
		intentRepo:     intentRepo,
		profileRepo:    profileRepo,
		overrideRepo:   overrideRepo,
		signalRepo:     signalRepo,
		priceSync:      priceSync,
		defaultProfile: defaultProfile,
		isRunning:      false,
	}
}

// Start starts the Exit evaluation loop
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		log.Warn().Msg("Exit Service already running")
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.isRunning = true

	log.Info().Msg("Starting Exit Service...")

	// Start evaluation loop
	go s.evaluationLoop()

	// Start reconciliation loop (30초마다)
	go s.reconciliationLoop()

	log.Info().Msg("✅ Exit Service started")

	return nil
}

// Stop stops the Exit evaluation loop
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	log.Info().Msg("Stopping Exit Service...")

	if s.cancel != nil {
		s.cancel()
	}

	s.isRunning = false

	log.Info().Msg("✅ Exit Service stopped")
}

// IsRunning returns whether Exit Service is running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// CreateManualIntent creates a manual exit intent
func (s *Service) CreateManualIntent(ctx context.Context, positionID uuid.UUID, qty int64, orderType string) error {
	// Get position
	pos, err := s.posRepo.GetPosition(ctx, positionID)
	if err != nil {
		return err
	}

	// Check exit mode
	if pos.ExitMode == exit.ExitModeDisabled {
		return exit.ErrExitDisabled
	}

	// Check available qty
	availableQty, err := s.posRepo.GetAvailableQty(ctx, positionID)
	if err != nil {
		return err
	}

	if availableQty <= 0 {
		return exit.ErrNoAvailableQty
	}

	// Clamp qty
	if qty > availableQty {
		qty = availableQty
	}

	// Create intent
	intent := &exit.OrderIntent{
		IntentID:   uuid.New(),
		PositionID: positionID,
		Symbol:     pos.Symbol,
		IntentType: exit.IntentTypeExitFull,
		Qty:        qty,
		OrderType:  orderType,
		ReasonCode: exit.ReasonManual,
		ActionKey:  positionID.String() + ":MANUAL",
		Status:     exit.IntentStatusNew,
	}

	return s.intentRepo.CreateIntent(ctx, intent)
}

// GetControl retrieves the current exit control mode
func (s *Service) GetControl(ctx context.Context) (*exit.ExitControl, error) {
	return s.controlRepo.GetControl(ctx)
}

// UpdateControl updates the exit control mode
func (s *Service) UpdateControl(ctx context.Context, mode string, reason *string, updatedBy string) error {
	return s.controlRepo.UpdateControl(ctx, mode, reason, updatedBy)
}

// GetPositionState retrieves the FSM state for a position
func (s *Service) GetPositionState(ctx context.Context, positionID uuid.UUID) (*exit.PositionState, error) {
	return s.stateRepo.GetState(ctx, positionID)
}

// GetActiveProfiles retrieves all active exit profiles
func (s *Service) GetActiveProfiles(ctx context.Context) ([]*exit.ExitProfile, error) {
	return s.profileRepo.GetActiveProfiles(ctx)
}

// CreateOrUpdateProfile creates or updates an exit profile
func (s *Service) CreateOrUpdateProfile(ctx context.Context, profile *exit.ExitProfile) error {
	return s.profileRepo.CreateOrUpdateProfile(ctx, profile)
}

// GetSymbolOverride retrieves symbol override
func (s *Service) GetSymbolOverride(ctx context.Context, symbol string) (*exit.SymbolExitOverride, error) {
	return s.overrideRepo.GetOverride(ctx, symbol)
}

// SetSymbolOverride sets or updates symbol override
func (s *Service) SetSymbolOverride(ctx context.Context, override *exit.SymbolExitOverride) error {
	return s.overrideRepo.SetOverride(ctx, override)
}

// DeleteSymbolOverride removes symbol override
func (s *Service) DeleteSymbolOverride(ctx context.Context, symbol string) error {
	return s.overrideRepo.DeleteOverride(ctx, symbol)
}
