package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
	"aegis/internal/service/execution"
	"aegis/internal/service/exit"
	"aegis/internal/service/pricesync"
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
	orderIntentRepo := postgres.NewOrderIntentRepository(dbPool.Pool)
	orderRepo := postgres.NewOrderRepository(dbPool.Pool)
	fillRepo := postgres.NewFillRepository(dbPool.Pool)
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)

	executionService := execution.NewService(
		ctx,
		kisClient,
		orderIntentRepo,
		orderRepo,
		fillRepo,
		holdingRepo,
	)

	// Bootstrap execution service (sync holdings, orders, fills from KIS)
	if err := executionService.Bootstrap(ctx); err != nil {
		log.Error().Err(err).Msg("Execution Service bootstrap failed, continuing anyway")
	} else {
		log.Info().Msg("✅ Execution Service bootstrapped")
	}

	// Start execution service loops
	go func() {
		if err := executionService.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Execution Service failed")
		}
	}()

	log.Info().Msg("✅ Execution Service started")

	// ========================================
	// 3. Initialize Exit Engine
	// ========================================
	positionRepo := postgres.NewPositionRepository(dbPool.Pool)
	positionStateRepo := postgres.NewPositionStateRepository(dbPool.Pool)
	exitProfileRepo := postgres.NewExitProfileRepository(dbPool.Pool)
	exitControlRepo := postgres.NewExitControlRepository(dbPool.Pool)
	symbolOverrideRepo := postgres.NewSymbolExitOverrideRepository(dbPool.Pool)

	exitService := exit.NewService(
		ctx,
		priceService,
		orderIntentRepo,
		positionRepo,
		positionStateRepo,
		exitProfileRepo,
		exitControlRepo,
		symbolOverrideRepo,
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
