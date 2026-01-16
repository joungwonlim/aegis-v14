package execution

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order Status
const (
	OrderStatusSubmitted       = "SUBMITTED"        // 제출됨
	OrderStatusPartial         = "PARTIAL"          // 부분 체결
	OrderStatusFilled          = "FILLED"           // 전량 체결
	OrderStatusRejected        = "REJECTED"         // 거부됨
	OrderStatusCancelled       = "CANCELLED"        // 취소됨
	OrderStatusCancelledPartial = "CANCELLED_PARTIAL" // 부분 체결 후 취소
	OrderStatusError           = "ERROR"            // 에러
	OrderStatusUnknown         = "UNKNOWN"          // 알 수 없음
)

// Order Type
const (
	OrderTypeMarket = "MKT" // 시장가
	OrderTypeLimit  = "LMT" // 지정가
)

// Order represents a broker order
type Order struct {
	OrderID      string           `json:"order_id"`       // KIS 주문번호 (PK)
	IntentID     uuid.UUID        `json:"intent_id"`      // 원본 의도 ID (FK)
	SubmittedTS  time.Time        `json:"submitted_ts"`   // 제출 시각
	Status       string           `json:"status"`         // 주문 상태
	BrokerStatus string           `json:"broker_status"`  // 브로커 원본 상태
	Qty          int64            `json:"qty"`            // 주문 수량
	OpenQty      int64            `json:"open_qty"`       // 미체결 수량
	FilledQty    int64            `json:"filled_qty"`     // 체결 수량
	Raw          map[string]any   `json:"raw"`            // KIS API 응답 원본
	UpdatedTS    time.Time        `json:"updated_ts"`     // 마지막 갱신
}

// Fill represents an execution/fill event
type Fill struct {
	FillID    string          `json:"fill_id"`     // 체결 고유 ID (PK)
	OrderID   string          `json:"order_id"`    // 주문 ID (FK)
	KisExecID string          `json:"kis_exec_id"` // KIS 체결번호 (Unique)
	TS        time.Time       `json:"ts"`          // 체결 시각
	Qty       int64           `json:"qty"`         // 체결 수량
	Price     decimal.Decimal `json:"price"`       // 체결 가격
	Fee       decimal.Decimal `json:"fee"`         // 수수료
	Tax       decimal.Decimal `json:"tax"`         // 세금
	Seq       int             `json:"seq"`         // 순번 (cursor용)
}

// Holding represents a broker holding (KIS 보유 현황)
type Holding struct {
	AccountID    string          `json:"account_id"`    // 계좌번호 (PK)
	Symbol       string          `json:"symbol"`        // 종목 코드 (PK)
	Qty          int64           `json:"qty"`           // 보유 수량
	AvgPrice     decimal.Decimal `json:"avg_price"`     // 평균 단가
	CurrentPrice decimal.Decimal `json:"current_price"` // 현재가 (참고용)
	Pnl          decimal.Decimal `json:"pnl"`           // 평가손익 (매입단가 대비)
	PnlPct       float64         `json:"pnl_pct"`       // 수익률 (%, 매입단가 대비)
	ChangePrice  int64           `json:"change_price"`  // 전일대비 가격 (원, from prices_best)
	ChangeRate   float64         `json:"change_rate"`   // 전일대비 등락률 (%, from prices_best)
	UpdatedTS    time.Time       `json:"updated_ts"`    // 마지막 동기화 시각
	ExitMode     string          `json:"exit_mode"`     // Exit Engine 모드 (ENABLED/DISABLED/MANUAL_ONLY)
	Raw          map[string]any  `json:"raw"`           // KIS API 원본
}

// ExitEvent represents a position exit event (SSOT)
type ExitEvent struct {
	ExitEventID    uuid.UUID       `json:"exit_event_id"`    // 청산 이벤트 ID (PK)
	PositionID     uuid.UUID       `json:"position_id"`      // 포지션 ID (FK, Unique)
	AccountID      string          `json:"account_id"`       // 계좌번호
	Symbol         string          `json:"symbol"`           // 종목 코드
	ExitTS         time.Time       `json:"exit_ts"`          // 청산 시각
	ExitQty        int64           `json:"exit_qty"`         // 청산 수량
	ExitAvgPrice   decimal.Decimal `json:"exit_avg_price"`   // 청산 평균가
	ExitReasonCode string          `json:"exit_reason_code"` // 청산 사유 (SL1, SL2, TP1, TP2, TP3, TRAIL, TIME, MANUAL, BROKER)
	Source         string          `json:"source"`           // 청산 소스 (AUTO_EXIT, MANUAL, BROKER, UNKNOWN)
	IntentID       *uuid.UUID      `json:"intent_id"`        // 원본 EXIT intent ID (optional)
	ExitProfileID  *string         `json:"exit_profile_id"`  // Exit Profile ID (optional)
	RealizedPnl    decimal.Decimal `json:"realized_pnl"`     // 실현손익
	RealizedPnlPct float64         `json:"realized_pnl_pct"` // 실현수익률 (%)
	CreatedTS      time.Time       `json:"created_ts"`       // 생성 시각
}

// Exit Event Source
const (
	ExitSourceAutoExit = "AUTO_EXIT" // 전략 자동 청산
	ExitSourceManual   = "MANUAL"    // 수동 청산
	ExitSourceBroker   = "BROKER"    // 브로커 강제청산
	ExitSourceUnknown  = "UNKNOWN"   // 알 수 없음
)

// Exit Reason Codes (from Exit Engine)
const (
	ExitReasonSL1    = "SL1"    // Stop Loss 1
	ExitReasonSL2    = "SL2"    // Stop Loss 2
	ExitReasonTP1    = "TP1"    // Take Profit 1
	ExitReasonTP2    = "TP2"    // Take Profit 2
	ExitReasonTP3    = "TP3"    // Take Profit 3
	ExitReasonTrail  = "TRAIL"  // Trailing Stop
	ExitReasonTime   = "TIME"   // Time Stop
	ExitReasonManual = "MANUAL" // Manual Exit
	ExitReasonBroker = "BROKER" // Broker Forced Exit
)

// FillCursor for fills sync cursor
type FillCursor struct {
	LastTS  time.Time `json:"last_ts"`  // 마지막 체결 시각
	LastSeq int       `json:"last_seq"` // 마지막 순번
}
