package routes

import (
	"github.com/gorilla/mux"
	"github.com/wonny/aegis/v14/internal/api/handlers"
	fetcherHandlers "github.com/wonny/aegis/v14/internal/api/handlers/fetcher"
)

// RegisterFetcherRoutes registers all fetcher-related routes
func RegisterFetcherRoutes(
	router *mux.Router,
	fetcherHandler *fetcherHandlers.Handler,
	statusHandler *handlers.FetcherStatusHandler,
) {
	// Status endpoints
	router.HandleFunc("/api/v1/fetcher/tables/stats", statusHandler.GetTableStats).Methods("GET")
	// Stock endpoints
	router.HandleFunc("/api/v1/fetcher/stocks", fetcherHandler.ListStocks).Methods("GET")
	router.HandleFunc("/api/v1/fetcher/stocks/{code}", fetcherHandler.GetStock).Methods("GET")
	router.HandleFunc("/api/v1/fetcher/stocks/{code}/data", fetcherHandler.GetStockData).Methods("GET")

	// Price endpoints
	router.HandleFunc("/api/v1/fetcher/prices/{code}", fetcherHandler.GetLatestPrice).Methods("GET")
	router.HandleFunc("/api/v1/fetcher/prices/{code}/history", fetcherHandler.GetPriceHistory).Methods("GET")

	// Flow endpoints
	router.HandleFunc("/api/v1/fetcher/flows/{code}", fetcherHandler.GetLatestFlow).Methods("GET")
	router.HandleFunc("/api/v1/fetcher/flows/{code}/history", fetcherHandler.GetFlowHistory).Methods("GET")

	// Disclosure endpoints
	router.HandleFunc("/api/v1/fetcher/disclosures", fetcherHandler.GetRecentDisclosures).Methods("GET")
	router.HandleFunc("/api/v1/fetcher/disclosures/{code}", fetcherHandler.GetStockDisclosures).Methods("GET")

	// Collection endpoints (admin)
	router.HandleFunc("/api/v1/fetcher/collect", fetcherHandler.TriggerCollection).Methods("POST")
	router.HandleFunc("/api/v1/fetcher/collect/{code}", fetcherHandler.CollectStock).Methods("POST")
	router.HandleFunc("/api/v1/fetcher/refresh-stocks", fetcherHandler.RefreshStockMaster).Methods("POST")
}
