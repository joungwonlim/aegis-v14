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
)

func main() {
	// Logger 초기화
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("Starting Aegis v14 API Server...")

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: Load configuration
	// TODO: Initialize database connection
	// TODO: Initialize Redis client
	// TODO: Initialize services
	// TODO: Initialize HTTP server

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	// TODO: Cleanup resources

	<-shutdownCtx.Done()
	log.Info().Msg("Server stopped")
}

func run(ctx context.Context) error {
	// Application logic will go here
	return fmt.Errorf("not implemented")
}
