package routes

import (
	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// RegisterRankingRoutes registers stock ranking routes
func RegisterRankingRoutes(router *mux.Router, dbPool *postgres.Pool) {
	handler := handlers.NewStockRankingsHandler(dbPool.Pool)

	// v1 API
	v1 := router.PathPrefix("/api/v1/rankings").Subrouter()

	v1.HandleFunc("/volume", handler.GetTopByVolume).Methods("GET")
	v1.HandleFunc("/trading-value", handler.GetTopByTradingValue).Methods("GET")
	v1.HandleFunc("/gainers", handler.GetTopGainers).Methods("GET")
	v1.HandleFunc("/losers", handler.GetTopLosers).Methods("GET")
	v1.HandleFunc("/foreign-net-buy", handler.GetTopForeignNetBuy).Methods("GET")
	v1.HandleFunc("/inst-net-buy", handler.GetTopInstNetBuy).Methods("GET")
}
