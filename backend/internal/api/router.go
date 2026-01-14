package api

import (
	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/api/middleware"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
	"github.com/wonny/aegis/v14/internal/pkg/config"
	"github.com/wonny/aegis/v14/internal/pkg/logger"
	"github.com/wonny/aegis/v14/internal/service/pricesync"
)

// Router holds all dependencies for API routing
type Router struct {
	engine        *gin.Engine
	config        *config.Config
	dbPool        *postgres.Pool
	healthHandler *handlers.HealthHandler
	stockHandler  *handlers.StockHandler
	priceHandler  *handlers.PriceHandler
}

// NewRouter creates a new API router with all dependencies
func NewRouter(cfg *config.Config, dbPool *postgres.Pool, version string) *Router {
	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create Gin engine
	engine := gin.New()

	// Create repositories
	stockRepo := postgres.NewStockRepository(dbPool)
	priceRepo := postgres.NewPriceRepository(dbPool.Pool)

	// Create services
	priceService := pricesync.NewService(priceRepo)

	// Create handlers
	healthHandler := handlers.NewHealthHandler(dbPool, version)
	stockHandler := handlers.NewStockHandler(stockRepo)
	priceHandler := handlers.NewPriceHandler(priceService)

	router := &Router{
		engine:        engine,
		config:        cfg,
		dbPool:        dbPool,
		healthHandler: healthHandler,
		stockHandler:  stockHandler,
		priceHandler:  priceHandler,
	}

	// Setup middlewares and routes
	router.setupMiddlewares()
	router.setupRoutes()

	return router
}

// setupMiddlewares configures all global middlewares
func (r *Router) setupMiddlewares() {
	// Recovery middleware (must be first)
	r.engine.Use(middleware.Recovery())

	// Request ID middleware
	r.engine.Use(middleware.RequestID())

	// Logging middleware
	accessLogger := logger.NewAccessLogger(
		r.config.Logging.FilePath,
		r.config.Logging.RotationSize,
		r.config.Logging.RetentionDays,
	)
	r.engine.Use(middleware.Logging(middleware.LoggingConfig{
		AccessLogger: &accessLogger,
		SkipPaths:    []string{"/health", "/health/ready"}, // Skip health checks to reduce noise
	}))

	// CORS middleware
	if r.config.Server.Mode == "debug" {
		r.engine.Use(middleware.CORS(middleware.DevelopmentCORSConfig()))
	} else {
		r.engine.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	}
}

// setupRoutes configures all API routes
func (r *Router) setupRoutes() {
	// Health checks (no /api prefix)
	r.engine.GET("/health", r.healthHandler.Health)
	r.engine.GET("/health/ready", r.healthHandler.Ready)

	// API routes
	api := r.engine.Group("/api")
	{
		// Detailed health check
		api.GET("/health/detailed", r.healthHandler.Detailed)

		// Stocks API
		stocks := api.Group("/stocks")
		{
			stocks.GET("", r.stockHandler.List)
			stocks.GET("/:symbol", r.stockHandler.GetBySymbol)
		}

		// Prices API
		prices := api.Group("/prices")
		{
			prices.GET("/:symbol", r.priceHandler.GetBestPrice)
			prices.POST("/batch", r.priceHandler.BatchGetBestPrices)
			prices.GET("/:symbol/freshness", r.priceHandler.GetFreshness)
		}
	}
}

// Engine returns the underlying Gin engine
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
