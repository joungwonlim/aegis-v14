package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	exitpg "github.com/wonny/aegis/v14/internal/infra/database/postgres/exit"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
	"github.com/wonny/aegis/v14/internal/service/execution"
	exitservice "github.com/wonny/aegis/v14/internal/service/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

const (
	serviceName    = "aegis-v14-runtime"
	serviceVersion = "1.0.0"
)

func main() {
	// Set timezone to Asia/Seoul (KST)
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load timezone")
	}
	time.Local = loc

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:          cfg.Logging.Level,
		Format:         cfg.Logging.Format,
		FileEnabled:    cfg.Logging.FileEnabled,
		FilePath:       cfg.Logging.FilePath,
		RotationSize:   cfg.Logging.RotationSize,
		RetentionDays:  cfg.Logging.RetentionDays,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	log.Info().
		Str("version", serviceVersion).
		Msg("🚀 Starting Aegis v14 Runtime (Core Trading Engine)...")

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	dbPool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbPool.Close()

	log.Info().Msg("✅ Database connected")

	// Initialize KIS client
	kisClient, err := kis.NewClientFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize KIS client (required for trading)")
	}

	log.Info().Msg("✅ KIS client initialized")

	// Create KIS Execution Adapter
	kisAdapter := kis.NewExecutionAdapter(kisClient)

	// Get account ID from environment
	accountID := os.Getenv("KIS_ACCOUNT_ID")
	if accountID == "" {
		accountID = os.Getenv("KIS_ACCOUNT_NO")
	}
	if accountID == "" {
		log.Fatal().Msg("KIS_ACCOUNT_ID or KIS_ACCOUNT_NO environment variable is required")
	}

	// ========================================
	// 1. Initialize PriceSync Service (V2 with DB protection)
	// ========================================
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)

	// Legacy service for Exit Engine (uses *price.BestPrice)
	priceService := pricesync.NewService(priceRepo)

	// ServiceV2 with DB protection (Coalescing + Cache + Broker) for REST/WS polling
	priceServiceV2 := pricesync.NewServiceV2(priceRepo, pricesync.DefaultServiceV2Config())

	// Note: PriorityManager will be configured later after Position/Order repositories are ready
	// Use V2 manager for optimized DB writes (coalescing/caching)
	priceSyncManager := pricesync.NewManagerV2(priceServiceV2, kisClient, nil)

	if err := priceSyncManager.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start PriceSync Manager")
	}

	log.Info().Msg("✅ PriceSync Manager started (V2 with DB protection)")

	// ========================================
	// 1.1. Subscribe to KIS Execution Notifications
	// ========================================
	// When execution notification is received, trigger immediate price-sync
	kisClient.WS.SetExecutionHandler(func(exec kis.ExecutionNotification) {
		log.Info().
			Str("symbol", exec.Symbol).
			Str("order_no", exec.OrderNo).
			Str("side", exec.Side).
			Int64("filled_qty", exec.FilledQty).
			Int64("filled_price", exec.FilledPrice).
			Msg("📣 Execution notification received - triggering price sync")

		// Trigger immediate price sync for this symbol
		priceSyncManager.TriggerRefresh(exec.Symbol)
	})

	// Subscribe to execution notifications for the account
	if err := kisClient.WS.SubscribeExecution(accountID); err != nil {
		log.Warn().Err(err).Msg("Failed to subscribe to execution notifications - will use polling instead")
	} else {
		log.Info().Str("account_id", accountID).Msg("✅ Subscribed to KIS execution notifications")
	}

	// ========================================
	// 1.5. Initialize Holdings Sync Service
	// ========================================
	holdingsSync := NewHoldingsSyncService(
		kisAdapter,
		postgres.NewHoldingRepository(dbPool.Pool),
		accountID,
		30*time.Second, // Sync every 30 seconds
	)

	go holdingsSync.Start(ctx)
	log.Info().Msg("✅ Holdings Sync Service started (interval: 30s)")

	// ========================================
	// 2. Initialize Execution Service
	// ========================================
	orderRepo := postgres.NewOrderRepository(dbPool.Pool)
	fillRepo := postgres.NewFillRepository(dbPool.Pool)
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	exitEventRepo := postgres.NewExitEventRepository(dbPool.Pool)
	orderIntentRepo := exitpg.NewOrderIntentRepository(dbPool.Pool)
	positionRepo := exitpg.NewPositionRepository(dbPool.Pool)

	executionService := execution.NewService(
		ctx,
		orderRepo,
		fillRepo,
		holdingRepo,
		exitEventRepo,
		orderIntentRepo,
		positionRepo,
		positionRepo, // exitPositionRepo - for auto-creating positions from holdings
		kisAdapter,
		accountID,
	)

	// Bootstrap execution service (sync holdings, orders, fills from KIS)
	if err := executionService.Bootstrap(ctx); err != nil {
		log.Error().Err(err).Msg("Execution Service bootstrap failed, continuing anyway")
	} else {
		log.Info().Msg("✅ Execution Service bootstrapped")
	}

	// Start execution service loops
	go func() {
		if err := executionService.Start(); err != nil {
			log.Error().Err(err).Msg("Execution Service failed")
		}
	}()

	log.Info().Msg("✅ Execution Service started")

	// ========================================
	// 3. Initialize Exit Engine
	// ========================================
	positionStateRepo := exitpg.NewPositionStateRepository(dbPool.Pool)
	exitProfileRepo := exitpg.NewExitProfileRepository(dbPool.Pool)
	exitControlRepo := exitpg.NewExitControlRepository(dbPool.Pool)
	symbolOverrideRepo := exitpg.NewSymbolExitOverrideRepository(dbPool.Pool)
	exitSignalRepo := exitpg.NewExitSignalRepository(dbPool.Pool)

	// Create default exit profile (v14 고정 비율)
	stopFloorProfit := 0.6 // 본전+0.6%
	defaultProfile := &exit.ExitProfile{
		ProfileID:   "default",
		Name:        "Default Exit Profile (v14)",
		Description: "v14 exit rules: TP1(10% @ +7%), TP2(20% @ +10%), TP3(30% @ +15%), Stop Floor +0.6%",
		Config: exit.ExitProfileConfig{
			ATR: exit.ATRConfig{
				Ref:       0.02,
				FactorMin: 1.0, // ATR 스케일링 비활성화 (고정 비율 사용)
				FactorMax: 1.0,
			},
			SL1: exit.TriggerConfig{
				BasePct: -0.03,  // -3%
				MinPct:  -0.03,  // 고정
				MaxPct:  -0.03,
				QtyPct:  0.5,    // 잔량의 50%
			},
			SL2: exit.TriggerConfig{
				BasePct: -0.05,  // -5%
				MinPct:  -0.05,  // 고정
				MaxPct:  -0.05,
				QtyPct:  1.0,    // 잔량의 100%
			},
			TP1: exit.TriggerConfig{
				BasePct:         0.07,   // +7%
				MinPct:          0.07,   // 고정
				MaxPct:          0.07,
				QtyPct:          0.10,   // 원본의 10%
				StopFloorProfit: &stopFloorProfit, // Stop Floor 활성화
			},
			TP2: exit.TriggerConfig{
				BasePct: 0.10,   // +10%
				MinPct:  0.10,   // 고정
				MaxPct:  0.10,
				QtyPct:  0.20,   // 원본의 20%
			},
			TP3: exit.TriggerConfig{
				BasePct:       0.15,   // +15%
				MinPct:        0.15,   // 고정
				MaxPct:        0.15,
				QtyPct:        0.30,   // 원본의 30%
				StartTrailing: true,   // 트레일링 활성화
			},
			Trailing: exit.TrailingConfig{
				PctTrail: 0.03,  // HWM 대비 -3%
				ATRK:     2.0,
			},
			TimeStop: exit.TimeStopConfig{
				MaxHoldDays:      0, // 비활성화 (0 = disabled)
				NoMomentumDays:   0,
				NoMomentumProfit: 0,
			},
			HardStop: exit.HardStopConfig{
				Enabled: true,
				Pct:     -0.10,
			},
		},
		IsActive:  true,
		CreatedBy: "system",
		CreatedTS: time.Now(),
	}

	exitService := exitservice.NewService(
		positionRepo,
		positionStateRepo,
		exitControlRepo,
		orderIntentRepo,
		exitProfileRepo,
		symbolOverrideRepo,
		exitSignalRepo,
		priceService,
		defaultProfile,
	)

	// Start exit engine loop
	go func() {
		if err := exitService.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Exit Engine failed")
		}
	}()

	log.Info().Msg("✅ Exit Engine started")

	// ========================================
	// 4. Initialize PriorityManager and Subscriptions
	// ========================================
	// Now that all repositories are ready, create PriorityManager
	positionAdapter := NewPositionRepoAdapter(positionRepo, holdingRepo, accountID)
	orderAdapter := NewOrderRepoAdapter(kisAdapter, accountID)
	watchlistAdapter := NewWatchlistRepoAdapter(dbPool.Pool)
	systemAdapter := NewSystemRepoAdapter()
	rankingAdapter := NewRankingRepoAdapter(dbPool.Pool)

	priorityManager := pricesync.NewPriorityManager(
		positionAdapter,
		orderAdapter,
		watchlistAdapter,
		systemAdapter,
		pricesync.WithRankingRepo(rankingAdapter),
	)

	// Set PriorityManager to existing running Manager
	priceSyncManager.SetPriorityManager(priorityManager)

	// Initialize subscriptions based on current positions/watchlist
	if err := priceSyncManager.InitializeSubscriptions(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to initialize PriceSync subscriptions, will retry periodically")
	} else {
		log.Info().Msg("✅ PriceSync subscriptions initialized from positions/watchlist")
	}

	// ========================================
	// All services running
	// ========================================
	log.Info().Msg("🎯 All Core Runtime services are running")
	log.Info().Msg("📊 Monitoring:")
	log.Info().Msg("  - PriceSync: Syncing prices from KIS/Naver")
	log.Info().Msg("  - Exit Engine: Evaluating exit rules on holdings")
	log.Info().Msg("  - Execution Service: Processing order intents → KIS orders")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("🛑 Shutdown signal received, stopping services...")

	// Cancel context to stop all services
	cancel()

	// Give services time to clean up
	time.Sleep(2 * time.Second)

	log.Info().Msg("👋 Aegis v14 Runtime stopped")
}
