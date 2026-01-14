package handlers

import (
	"context"

	"github.com/wonny/aegis/v14/internal/domain/execution"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// HoldingReader is a read-only interface for holdings (monitoring)
type HoldingReader interface {
	GetAllHoldings(ctx context.Context) ([]*execution.Holding, error)
}

// OrderReader is a read-only interface for orders (monitoring)
type OrderReader interface {
	GetRecentOrders(ctx context.Context, limit int) ([]*execution.Order, error)
}

// FillReader is a read-only interface for fills (monitoring)
type FillReader interface {
	GetRecentFills(ctx context.Context, limit int) ([]*execution.Fill, error)
}

// IntentReader is a read-only interface for order intents (monitoring)
type IntentReader interface {
	GetRecentIntents(ctx context.Context, limit int) ([]*exit.OrderIntent, error)
}

// IntentWriter is a write interface for order intents (approval/rejection)
type IntentWriter interface {
	ApproveIntent(ctx context.Context, intentID string) error
	RejectIntent(ctx context.Context, intentID string) error
}
