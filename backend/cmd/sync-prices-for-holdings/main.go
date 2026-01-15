package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/google/uuid"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/domain/price"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	exitpg "github.com/wonny/aegis/v14/internal/infra/database/postgres/exit"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	exitservice "github.com/wonny/aegis/v14/internal/service/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msg("🚀 Starting Holdings Price Sync...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Load configuration
	log.Info().Msg("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// 2. Connect to PostgreSQL
	log.Info().Msg("Connecting to PostgreSQL...")
	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer dbPool.Close()
	log.Info().Msg("✅ PostgreSQL connected")

	// 2. Initialize KIS Client
	log.Info().Msg("Initializing KIS client...")
	kisClient, err := kis.NewClientFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create KIS client")
	}
	log.Info().Msg("✅ KIS client initialized")

	// 3. Initialize PriceSync Service and Manager
	log.Info().Msg("Initializing PriceSync...")
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)
	priceSyncService := pricesync.NewService(priceRepo)
	// Note: PriorityManager is nil for this utility - subscriptions will be done manually
	priceSyncManager := pricesync.NewManager(priceSyncService, kisClient, nil)

	// 4. Start PriceSync Manager
	log.Info().Msg("Starting PriceSync Manager...")
	if err := priceSyncManager.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start PriceSync Manager")
	}
	defer priceSyncManager.Stop()
	log.Info().Msg("✅ PriceSync Manager started")

	// 5. Initialize and Start Exit Engine
	log.Info().Msg("Initializing Exit Engine...")
	posRepo := exitpg.NewPositionRepository(dbPool.Pool)
	stateRepo := exitpg.NewPositionStateRepository(dbPool.Pool)
	controlRepo := exitpg.NewExitControlRepository(dbPool.Pool)
	intentRepo := exitpg.NewOrderIntentRepository(dbPool.Pool)
	profileRepo := exitpg.NewExitProfileRepository(dbPool.Pool)
	overrideRepo := exitpg.NewSymbolExitOverrideRepository(dbPool.Pool)

	// Create default exit profile manually (repository schema mismatch workaround)
	// Phase 1: 기관식 안정화 (confirm_ticks + fire_once + phase 분기)
	// - Stop Floor: 2틱 연속 breach 시 발화 (노이즈 청산 방지)
	// - TP2 부분 트레일: 원본 20% 단발 (action_key 멱등)
	// - Trailing: 2틱 연속 breach 시 발화
	// - SL1/SL2: 즉시 발화 (손실 방어 우선)
	// Phase 2 예정: ATR 기반 트레일 (변동성 레짐 대응)
	stopFloorProfit := 0.006 // 0.6% (본전+0.6%)
	defaultProfile := &exit.ExitProfile{
		ProfileID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa").String(),
		Name:      "default",
		IsActive:  true,
		Config: exit.ExitProfileConfig{
			// Stop Loss: 즉시 발화 (confirm_ticks=0)
			SL1: exit.TriggerConfig{BasePct: -0.03, QtyPct: 0.5},  // -3%, 잔량의 50%
			SL2: exit.TriggerConfig{BasePct: -0.05, QtyPct: 1.0},  // -5%, 잔량의 100%

			// Take Profit: 원본 기준
			TP1: exit.TriggerConfig{
				BasePct:         0.07, // +7%
				QtyPct:          0.1,  // 원본의 10%
				StopFloorProfit: &stopFloorProfit, // Stop Floor 활성화 (본전+0.6%, confirm_ticks=2)
			},
			TP2: exit.TriggerConfig{
				BasePct: 0.10, // +10%
				QtyPct:  0.2,  // 원본의 20% (부분 트레일 활성화)
			},
			TP3: exit.TriggerConfig{
				BasePct:       0.15,  // +15%
				QtyPct:        0.3,   // 원본의 30%
				StartTrailing: true,  // 잔량 트레일링 활성화
			},

			// Trailing: 고정 3% (Phase 2에서 ATR 기반으로 전환 예정)
			// - TP2 이후: 원본 20% 부분청산 (fire_once=true, confirm_ticks=2)
			// - TP3 이후: 잔량 전량청산 (confirm_ticks=2)
			Trailing: exit.TrailingConfig{PctTrail: 0.03}, // HWM 대비 -3%
		},
	}
	log.Info().Msg("✅ Default exit profile loaded")

	exitService := exitservice.NewService(
		posRepo,
		stateRepo,
		controlRepo,
		intentRepo,
		profileRepo,
		overrideRepo,
		priceSyncService,
		defaultProfile,
	)

	log.Info().Msg("Starting Exit Engine...")
	if err := exitService.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start Exit Engine")
	}
	defer exitService.Stop()
	log.Info().Msg("✅ Exit Engine started")

	// 6. Load holdings and subscribe to price updates
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	accountID := os.Getenv("KIS_ACCOUNT_ID")
	if accountID == "" {
		accountID = os.Getenv("KIS_ACCOUNT_NO")
	}

	log.Info().Str("account_id", accountID).Msg("Loading holdings...")
	holdings, err := holdingRepo.GetHoldingsByAccount(ctx, accountID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load holdings")
	}
	log.Info().Int("count", len(holdings)).Msg("✅ Holdings loaded")

	// 6. Subscribe all holdings to price sync (Tier0 = highest priority)
	for _, holding := range holdings {
		if err := priceSyncManager.SubscribeSymbol(holding.Symbol, pricesync.Tier0); err != nil {
			log.Error().Err(err).Str("symbol", holding.Symbol).Msg("Failed to subscribe symbol")
		} else {
			log.Info().Str("symbol", holding.Symbol).Msg("✅ Subscribed to price updates")
		}
	}

	// 7. Start background goroutine to update holdings prices
	log.Info().Msg("Starting price update loop...")
	go func() {
		ticker := time.NewTicker(1 * time.Second) // Check every 1 second
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Update holdings with latest prices
				if err := updateHoldingsPrices(ctx, holdingRepo, priceRepo, accountID); err != nil {
					log.Error().Err(err).Msg("Failed to update holdings prices")
				}
			}
		}
	}()

	log.Info().Msg("✅ Holdings Price Sync running. Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Info().Msg("✅ Shutdown complete")
}

// updateHoldingsPrices updates current_price in holdings from best prices
func updateHoldingsPrices(ctx context.Context, holdingRepo *postgres.HoldingRepository, priceRepo *postgres.PriceRepository, accountID string) error {
	// 1. Load holdings
	holdings, err := holdingRepo.GetHoldingsByAccount(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get holdings: %w", err)
	}

	// 2. For each holding, get best price and update
	for _, holding := range holdings {
		bestPrice, err := priceRepo.GetBestPrice(ctx, holding.Symbol)
		if err != nil {
			if err == price.ErrBestPriceNotFound {
				// No price yet, skip
				continue
			}
			log.Error().Err(err).Str("symbol", holding.Symbol).Msg("Failed to get best price")
			continue
		}

		// Update current_price if different
		newPrice := decimal.NewFromInt(bestPrice.BestPrice)
		if !newPrice.Equal(holding.CurrentPrice) {
			// Recalculate PnL
			entryValue := holding.AvgPrice.Mul(decimal.NewFromInt(holding.Qty))
			exitValue := newPrice.Mul(decimal.NewFromInt(holding.Qty))
			pnl := exitValue.Sub(entryValue)

			pnlPct := 0.0
			if !entryValue.IsZero() {
				pnlPct, _ = pnl.Div(entryValue).Mul(decimal.NewFromInt(100)).Float64()
			}

			// Update holding
			holding.CurrentPrice = newPrice
			holding.Pnl = pnl
			holding.PnlPct = pnlPct
			holding.UpdatedTS = time.Now()

			if err := holdingRepo.UpsertHolding(ctx, holding); err != nil {
				log.Error().Err(err).Str("symbol", holding.Symbol).Msg("Failed to update holding")
				continue
			}

			log.Info().
				Str("symbol", holding.Symbol).
				Str("old_price", holding.CurrentPrice.StringFixed(0)).
				Str("new_price", newPrice.StringFixed(0)).
				Msg("✅ Updated holding price")
		}
	}

	return nil
}
