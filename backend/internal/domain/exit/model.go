package exit

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ====================
// Position (trade.positions)
// ====================

// Position represents a trading position (long only for now)
type Position struct {
	PositionID    uuid.UUID       `json:"position_id"`
	AccountID     string          `json:"account_id"`
	Symbol        string          `json:"symbol"`
	Side          string          `json:"side"`      // LONG (short later)
	Qty           int64           `json:"qty"`       // Current quantity (Execution owns)
	OriginalQty   int64           `json:"original_qty"` // Original entry quantity (for TP % calculation)
	AvgPrice      decimal.Decimal `json:"avg_price"` // Average entry price (Execution owns)
	EntryTS       time.Time       `json:"entry_ts"`
	Status        string          `json:"status"`           // OPEN/CLOSING/CLOSED (Exit Engine owns)
	ExitMode      string          `json:"exit_mode"`        // ENABLED/DISABLED/MANUAL_ONLY (Exit Engine owns)
	ExitProfileID *string         `json:"exit_profile_id"`  // Override profile (Exit Engine owns), NULL = use default
	StrategyID    string          `json:"strategy_id"`      // Entry strategy
	UpdatedTS     time.Time       `json:"updated_ts"`
	Version       int             `json:"version"` // Optimistic locking
}

// Position Status
const (
	StatusOpen    = "OPEN"
	StatusClosing = "CLOSING"
	StatusClosed  = "CLOSED"
)

// Exit Mode
const (
	ExitModeEnabled    = "ENABLED"
	ExitModeDisabled   = "DISABLED"
	ExitModeManualOnly = "MANUAL_ONLY"
)

// ====================
// PositionState (trade.position_state)
// ====================

// PositionState represents Exit FSM state
type PositionState struct {
	PositionID     uuid.UUID        `json:"position_id"`
	Phase          string           `json:"phase"` // FSM phase
	HWMPrice       *decimal.Decimal `json:"hwm_price"`        // High-Water Mark
	StopFloorPrice *decimal.Decimal `json:"stop_floor_price"` // Stop Floor (breakeven protect)
	ATR            *decimal.Decimal `json:"atr"`              // ATR (cached, daily)
	CooldownUntil  *time.Time       `json:"cooldown_until"`   // Re-entry cooldown
	LastEvalTS     *time.Time       `json:"last_eval_ts"`
	LastAvgPrice   *decimal.Decimal `json:"last_avg_price"`   // 마지막 평단가 (추가매수 감지용)
	UpdatedTS      time.Time        `json:"updated_ts"`
	BreachTicks    int              `json:"breach_ticks"` // Phase 1: 연속 breach 카운터 (confirm_ticks)
}

// FSM Phases
const (
	PhaseOpen          = "OPEN"
	PhaseTP1Done       = "TP1_DONE"
	PhaseTP2Done       = "TP2_DONE"
	PhaseTP3Done       = "TP3_DONE"
	PhaseTrailingActive = "TRAILING_ACTIVE"
	PhaseExited        = "EXITED"
)

// ====================
// ExitControl (trade.exit_control)
// ====================

// ExitControl represents global exit control (kill switch)
type ExitControl struct {
	ID        int       `json:"id"` // Always 1 (singleton)
	Mode      string    `json:"mode"`
	Reason    *string   `json:"reason"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedTS time.Time `json:"updated_ts"`
}

// Control Modes
const (
	ControlModeRunning          = "RUNNING"
	ControlModePauseProfit      = "PAUSE_PROFIT"       // Block TP/TRAIL, allow SL
	ControlModePauseAll         = "PAUSE_ALL"          // Block all auto exits
	ControlModeEmergencyFlatten = "EMERGENCY_FLATTEN" // Force close all (optional)
)

// ====================
// ExitProfile (trade.exit_profiles)
// ====================

// ExitProfile represents exit rule profile
type ExitProfile struct {
	ProfileID   string              `json:"profile_id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Config      ExitProfileConfig   `json:"config"`
	IsActive    bool                `json:"is_active"`
	CreatedBy   string              `json:"created_by"`
	CreatedTS   time.Time           `json:"created_ts"`
}

// ExitProfileConfig represents exit rules configuration
type ExitProfileConfig struct {
	// ATR settings
	ATR ATRConfig `json:"atr"`

	// Triggers
	SL1 TriggerConfig `json:"sl1"` // Stop Loss 1 (partial)
	SL2 TriggerConfig `json:"sl2"` // Stop Loss 2 (full)
	TP1 TriggerConfig `json:"tp1"` // Take Profit 1
	TP2 TriggerConfig `json:"tp2"` // Take Profit 2
	TP3 TriggerConfig `json:"tp3"` // Take Profit 3

	// Trailing
	Trailing TrailingConfig `json:"trailing"`

	// Time stop
	TimeStop TimeStopConfig `json:"time_stop"`

	// HardStop
	HardStop HardStopConfig `json:"hardstop"`
}

type ATRConfig struct {
	Ref       float64 `json:"ref"`        // Reference ATR% (e.g., 0.02 = 2%)
	FactorMin float64 `json:"factor_min"` // Min factor (e.g., 0.7)
	FactorMax float64 `json:"factor_max"` // Max factor (e.g., 1.6)
}

type TriggerConfig struct {
	BasePct          float64  `json:"base_pct"`           // Base % (e.g., -0.03 = -3%)
	MinPct           float64  `json:"min_pct"`            // Min % after scaling
	MaxPct           float64  `json:"max_pct"`            // Max % after scaling
	QtyPct           float64  `json:"qty_pct"`            // Qty % to exit (e.g., 0.25 = 25%)
	StopFloorProfit  *float64 `json:"stop_floor_profit"`  // Stop floor profit % (TP1 only)
	StartTrailing    bool     `json:"start_trailing"`     // Start trailing after hit (TP3 only)
}

type TrailingConfig struct {
	PctTrail float64 `json:"pct_trail"` // % trail (e.g., 0.04 = 4%)
	ATRK     float64 `json:"atr_k"`     // ATR multiplier (e.g., 2.0)
}

type TimeStopConfig struct {
	MaxHoldDays       int     `json:"max_hold_days"`        // Max hold days (e.g., 10)
	NoMomentumDays    int     `json:"no_momentum_days"`     // No momentum days (e.g., 3)
	NoMomentumProfit  float64 `json:"no_momentum_profit"`   // No momentum profit % (e.g., 0.02)
}

type HardStopConfig struct {
	Enabled bool    `json:"enabled"` // Always on (even in PAUSE_ALL)
	Pct     float64 `json:"pct"`     // HardStop % (e.g., -0.10 = -10%)
}

// ====================
// OrderIntent (trade.order_intents)
// ====================

// OrderIntent represents exit intent (idempotent)
type OrderIntent struct {
	IntentID   uuid.UUID       `json:"intent_id"`
	PositionID uuid.UUID       `json:"position_id"`
	Symbol     string          `json:"symbol"`
	IntentType string          `json:"intent_type"` // EXIT_PARTIAL | EXIT_FULL
	Qty        int64           `json:"qty"`
	OrderType  string          `json:"order_type"` // MKT | LMT
	LimitPrice *decimal.Decimal `json:"limit_price"`
	ReasonCode string          `json:"reason_code"` // SL1 | SL2 | TP1 | TP2 | TP3 | TRAIL | TIME | MANUAL
	ActionKey  string          `json:"action_key"`  // {position_id}:SL1 (unique)
	Status     string          `json:"status"`      // NEW | ACK | REJECTED | FILLED
	CreatedTS  time.Time       `json:"created_ts"`
}

// Intent Types
const (
	IntentTypeExitPartial = "EXIT_PARTIAL"
	IntentTypeExitFull    = "EXIT_FULL"
)

// Order Types
const (
	OrderTypeMKT = "MKT"
	OrderTypeLMT = "LMT"
)

// Reason Codes
const (
	ReasonSL1           = "SL1"
	ReasonSL2           = "SL2"
	ReasonTP1           = "TP1"
	ReasonTP2           = "TP2"
	ReasonTP3           = "TP3"
	ReasonTrail         = "TRAIL"          // TP3 이후 잔량 전량 트레일
	ReasonTrailPartial  = "TRAIL_PARTIAL"  // Phase 1: TP2 이후 원본 20% 부분 트레일 (단발)
	ReasonTime          = "TIME"
	ReasonManual        = "MANUAL"
	ReasonHardStop      = "HARDSTOP"
	ReasonStopFloor     = "STOP_FLOOR"
)

// Intent Status
const (
	IntentStatusPendingApproval = "PENDING_APPROVAL" // 사용자 승인 대기
	IntentStatusNew             = "NEW"              // 승인됨, Execution 대기
	IntentStatusAck             = "ACK"              // Execution에서 처리 시작
	IntentStatusRejected        = "REJECTED"         // Execution 거부
	IntentStatusFilled          = "FILLED"           // 체결 완료
	IntentStatusCancelled       = "CANCELLED"        // 사용자가 취소
)

// ====================
// ExitTrigger (evaluation result)
// ====================

// ExitTrigger represents a triggered exit condition
type ExitTrigger struct {
	ReasonCode string
	Qty        int64
	OrderType  string
	LimitPrice *decimal.Decimal
}

// ====================
// SymbolExitOverride (trade.symbol_exit_overrides)
// ====================

// SymbolExitOverride represents symbol-level exit profile override
type SymbolExitOverride struct {
	Symbol        string     `json:"symbol"`
	ProfileID     string     `json:"profile_id"`
	Enabled       bool       `json:"enabled"`
	EffectiveFrom *time.Time `json:"effective_from"`
	Reason        string     `json:"reason"`
	CreatedBy     string     `json:"created_by"`
	CreatedTS     time.Time  `json:"created_ts"`
}

// ====================
// ExitSignal (trade.exit_signals) - Debugging/Backtest
// ====================

// ExitSignal represents exit trigger evaluation record (for debugging/backtest)
type ExitSignal struct {
	SignalID    uuid.UUID       `json:"signal_id"`
	PositionID  uuid.UUID       `json:"position_id"`
	RuleName    string          `json:"rule_name"` // HARD_STOP | SL1 | SL2 | TP1 | TP2 | TP3 | TRAIL | TIME
	IsTriggered bool            `json:"is_triggered"`
	Reason      string          `json:"reason"`
	Distance    *decimal.Decimal `json:"distance"` // Distance to trigger (debugging)
	Price       decimal.Decimal `json:"price"`
	EvaluatedTS time.Time       `json:"evaluated_ts"`
}
