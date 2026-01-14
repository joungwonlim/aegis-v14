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
	"github.com/wonny/aegis/v14/internal/api"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
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
		Str("port", cfg.Server.Port).
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

	log.Info().Msg("✅ All dependencies initialized")

	// Initialize HTTP router
	router := api.NewRouter(cfg, dbPool, serviceVersion)

	// Create HTTP server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router.Engine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("address", addr).
			Msg("🌐 HTTP server listening")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("📥 Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Database connection will be closed by defer

	log.Info().Msg("✅ Server stopped gracefully")
}
