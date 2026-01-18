package execution

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

const (
	intentMonitorInterval = 2 * time.Second   // Intent monitor 주기 (1~3초 권장)
	reconcileInterval     = 120 * time.Second // Reconciliation 주기 (2분, rate limit 완화)
	holdingsSyncInterval  = 120 * time.Second // Holdings sync 주기 (2분, rate limit 완화)
	fillsSyncInterval     = 60 * time.Second  // Fills sync 주기 (1분, rate limit 완화)

	// ✅ 2026-01-18: Fills/Holdings sync 간격 증가
	// Portfolio 가격(Tier0)은 2.5초로 유지 (Exit Engine 우선)
	// Fills/Holdings는 실시간성보다 안정성 우선

	// Fills sync backoff (EGW00201 대응)
	fillsSyncBaseBackoff = 5 * time.Second
	fillsSyncMaxBackoff  = 120 * time.Second
	fillsSyncMaxRetries  = 5
)

// Service is the Execution Engine service
type Service struct {
	// Context
	ctx context.Context

	// Repositories
	orderRepo         execution.OrderRepository
	fillRepo          execution.FillRepository
	holdingRepo       execution.HoldingRepository
	exitEventRepo     execution.ExitEventRepository
	intentRepo        execution.IntentReader
	positionRepo      execution.PositionReader
	exitPositionRepo  exit.PositionRepository // For auto-creating positions

	// External adapters
	kisAdapter execution.KISAdapter

	// Optional hooks
	auditTradeWriter execution.AuditTradeWriter // For saving trades to audit (performance page)

	// Config
	accountID string

	// State
	prevHoldings []*execution.Holding // Previous holdings snapshot for ExitEvent detection

	// Fills sync backoff state (rate limit 대응)
	fillsSyncFailCount   int
	fillsSyncNextAllowed time.Time
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
	exitPositionRepo exit.PositionRepository,
	kisAdapter execution.KISAdapter,
	accountID string,
) *Service {
	return &Service{
		ctx:              ctx,
		orderRepo:        orderRepo,
		fillRepo:         fillRepo,
		holdingRepo:      holdingRepo,
		exitEventRepo:    exitEventRepo,
		intentRepo:       intentRepo,
		positionRepo:     positionRepo,
		exitPositionRepo: exitPositionRepo,
		kisAdapter:       kisAdapter,
		accountID:        accountID,
		prevHoldings:     []*execution.Holding{},
	}
}

// SetAuditTradeWriter sets the optional audit trade writer hook
func (s *Service) SetAuditTradeWriter(writer execution.AuditTradeWriter) {
	s.auditTradeWriter = writer
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

// fillsSyncLoop syncs fills from KIS with backoff on rate limit
func (s *Service) fillsSyncLoop() {
	ticker := time.NewTicker(fillsSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if we're in backoff period (rate limit)
			if time.Now().Before(s.fillsSyncNextAllowed) {
				remaining := time.Until(s.fillsSyncNextAllowed)
				log.Warn().
					Dur("remaining", remaining).
					Int("fail_count", s.fillsSyncFailCount).
					Msg("Fills sync skipped (in backoff)")
				continue
			}

			// Attempt sync
			if err := s.syncFills(s.ctx); err != nil {
				s.fillsSyncFailCount++

				// Check if EGW00201 (rate limit)
				isRateLimit := isEGW00201Error(err)

				// Calculate backoff duration (exponential with cap)
				backoff := fillsSyncBaseBackoff
				for i := 1; i < s.fillsSyncFailCount && i < fillsSyncMaxRetries; i++ {
					backoff *= 2
				}
				if backoff > fillsSyncMaxBackoff {
					backoff = fillsSyncMaxBackoff
				}

				// Set next allowed time
				s.fillsSyncNextAllowed = time.Now().Add(backoff)

				log.Error().
					Err(err).
					Bool("rate_limit", isRateLimit).
					Int("fail_count", s.fillsSyncFailCount).
					Dur("backoff", backoff).
					Time("next_allowed", s.fillsSyncNextAllowed).
					Msg("Fills sync failed")
			} else {
				// Success - reset backoff state
				if s.fillsSyncFailCount > 0 {
					log.Info().
						Int("prev_fail_count", s.fillsSyncFailCount).
						Msg("Fills sync recovered")
					s.fillsSyncFailCount = 0
					s.fillsSyncNextAllowed = time.Time{}
				}
			}

		case <-s.ctx.Done():
			log.Info().Msg("Fills sync loop stopped")
			return
		}
	}
}

// isEGW00201Error checks if error is EGW00201 (rate limit)
func isEGW00201Error(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "EGW00201") || contains(errStr, "초당 거래건수를 초과")
}

// contains checks if haystack contains needle (case-sensitive)
func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle || len(haystack) > len(needle) &&
		findSubstring(haystack, needle))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
