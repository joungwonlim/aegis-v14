package reentry

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReentryCandidate represents a reentry candidate FSM
type ReentryCandidate struct {
	CandidateID       uuid.UUID       `json:"candidate_id"`        // 후보 고유 ID (PK)
	ExitEventID       uuid.UUID       `json:"exit_event_id"`       // ExitEvent 참조 (SSOT, UNIQUE)
	Symbol            string          `json:"symbol"`              // 종목 코드
	OriginPositionID  uuid.UUID       `json:"origin_position_id"`  // 원 포지션 ID
	ExitReasonCode    string          `json:"exit_reason_code"`    // SL1/SL2/TRAIL/TP1/TP2/TP3/TIME
	ExitTS            time.Time       `json:"exit_ts"`             // 청산 시각
	ExitPrice         decimal.Decimal `json:"exit_price"`          // 청산 가격
	ExitProfileID     *string         `json:"exit_profile_id"`     // 적용된 Exit 프로파일
	CooldownUntil     time.Time       `json:"cooldown_until"`      // 쿨다운 종료 시각
	State             string          `json:"state"`               // FSM 상태
	MaxReentries      int             `json:"max_reentries"`       // 최대 재진입 횟수
	ReentryCount      int             `json:"reentry_count"`       // 현재 재진입 횟수
	ReentryProfileID  *string         `json:"reentry_profile_id"`  // 재진입 프로파일
	LastEvalTS        *time.Time      `json:"last_eval_ts"`        // 마지막 평가 시각
	UpdatedTS         time.Time       `json:"updated_ts"`          // 마지막 갱신
}

// Candidate FSM States
const (
	StateCooldown = "COOLDOWN" // 쿨다운 대기
	StateWatch    = "WATCH"    // 모니터링 중
	StateReady    = "READY"    // 진입 준비
	StateEntered  = "ENTERED"  // 진입 완료
	StateExpired  = "EXPIRED"  // 만료됨
	StateBlocked  = "BLOCKED"  // 차단됨
)

// ReentryControl represents global reentry control (kill switch)
type ReentryControl struct {
	ID        int       `json:"id"` // Always 1 (singleton)
	Mode      string    `json:"mode"`
	Reason    *string   `json:"reason"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedTS time.Time `json:"updated_ts"`
}

// Control Modes
const (
	ControlModeRunning    = "RUNNING"     // 정상 작동
	ControlModePauseEntry = "PAUSE_ENTRY" // Candidate 추적만 (진입 차단)
	ControlModePauseAll   = "PAUSE_ALL"   // 완전 정지
)

// ReentryProfile represents reentry strategy configuration
type ReentryProfile struct {
	ProfileID       string              `json:"profile_id"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Config          ReentryProfileConfig `json:"config"`
	IsActive        bool                `json:"is_active"`
	CreatedBy       string              `json:"created_by"`
	CreatedTS       time.Time           `json:"created_ts"`
}

// ReentryProfileConfig represents reentry configuration
type ReentryProfileConfig struct {
	// Cooldown settings (by exit reason)
	CooldownSL   int `json:"cooldown_sl"`   // SL 후 쿨다운 (초)
	CooldownTP   int `json:"cooldown_tp"`   // TP 후 쿨다운 (초)
	CooldownTime int `json:"cooldown_time"` // TIME 후 쿨다운 (초)

	// Reentry limits
	MaxReentries  int `json:"max_reentries"`   // 최대 재진입 횟수
	MaxWatchHours int `json:"max_watch_hours"` // 최대 모니터링 시간

	// Trigger settings (different strategies)
	TriggerRebound  ReboundConfig  `json:"trigger_rebound"`  // Rebound 전략
	TriggerBreakout BreakoutConfig `json:"trigger_breakout"` // Breakout 전략
	TriggerChase    ChaseConfig    `json:"trigger_chase"`    // Chase 전략

	// Position sizing
	SizingMode    string  `json:"sizing_mode"`     // FIXED | PERCENT | KELLY
	SizingPercent float64 `json:"sizing_percent"`  // 포트폴리오 %
	SizingMax     int64   `json:"sizing_max"`      // 최대 수량
}

// ReboundConfig for rebound strategy (SL → bounce back)
type ReboundConfig struct {
	Enabled       bool    `json:"enabled"`
	BouncePercent float64 `json:"bounce_percent"` // Exit price 대비 반등 %
	MinVolume     int64   `json:"min_volume"`     // 최소 거래량
}

// BreakoutConfig for breakout strategy (TP → continue)
type BreakoutConfig struct {
	Enabled       bool    `json:"enabled"`
	BreakPercent  float64 `json:"break_percent"`  // Exit price 대비 돌파 %
	MinVolume     int64   `json:"min_volume"`     // 최소 거래량
}

// ChaseConfig for chase strategy (momentum)
type ChaseConfig struct {
	Enabled       bool    `json:"enabled"`
	ChasePercent  float64 `json:"chase_percent"`  // Exit price 대비 상승 %
	MinVolume     int64   `json:"min_volume"`     // 최소 거래량
}

// Reentry Reason Codes
const (
	ReasonReentryRebound  = "REENTRY_REBOUND"  // Rebound 전략
	ReasonReentryBreakout = "REENTRY_BREAKOUT" // Breakout 전략
	ReasonReentryChase    = "REENTRY_CHASE"    // Chase 전략
)

// Sizing Modes
const (
	SizingModeFixed   = "FIXED"   // 고정 수량
	SizingModePercent = "PERCENT" // 포트폴리오 %
	SizingModeKelly   = "KELLY"   // Kelly Criterion
)
