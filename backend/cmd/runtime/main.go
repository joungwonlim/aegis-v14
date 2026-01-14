package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
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
		log.Fatal().Msg("KIS_ACCOUNT_ID environment variable is required")
	}

	// ========================================
	// 1. Initialize PriceSync Service
	// ========================================
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)
	priceService := pricesync.NewService(priceRepo)
	priceSyncManager := pricesync.NewManager(priceService, kisClient)

	if err := priceSyncManager.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start PriceSync Manager")
	}

	log.Info().Msg("✅ PriceSync Manager started")

	// ========================================
	// 2. Initialize Execution Service
	// ========================================
	orderRepo := postgres.NewOrderRepository(dbPool.Pool)
	fillRepo := postgres.NewFillRepository(dbPool.Pool)
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	exitEventRepo := postgres.NewExitEventRepository(dbPool.Pool)
	orderIntentRepo := postgres.NewOrderIntentRepository(dbPool.Pool)
	positionRepo := postgres.NewPositionRepository(dbPool.Pool)

	executionService := execution.NewService(
		ctx,
		orderRepo,
		fillRepo,
		holdingRepo,
		exitEventRepo,
		orderIntentRepo,
		positionRepo,
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
	positionStateRepo := postgres.NewPositionStateRepository(dbPool.Pool)
	exitProfileRepo := postgres.NewExitProfileRepository(dbPool.Pool)
	exitControlRepo := postgres.NewExitControlRepository(dbPool.Pool)
	symbolOverrideRepo := postgres.NewSymbolExitOverrideRepository(dbPool.Pool)

	// Create default exit profile
	defaultProfile := &exit.ExitProfile{
		ProfileID:   "default",
		Name:        "Default Exit Profile",
		Description: "Default exit rules for all positions",
		Config: exit.ExitProfileConfig{
			ATR: exit.ATRConfig{
				Enabled: false,
				Period:  14,
			},
			SL1: exit.TriggerConfig{
				Enabled:       true,
				ThresholdPct:  decimal.NewFromFloat(-3.0),
				ExitPct:       decimal.NewFromFloat(0.5),
				UseATRStop:    false,
				OrderType:     "MKT",
			},
			SL2: exit.TriggerConfig{
				Enabled:       true,
				ThresholdPct:  decimal.NewFromFloat(-5.0),
				ExitPct:       decimal.NewFromFloat(1.0),
				UseATRStop:    false,
				OrderType:     "MKT",
			},
			TP1: exit.TriggerConfig{
				Enabled:       true,
				ThresholdPct:  decimal.NewFromFloat(5.0),
				ExitPct:       decimal.NewFromFloat(0.3),
				UseATRStop:    false,
				OrderType:     "LMT",
			},
			TP2: exit.TriggerConfig{
				Enabled:       true,
				ThresholdPct:  decimal.NewFromFloat(10.0),
				ExitPct:       decimal.NewFromFloat(0.5),
				UseATRStop:    false,
				OrderType:     "LMT",
			},
			TP3: exit.TriggerConfig{
				Enabled:       false,
				ThresholdPct:  decimal.NewFromFloat(15.0),
				ExitPct:       decimal.NewFromFloat(0.2),
				UseATRStop:    false,
				OrderType:     "LMT",
			},
			Trailing: exit.TrailingConfig{
				Enabled:           true,
				ActivationPct:     decimal.NewFromFloat(3.0),
				TrailingPct:       decimal.NewFromFloat(1.5),
				UseATRTrail:       false,
				UsePartialExit:    true,
				PartialExitPct:    decimal.NewFromFloat(0.5),
				PartialActivation: decimal.NewFromFloat(5.0),
			},
			TimeStop: exit.TimeStopConfig{
				Enabled:        false,
				MaxHoldMinutes: 240,
				ExitPct:        decimal.NewFromFloat(1.0),
			},
			HardStop: exit.HardStopConfig{
				Enabled:      true,
				ThresholdPct: decimal.NewFromFloat(-7.0),
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
