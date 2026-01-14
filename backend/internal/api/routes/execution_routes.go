package routes

import (
	"github.com/gorilla/mux"
	executionHandlers "github.com/wonny/aegis/v14/internal/api/handlers/execution"
	"github.com/wonny/aegis/v14/internal/domain/execution"
)

// RegisterExecutionRoutes registers all execution-related routes
func RegisterExecutionRoutes(
	router *mux.Router,
	orderRepo execution.OrderRepository,
	fillRepo execution.FillRepository,
	holdingRepo execution.HoldingRepository,
	exitEventRepo execution.ExitEventRepository,
) {
	// Create handlers
	orderHandler := executionHandlers.NewOrderHandler(orderRepo)
	fillHandler := executionHandlers.NewFillHandler(fillRepo)
	holdingHandler := executionHandlers.NewHoldingHandler(holdingRepo)
	exitEventHandler := executionHandlers.NewExitEventHandler(exitEventRepo)

	// Order endpoints
	router.HandleFunc("/api/v1/execution/orders/{orderId}", orderHandler.GetOrder).Methods("GET")
	router.HandleFunc("/api/v1/execution/orders/open", orderHandler.ListOpenOrders).Methods("GET")

	// Fill endpoints
	router.HandleFunc("/api/v1/execution/orders/{orderId}/fills", fillHandler.GetFillsForOrder).Methods("GET")

	// Holding endpoints
	router.HandleFunc("/api/v1/execution/holdings", holdingHandler.ListHoldings).Methods("GET")

	// ExitEvent endpoints
	router.HandleFunc("/api/v1/execution/exit-events/{exitEventId}", exitEventHandler.GetExitEvent).Methods("GET")
	router.HandleFunc("/api/v1/execution/exit-events", exitEventHandler.ListExitEvents).Methods("GET")
}
