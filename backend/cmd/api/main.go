package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	audithandlers "github.com/wonny/aegis/v14/internal/api/handlers/audit"
	fetcherhandlers "github.com/wonny/aegis/v14/internal/api/handlers/fetcher"
	signalshandlers "github.com/wonny/aegis/v14/internal/api/handlers/signals"
	universehandlers "github.com/wonny/aegis/v14/internal/api/handlers/universe"
	"github.com/wonny/aegis/v14/internal/api/routes"
	"github.com/wonny/aegis/v14/internal/infra/external/deepseek"
	"github.com/wonny/aegis/v14/internal/infra/external/openai"
	ai_analysis_repo "github.com/wonny/aegis/v14/internal/infrastructure/postgres/ai_analysis"
	ai_analysis_service "github.com/wonny/aegis/v14/internal/service/ai_analysis"
	"github.com/wonny/aegis/v14/internal/domain/exit"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	fetcherrepo "github.com/wonny/aegis/v14/internal/infra/database/postgres/fetcher"
	signalsrepo "github.com/wonny/aegis/v14/internal/infra/database/postgres/signals"
	exitrepo "github.com/wonny/aegis/v14/internal/infra/database/postgres/exit"
	"github.com/wonny/aegis/v14/internal/infra/external/dart"
	"github.com/wonny/aegis/v14/internal/infra/external/naver"
	"github.com/wonny/aegis/v14/internal/infra/kis"
	universerepo "github.com/wonny/aegis/v14/internal/infrastructure/postgres/universe"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
	auditservice "github.com/wonny/aegis/v14/internal/service/audit"
	exitservice "github.com/wonny/aegis/v14/internal/service/exit"
	fetcherservice "github.com/wonny/aegis/v14/internal/service/fetcher"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
	universeservice "github.com/wonny/aegis/v14/internal/service/universe"
	signalsservice "github.com/wonny/aegis/v14/internal/strategy/signals"
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

	// Parse account number and product code (format: "12345678-01")
	accountNo := accountID
	accountProductCode := "01" // Default
	if idx := strings.Index(accountID, "-"); idx > 0 {
		accountNo = accountID[:idx]
		accountProductCode = accountID[idx+1:]
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
	apiRouter.HandleFunc("/kis/trade-profit-loss", kisOrdersHandler.GetTradeProfitLoss).Methods("GET")

	// Register Exit routes
	routes.RegisterExitRoutes(httpRouter, exitSvc)

	// Register Watchlist routes
	routes.RegisterWatchlistRoutes(httpRouter, dbPool)

	// Register Stocks routes
	routes.RegisterStocksRoutes(httpRouter, dbPool)

	// Register Chart routes (simple price/flow history for charts)
	routes.RegisterChartRoutes(httpRouter, dbPool)

	// Register Ranking routes
	routes.RegisterRankingRoutes(httpRouter, dbPool)

	// Initialize AI Analysis
	openaiAPIKey := os.Getenv("CHATGPT_API_KEY")
	deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY")
	deepseekBaseURL := os.Getenv("DEEPSEEK_BASE_URL")

	if openaiAPIKey != "" || deepseekAPIKey != "" {
		openaiClient := openai.NewClient(openaiAPIKey)
		deepseekClient := deepseek.NewClient(deepseekAPIKey, deepseekBaseURL)
		aiRepo := ai_analysis_repo.NewRepository(dbPool.Pool)
		aiSvc := ai_analysis_service.NewService(aiRepo, openaiClient, deepseekClient)
		aiHandler := handlers.NewAIAnalysisHandler(aiSvc)

		// AI Analysis routes
		apiRouter.HandleFunc("/v1/stock/{symbol}/ai-analysis", aiHandler.AnalyzeStock).Methods("POST")
		apiRouter.HandleFunc("/v1/ai-analyses/recent", aiHandler.GetRecentAnalyses).Methods("GET")

		log.Info().Msg("✅ AI Analysis initialized")
	} else {
		log.Warn().Msg("⚠️ AI API keys not set, AI analysis will be disabled")
	}

	// Initialize Fetcher Service
	// 1. External Clients
	naverClient := naver.NewClient()
	dartAPIKey := os.Getenv("DART_API_KEY")
	var dartClient *dart.Client
	if dartAPIKey != "" {
		dartClient = dart.NewClient(dartAPIKey)
		log.Info().Msg("✅ DART client initialized")
	} else {
		log.Warn().Msg("⚠️ DART_API_KEY not set, disclosure collection will be disabled")
	}

	// 2. Fetcher Repositories
	fetcherStockRepo := fetcherrepo.NewStockRepository(dbPool)
	fetcherPriceRepo := fetcherrepo.NewPriceRepository(dbPool)
	fetcherFlowRepo := fetcherrepo.NewFlowRepository(dbPool)
	fetcherFundamentalRepo := fetcherrepo.NewFundamentalsRepository(dbPool)
	fetcherMarketCapRepo := fetcherrepo.NewMarketCapRepository(dbPool)
	fetcherDisclosureRepo := fetcherrepo.NewDisclosureRepository(dbPool)
	fetcherLogRepo := fetcherrepo.NewFetchLogRepository(dbPool.Pool)
	rankingRepo := postgres.NewRankingRepository(dbPool.Pool)

	// 3. Fetcher Service Configuration
	fetcherConfig := &fetcherservice.Config{
		PriceInterval:       1 * time.Hour,
		FlowInterval:        1 * time.Hour,
		FundamentalInterval: 24 * time.Hour,
		MarketCapInterval:   24 * time.Hour * 365, // 임시 비활성화 (파싱 로직 수정 필요)
		DisclosureInterval:  30 * time.Minute,
		BatchSize:           100,
		MaxRetries:          3,
		RetryBackoff:        5 * time.Second,
		MaxConcurrent:       5,
	}

	// 4. Create Fetcher Service
	fetcherSvc := fetcherservice.NewService(
		ctx,
		fetcherConfig,
		dbPool.Pool,
		naverClient,
		dartClient,
		fetcherStockRepo,
		fetcherPriceRepo,
		fetcherFlowRepo,
		fetcherFundamentalRepo,
		fetcherMarketCapRepo,
		fetcherDisclosureRepo,
		fetcherLogRepo,
		rankingRepo,
	)

	// 5. Start Fetcher Service in background
	if err := fetcherSvc.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start Fetcher service")
	} else {
		log.Info().Msg("✅ Fetcher service started (background data collection running)")
	}

	// Register Fetcher routes
	fetcherStatusHandler := handlers.NewFetcherStatusHandler(dbPool.Pool)
	fetcherHandler := fetcherhandlers.NewHandler(fetcherSvc)
	routes.RegisterFetcherRoutes(httpRouter, fetcherHandler, fetcherStatusHandler)

	// Initialize Universe Service
	// 1. Universe Repository
	universeRepo := universerepo.NewUniverseRepository(dbPool.Pool)

	// 2. Readers for Universe
	holdingReader := universerepo.NewHoldingReader(dbPool.Pool)
	watchlistReader := universerepo.NewWatchlistReader(dbPool.Pool)
	rankingReader := universerepo.NewRankingReader(dbPool.Pool)

	// 3. Stock Repository and Statistics Reader
	universeStockRepo := universerepo.NewStockRepository(dbPool.Pool)
	universeStatsReader := universerepo.NewStatisticsReader(dbPool.Pool)

	// 4. Create Universe Service
	universeSvc := universeservice.NewService(
		ctx,
		universeRepo, // Already created above for Signals
		universeStockRepo,
		universeStatsReader,
		holdingReader,
		watchlistReader,
		rankingReader,
	)

	// 5. Start Universe Service (will auto-generate initial snapshot)
	if err := universeSvc.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start Universe service")
	} else {
		log.Info().Msg("✅ Universe service started")
	}

	// 6. Create Universe Handler and Register Routes
	universeHandler := universehandlers.NewUniverseHandler(universeSvc)
	routes.RegisterUniverseRoutes(httpRouter, universeHandler)

	// Initialize Stock Info Handler (for company overview)
	stockInfoRepo := universerepo.NewStockInfoRepository(dbPool.Pool)
	stockInfoHandler := handlers.NewStockInfoHandler(stockInfoRepo, naverClient)

	// Stock Info routes
	apiRouter.HandleFunc("/stocks/{symbol}/info", stockInfoHandler.GetCompanyOverview).Methods("GET")
	apiRouter.HandleFunc("/stocks/{symbol}/info/refresh", stockInfoHandler.RefreshCompanyOverview).Methods("POST")

	// Initialize Audit Service
	auditRepo := postgres.NewAuditRepository(dbPool.Pool)
	auditSvc := auditservice.NewService(auditRepo)
	auditHandler := audithandlers.NewHandler(auditSvc)

	// Initialize KIS Audit Builder
	kisAuditBuilder := auditservice.NewKISAuditBuilder(auditSvc, kisClient, accountNo, accountProductCode)
	auditHandler.SetKISBuilder(kisAuditBuilder)

	routes.RegisterAuditRoutes(httpRouter, auditHandler)

	// Initialize Signals Service
	// 1. Signals Repositories
	signalsSignalRepo := signalsrepo.NewSignalRepository(dbPool.Pool)
	signalsFactorRepo := signalsrepo.NewFactorRepository(dbPool.Pool)

	// 2. Create Signals Service (using universeRepo from Universe Service above)
	signalsSvc := signalsservice.NewService(
		ctx,
		signalsSignalRepo,
		signalsFactorRepo,
		universeRepo,
	)

	// 4. Start Signals Service
	if err := signalsSvc.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start Signals service")
	} else {
		log.Info().Msg("✅ Signals service started")
	}

	// 5. Create Signals Handler and Register Routes
	signalsHandler := signalshandlers.NewHandler(signalsSvc, signalsFactorRepo)
	routes.RegisterSignalsRoutes(httpRouter, signalsHandler)

	log.Info().Msg("✅ All routes registered (Exit, Holdings, Intents, Orders, Fills, KIS, Watchlist, Stocks, Charts, Fetcher, Universe, Audit, Signals)")

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

	// Stop Fetcher Service
	if err := fetcherSvc.Stop(); err != nil {
		log.Error().Err(err).Msg("Fetcher service stop failed")
	} else {
		log.Info().Msg("✅ Fetcher service stopped")
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("👋 Aegis v14 API Server stopped")
}
