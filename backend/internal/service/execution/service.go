package execution

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

const (
	intentMonitorInterval = 2 * time.Second  // Intent monitor 주기 (1~3초 권장)
	reconcileInterval     = 15 * time.Second // Reconciliation 주기 (10~30초 권장)
	holdingsSyncInterval  = 20 * time.Second // Holdings sync 주기 (10~30초 권장)
	fillsSyncInterval     = 3 * time.Second  // Fills sync 주기 (3~5초 권장)
)

// Service is the Execution Engine service
type Service struct {
	// Context
	ctx context.Context

	// Repositories
	orderRepo     execution.OrderRepository
	fillRepo      execution.FillRepository
	holdingRepo   execution.HoldingRepository
	exitEventRepo execution.ExitEventRepository
	intentRepo    execution.IntentReader
	positionRepo  execution.PositionReader

	// External adapters
	kisAdapter execution.KISAdapter

	// Config
	accountID string

	// State
	prevHoldings []*execution.Holding // Previous holdings snapshot for ExitEvent detection
}

// NewService creates a new Execution service
func NewService(
	ctx context.Context,
	orderRepo execution.OrderRepository,
	fillRepo execution.FillRepository,
	holdingRepo execution.HoldingRepository,
	exitEventRepo execution.ExitEventRepository,
	intentRepo execution.IntentReader,
	positionRepo execution.PositionReader,
	kisAdapter execution.KISAdapter,
	accountID string,
) *Service {
	return &Service{
		ctx:           ctx,
		orderRepo:     orderRepo,
		fillRepo:      fillRepo,
		holdingRepo:   holdingRepo,
		exitEventRepo: exitEventRepo,
		intentRepo:    intentRepo,
		positionRepo:  positionRepo,
		kisAdapter:    kisAdapter,
		accountID:     accountID,
		prevHoldings:  []*execution.Holding{},
	}
}

// Start starts the Execution Engine
func (s *Service) Start() error {
	log.Info().Msg("Starting Execution Engine")

	// Bootstrap from KIS
	if err := s.Bootstrap(s.ctx); err != nil {
		log.Error().Err(err).Msg("Bootstrap failed")
		return err
	}

	// Start background loops
	go s.intentMonitorLoop()
	go s.reconcileLoop()
	go s.holdingsSyncLoop()
	go s.fillsSyncLoop()

	log.Info().Msg("Execution Engine started")
	return nil
}

// intentMonitorLoop monitors order_intents for NEW status
func (s *Service) intentMonitorLoop() {
	ticker := time.NewTicker(intentMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.processNewIntents(s.ctx); err != nil {
				log.Error().Err(err).Msg("Intent monitor failed")
			}

		case <-s.ctx.Done():
			log.Info().Msg("Intent monitor loop stopped")
			return
		}
	}
}

// reconcileLoop reconciles order states with KIS
func (s *Service) reconcileLoop() {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.reconcileOrders(s.ctx); err != nil {
				log.Error().Err(err).Msg("Reconciliation failed")
			}

		case <-s.ctx.Done():
			log.Info().Msg("Reconcile loop stopped")
			return
		}
	}
}

// holdingsSyncLoop syncs holdings from KIS
func (s *Service) holdingsSyncLoop() {
	ticker := time.NewTicker(holdingsSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.syncHoldings(s.ctx); err != nil {
				log.Error().Err(err).Msg("Holdings sync failed")
			}

		case <-s.ctx.Done():
			log.Info().Msg("Holdings sync loop stopped")
			return
		}
	}
}

// fillsSyncLoop syncs fills from KIS
func (s *Service) fillsSyncLoop() {
	ticker := time.NewTicker(fillsSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.syncFills(s.ctx); err != nil {
				log.Error().Err(err).Msg("Fills sync failed")
			}

		case <-s.ctx.Done():
			log.Info().Msg("Fills sync loop stopped")
			return
		}
	}
}
