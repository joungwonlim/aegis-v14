package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/wonny/aegis/v14/internal/api/handlers"
)

// Config holds router configuration
type Config struct {
	HoldingsHandler   *handlers.HoldingsHandler
	IntentsHandler    *handlers.IntentsHandler
	OrdersHandler     *handlers.OrdersHandler
	FillsHandler      *handlers.FillsHandler
	KISOrdersHandler  *handlers.KISOrdersHandler
}

// NewRouter creates a new HTTP router
func NewRouter(cfg *Config) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3099"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Holdings
		r.Get("/holdings", cfg.HoldingsHandler.GetHoldings)
		r.Put("/holdings/{account_id}/{symbol}/exit-mode", cfg.HoldingsHandler.UpdateExitMode)

		// Order Intents
		r.Get("/intents", cfg.IntentsHandler.GetIntents)
		r.Post("/intents/{intent_id}/approve", cfg.IntentsHandler.ApproveIntent)
		r.Post("/intents/{intent_id}/reject", cfg.IntentsHandler.RejectIntent)

		// Orders
		r.Get("/orders", cfg.OrdersHandler.GetOrders)

		// Fills
		r.Get("/fills", cfg.FillsHandler.GetFills)

		// KIS Orders (직접 KIS API에서 가져옴)
		r.Get("/kis/unfilled-orders", cfg.KISOrdersHandler.GetUnfilledOrders)
		r.Get("/kis/filled-orders", cfg.KISOrdersHandler.GetFilledOrders)
		r.Route("/kis/orders", func(r chi.Router) {
			r.Post("/", cfg.KISOrdersHandler.PlaceOrder)
			r.Delete("/{order_no}", cfg.KISOrdersHandler.CancelOrder)
		})
	})

	return r
}
