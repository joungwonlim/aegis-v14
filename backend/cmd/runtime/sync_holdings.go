package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// Holdings Sync Service
type HoldingsSyncService struct {
	kisAdapter     execution.KISAdapter
	holdingRepo    execution.HoldingRepository
	accountID      string
	syncInterval   time.Duration
	stopChan       chan struct{}
}

// NewHoldingsSyncService creates a new holdings sync service
func NewHoldingsSyncService(
	kisAdapter execution.KISAdapter,
	holdingRepo execution.HoldingRepository,
	accountID string,
	syncInterval time.Duration,
) *HoldingsSyncService {
	return &HoldingsSyncService{
		kisAdapter:   kisAdapter,
		holdingRepo:  holdingRepo,
		accountID:    accountID,
		syncInterval: syncInterval,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the sync loop
func (s *HoldingsSyncService) Start(ctx context.Context) {
	log.Info().
		Str("account_id", s.accountID).
		Dur("interval", s.syncInterval).
		Msg("Starting holdings sync service")

	// Initial sync
	if err := s.syncOnce(ctx); err != nil {
		log.Error().Err(err).Msg("Initial holdings sync failed")
	}

	// Start periodic sync
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.syncOnce(ctx); err != nil {
				log.Error().Err(err).Msg("Holdings sync failed")
			}
		case <-s.stopChan:
			log.Info().Msg("Holdings sync service stopped")
			return
		case <-ctx.Done():
			log.Info().Msg("Holdings sync service context cancelled")
			return
		}
	}
}

// Stop stops the sync loop
func (s *HoldingsSyncService) Stop() {
	close(s.stopChan)
}

// syncOnce performs a single sync
func (s *HoldingsSyncService) syncOnce(ctx context.Context) error {
	startTime := time.Now()

	// Fetch holdings from KIS
	kisHoldings, err := s.kisAdapter.GetHoldings(ctx, s.accountID)
	if err != nil {
		return err
	}

	log.Debug().
		Int("count", len(kisHoldings)).
		Msg("Fetched holdings from KIS")

	// Upsert holdings to database
	for _, kh := range kisHoldings {
		holding := &execution.Holding{
			AccountID:    kh.AccountID,
			Symbol:       kh.Symbol,
			Qty:          kh.Qty,
			AvgPrice:     kh.AvgPrice,
			CurrentPrice: kh.CurrentPrice,
			Pnl:          kh.Pnl,
			PnlPct:       kh.PnlPct,
			UpdatedTS:    time.Now(),
			Raw:          kh.Raw,
		}

		if err := s.holdingRepo.UpsertHolding(ctx, holding); err != nil {
			log.Error().
				Err(err).
				Str("symbol", kh.Symbol).
				Msg("Failed to upsert holding")
			continue
		}
	}

	elapsed := time.Since(startTime)
	log.Info().
		Int("count", len(kisHoldings)).
		Dur("elapsed", elapsed).
		Msg("✅ Holdings synced successfully")

	return nil
}
