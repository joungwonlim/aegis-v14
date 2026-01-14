package execution

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// OrderRepository manages order persistence
type OrderRepository interface {
	// CreateOrder creates a new order
	CreateOrder(ctx context.Context, order *Order) error

	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*Order, error)

	// GetOrderByIntentID retrieves an order by intent ID
	GetOrderByIntentID(ctx context.Context, intentID uuid.UUID) (*Order, error)

	// UpsertOrder creates or updates an order (idempotent)
	UpsertOrder(ctx context.Context, order *Order) error

	// UpdateOrderStatus updates order status
	UpdateOrderStatus(ctx context.Context, orderID string, status string) error

	// UpdateFilledQty increments filled_qty and updates open_qty
	UpdateFilledQty(ctx context.Context, orderID string, filledQty int64) error

	// LoadOpenOrders loads all SUBMITTED/PARTIAL orders
	LoadOpenOrders(ctx context.Context) ([]*Order, error)

	// LoadOrdersByStatus loads orders by status
	LoadOrdersByStatus(ctx context.Context, statuses []string) ([]*Order, error)
}

// FillRepository manages fill persistence
type FillRepository interface {
	// UpsertFill creates or updates a fill (idempotent by fill_id)
	UpsertFill(ctx context.Context, fill *Fill) error

	// LoadFills loads all fills for an order
	LoadFills(ctx context.Context, orderID string) ([]*Fill, error)

	// LoadFillsForPosition loads all fills for a position (via orders.intent_id → order_intents.position_id)
	LoadFillsForPosition(ctx context.Context, positionID uuid.UUID, intentType string) ([]*Fill, error)

	// LoadFillsSinceCursor loads fills since cursor (for sync)
	LoadFillsSinceCursor(ctx context.Context, cursor FillCursor) ([]*Fill, error)

	// GetLastCursor retrieves the last sync cursor
	GetLastCursor(ctx context.Context) (*FillCursor, error)

	// SaveCursor saves the sync cursor
	SaveCursor(ctx context.Context, cursor FillCursor) error
}

// HoldingRepository manages holding persistence
type HoldingRepository interface {
	// UpsertHolding creates or updates a holding (idempotent by account_id, symbol)
	UpsertHolding(ctx context.Context, holding *Holding) error

	// GetHolding retrieves a holding by account and symbol
	GetHolding(ctx context.Context, accountID, symbol string) (*Holding, error)

	// LoadHoldings loads all holdings for an account
	LoadHoldings(ctx context.Context, accountID string) ([]*Holding, error)

	// DeleteHolding deletes a holding (qty=0 cleanup)
	DeleteHolding(ctx context.Context, accountID, symbol string) error
}

// ExitEventRepository manages exit event persistence
type ExitEventRepository interface {
	// CreateExitEvent creates a new exit event
	CreateExitEvent(ctx context.Context, event *ExitEvent) error

	// GetExitEvent retrieves an exit event by ID
	GetExitEvent(ctx context.Context, exitEventID uuid.UUID) (*ExitEvent, error)

	// GetExitEventByPosition retrieves an exit event by position ID
	GetExitEventByPosition(ctx context.Context, positionID uuid.UUID) (*ExitEvent, error)

	// ExitEventExists checks if an exit event exists for a position
	ExitEventExists(ctx context.Context, positionID uuid.UUID) (bool, error)

	// LoadExitEventsSince loads exit events since timestamp
	LoadExitEventsSince(ctx context.Context, since time.Time) ([]*ExitEvent, error)
}

// IntentReader is a read-only interface for order_intents (owned by Strategy)
type IntentReader interface {
	// LoadNewIntents loads all intents with status=NEW
	LoadNewIntents(ctx context.Context) ([]*exit.OrderIntent, error)

	// GetIntent retrieves an intent by ID
	GetIntent(ctx context.Context, intentID uuid.UUID) (*exit.OrderIntent, error)

	// LoadIntentsForPosition loads all intents for a position (for exit reason determination)
	LoadIntentsForPosition(ctx context.Context, positionID uuid.UUID, intentTypes []string, statuses []string, since time.Time) ([]*exit.OrderIntent, error)

	// UpdateIntentStatus updates intent status (ONLY: NEW → SUBMITTED/FAILED/REJECTED/DUPLICATE)
	UpdateIntentStatus(ctx context.Context, intentID uuid.UUID, status string) error
}

// Intent Status (Execution-specific additions)
const (
	IntentStatusSubmitted = "SUBMITTED" // 제출됨
	IntentStatusFailed    = "FAILED"    // 제출 실패
	IntentStatusDuplicate = "DUPLICATE" // 중복 (이미 제출됨)
)

// Intent Type for ENTRY (Reentry Engine)
const (
	IntentTypeEntry = "ENTRY" // 진입 (재진입)
)

// PositionReader is a read-only interface for positions (owned by Strategy)
type PositionReader interface {
	// GetPositionBySymbol retrieves a position by symbol and status
	GetPositionBySymbol(ctx context.Context, accountID, symbol, status string) (*exit.Position, error)

	// GetPosition retrieves a position by ID
	GetPosition(ctx context.Context, positionID uuid.UUID) (*exit.Position, error)
}
