package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/api/routes"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	exitrepo "github.com/wonny/aegis/v14/internal/infra/database/postgres/exit"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
	exitservice "github.com/wonny/aegis/v14/internal/service/exit"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

const (
	serviceName    = "aegis-v14-api"
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
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)

	// Exit Engine repositories
	stateRepo := exitrepo.NewPositionStateRepository(dbPool.Pool)
	controlRepo := exitrepo.NewExitControlRepository(dbPool.Pool)
	profileRepo := exitrepo.NewExitProfileRepository(dbPool.Pool)
	symbolOverrideRepo := exitrepo.NewSymbolExitOverrideRepository(dbPool.Pool)
	signalRepo := exitrepo.NewExitSignalRepository(dbPool.Pool)

	// Initialize KIS client and adapter
	kisClient, err := kis.NewClientFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create KIS client")
	}
	kisAdapter := kis.NewExecutionAdapter(kisClient)

	// Get account ID from environment
	accountID := os.Getenv("KIS_ACCOUNT_ID")
	if accountID == "" {
		accountID = os.Getenv("KIS_ACCOUNT_NO")
	}
	if accountID == "" {
		log.Fatal().Msg("KIS_ACCOUNT_ID or KIS_ACCOUNT_NO environment variable is required")
	}

	log.Info().Str("account_id", accountID).Msg("✅ KIS client initialized")

	// Initialize PriceSync Service (needed by Exit Service)
	priceService := pricesync.NewService(priceRepo)

	// Initialize Exit Service (without starting the evaluation loop)
	// Default profile (simple conservative profile for API)
	defaultProfile := &exit.ExitProfile{
		ProfileID:   "default",
		Name:        "Default Profile",
		Description: "Conservative exit strategy",
		Config:      exit.ExitProfileConfig{}, // Will be loaded from DB if needed
		IsActive:    true,
		CreatedBy:   "system",
	}
	exitSvc := exitservice.NewService(
		positionRepo,
		stateRepo,
		controlRepo,
		orderIntentRepo,
		profileRepo,
		symbolOverrideRepo,
		signalRepo,
		priceService,
		defaultProfile,
	)

	// Initialize handlers
	holdingsHandler := handlers.NewHoldingsHandler(holdingRepo, positionRepo, priceRepo)
	intentsHandler := handlers.NewIntentsHandler(orderIntentRepo, orderIntentRepo) // Reader and Writer
	ordersHandler := handlers.NewOrdersHandler(orderRepo)
	fillsHandler := handlers.NewFillsHandler(fillRepo)
	kisOrdersHandler := handlers.NewKISOrdersHandler(kisAdapter, accountID)

	// Create gorilla/mux router
	httpRouter := mux.NewRouter()

	// CORS configuration
	allowedOrigins := gorillaHandlers.AllowedOrigins([]string{"http://localhost:3099"})
	allowedMethods := gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := gorillaHandlers.AllowedHeaders([]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	allowCredentials := gorillaHandlers.AllowCredentials()

	// Register existing routes
	httpRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// API routes
	apiRouter := httpRouter.PathPrefix("/api").Subrouter()

	// Holdings
	apiRouter.HandleFunc("/holdings", holdingsHandler.GetHoldings).Methods("GET")
	apiRouter.HandleFunc("/holdings/{account_id}/{symbol}/exit-mode", holdingsHandler.UpdateExitMode).Methods("PUT")

	// Order Intents
	apiRouter.HandleFunc("/intents", intentsHandler.GetIntents).Methods("GET")
	apiRouter.HandleFunc("/intents/{intent_id}/approve", intentsHandler.ApproveIntent).Methods("POST")
	apiRouter.HandleFunc("/intents/{intent_id}/reject", intentsHandler.RejectIntent).Methods("POST")

	// Orders
	apiRouter.HandleFunc("/orders", ordersHandler.GetOrders).Methods("GET")

	// Fills
	apiRouter.HandleFunc("/fills", fillsHandler.GetFills).Methods("GET")

	// KIS Orders
	apiRouter.HandleFunc("/kis/unfilled-orders", kisOrdersHandler.GetUnfilledOrders).Methods("GET")
	apiRouter.HandleFunc("/kis/filled-orders", kisOrdersHandler.GetFilledOrders).Methods("GET")
	apiRouter.HandleFunc("/kis/orders", kisOrdersHandler.PlaceOrder).Methods("POST")
	apiRouter.HandleFunc("/kis/orders/{order_no}", kisOrdersHandler.CancelOrder).Methods("DELETE")

	// Register Exit routes
	routes.RegisterExitRoutes(httpRouter, exitSvc)

	// Register Watchlist routes
	routes.RegisterWatchlistRoutes(httpRouter, dbPool)

	// Register Stocks routes
	routes.RegisterStocksRoutes(httpRouter, dbPool)

	// Register Chart routes (simple price/flow history for charts)
	routes.RegisterChartRoutes(httpRouter, dbPool)

	log.Info().Msg("✅ All routes registered (Exit, Holdings, Intents, Orders, Fills, KIS, Watchlist, Stocks, Charts)")

	// Wrap with CORS
	handler := gorillaHandlers.CORS(allowedOrigins, allowedMethods, allowedHeaders, allowCredentials)(httpRouter)

	// HTTP server port
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8099"
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
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
