package exit

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PositionRepository manages positions (shared with Execution)
type PositionRepository interface {
	// GetPosition retrieves a position by ID with version for optimistic locking
	GetPosition(ctx context.Context, positionID uuid.UUID) (*Position, error)

	// GetOpenPositions retrieves all OPEN positions for an account
	GetOpenPositions(ctx context.Context, accountID string) ([]*Position, error)

	// GetAllOpenPositions retrieves all OPEN positions (across all accounts)
	GetAllOpenPositions(ctx context.Context) ([]*Position, error)

	// UpdateStatus updates position status (Exit Engine owns this column)
	// Uses optimistic locking with version check
	UpdateStatus(ctx context.Context, positionID uuid.UUID, status string, expectedVersion int) error

	// UpdateExitMode updates exit mode for a position (Exit Engine owns this column)
	UpdateExitMode(ctx context.Context, positionID uuid.UUID, mode string, profileID *string) error

	// UpdateExitModeBySymbol updates exit mode by account_id and symbol
	UpdateExitModeBySymbol(ctx context.Context, accountID string, symbol string, mode string) error

	// SyncQtyAndAvgPrice syncs position qty and avg_price from holdings (KIS source of truth)
	SyncQtyAndAvgPrice(ctx context.Context, accountID string, symbol string, qty int64, avgPrice decimal.Decimal) error

	// GetAvailableQty calculates available qty (position qty - locked qty from pending orders)
	GetAvailableQty(ctx context.Context, positionID uuid.UUID) (int64, error)
}

// PositionStateRepository manages Exit FSM state
type PositionStateRepository interface {
	// GetState retrieves position state (FSM)
	GetState(ctx context.Context, positionID uuid.UUID) (*PositionState, error)

	// UpsertState creates or updates position state
	UpsertState(ctx context.Context, state *PositionState) error

	// UpdatePhase updates FSM phase
	UpdatePhase(ctx context.Context, positionID uuid.UUID, phase string) error

	// UpdateHWM updates High-Water Mark
	UpdateHWM(ctx context.Context, positionID uuid.UUID, hwmPrice decimal.Decimal) error

	// UpdateStopFloor updates stop floor price
	UpdateStopFloor(ctx context.Context, positionID uuid.UUID, stopFloorPrice decimal.Decimal) error

	// UpdateATR updates cached ATR
	UpdateATR(ctx context.Context, positionID uuid.UUID, atr decimal.Decimal) error

	// IncrementBreachTicks increments breach tick counter (Phase 1: confirm_ticks)
	IncrementBreachTicks(ctx context.Context, positionID uuid.UUID) error

	// ResetBreachTicks resets breach tick counter to 0
	ResetBreachTicks(ctx context.Context, positionID uuid.UUID) error

	// ResetStateToOpen resets state to OPEN phase (for 평단가 변경 시)
	// Resets: Phase=OPEN, HWM=null, StopFloor=null, BreachTicks=0, LastAvgPrice=newAvgPrice
	ResetStateToOpen(ctx context.Context, positionID uuid.UUID, newAvgPrice decimal.Decimal) error
}

// ExitControlRepository manages global exit control (kill switch)
type ExitControlRepository interface {
	// GetControl retrieves current control mode (singleton, id=1)
	GetControl(ctx context.Context) (*ExitControl, error)

	// UpdateControl updates control mode
	UpdateControl(ctx context.Context, mode string, reason *string, updatedBy string) error
}

// ExitProfileRepository manages exit rule profiles
type ExitProfileRepository interface {
	// GetProfile retrieves a profile by ID
	GetProfile(ctx context.Context, profileID string) (*ExitProfile, error)

	// GetActiveProfiles retrieves all active profiles
	GetActiveProfiles(ctx context.Context) ([]*ExitProfile, error)

	// CreateOrUpdateProfile creates or updates a profile
	CreateOrUpdateProfile(ctx context.Context, profile *ExitProfile) error

	// DeleteProfile deactivates a profile
	DeleteProfile(ctx context.Context, profileID string) error
}

// SymbolExitOverrideRepository manages symbol-level overrides
type SymbolExitOverrideRepository interface {
	// GetOverride retrieves symbol override
	GetOverride(ctx context.Context, symbol string) (*SymbolExitOverride, error)

	// SetOverride creates or updates symbol override
	SetOverride(ctx context.Context, override *SymbolExitOverride) error

	// DeleteOverride removes symbol override
	DeleteOverride(ctx context.Context, symbol string) error
}

// OrderIntentRepository manages exit intents (idempotent)
type OrderIntentRepository interface {
	// CreateIntent creates a new intent (idempotent via action_key unique constraint)
	CreateIntent(ctx context.Context, intent *OrderIntent) error

	// GetIntent retrieves an intent by ID
	GetIntent(ctx context.Context, intentID uuid.UUID) (*OrderIntent, error)

	// GetIntentByActionKey retrieves an intent by action key (for idempotency check)
	GetIntentByActionKey(ctx context.Context, actionKey string) (*OrderIntent, error)

	// UpdateIntentStatus updates intent status
	UpdateIntentStatus(ctx context.Context, intentID uuid.UUID, status string) error

	// GetRecentIntents retrieves recent intents (for monitoring)
	GetRecentIntents(ctx context.Context, limit int) ([]*OrderIntent, error)

	// ApproveIntent approves an intent (PENDING_APPROVAL → NEW)
	ApproveIntent(ctx context.Context, intentID uuid.UUID) error

	// RejectIntent rejects an intent (PENDING_APPROVAL → CANCELLED)
	RejectIntent(ctx context.Context, intentID uuid.UUID) error
}

// ExitSignalRepository manages exit trigger evaluation records (debugging/backtest)
type ExitSignalRepository interface {
	// InsertSignal inserts a signal record
	InsertSignal(ctx context.Context, signal *ExitSignal) error

	// GetSignals retrieves signals for a position
	GetSignals(ctx context.Context, positionID uuid.UUID, limit int) ([]*ExitSignal, error)
}
