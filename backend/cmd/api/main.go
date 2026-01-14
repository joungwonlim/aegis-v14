package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/api/router"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	exitrepo "github.com/wonny/aegis/v14/internal/infra/database/postgres/exit"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
)

const (
	serviceName    = "aegis-v14-api"
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
		Msg("🚀 Starting Aegis v14 API Server...")

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

	// Initialize repositories
	holdingRepo := postgres.NewHoldingRepository(dbPool.Pool)
	positionRepo := exitrepo.NewPositionRepository(dbPool.Pool)
	orderIntentRepo := exitrepo.NewOrderIntentRepository(dbPool.Pool)
	orderRepo := postgres.NewOrderRepository(dbPool.Pool)
	fillRepo := postgres.NewFillRepository(dbPool.Pool)

	// Initialize handlers
	holdingsHandler := handlers.NewHoldingsHandler(holdingRepo, positionRepo)
	intentsHandler := handlers.NewIntentsHandler(orderIntentRepo, orderIntentRepo) // Reader and Writer
	ordersHandler := handlers.NewOrdersHandler(orderRepo)
	fillsHandler := handlers.NewFillsHandler(fillRepo)

	// Initialize router
	routerCfg := &router.Config{
		HoldingsHandler: holdingsHandler,
		IntentsHandler:  intentsHandler,
		OrdersHandler:   ordersHandler,
		FillsHandler:    fillsHandler,
	}

	httpRouter := router.NewRouter(routerCfg)

	// HTTP server port
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8099"
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      httpRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("address", addr).
			Msg("🎯 API Server listening")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start API server")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("🛑 Shutdown signal received, stopping server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("👋 Aegis v14 API Server stopped")
}
