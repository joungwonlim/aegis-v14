package execution

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// KISAdapter is an interface for KIS API operations
type KISAdapter interface {
	// SubmitOrder submits an order to KIS
	SubmitOrder(ctx context.Context, req KISOrderRequest) (*KISOrderResponse, error)

	// GetUnfilledOrders retrieves unfilled orders from KIS
	GetUnfilledOrders(ctx context.Context, accountID string) ([]*KISUnfilledOrder, error)

	// GetFills retrieves fills since timestamp
	GetFills(ctx context.Context, accountID string, since time.Time) ([]*KISFill, error)

	// GetFillsForOrder retrieves fills for a specific order
	GetFillsForOrder(ctx context.Context, orderID string) ([]*KISFill, error)

	// GetHoldings retrieves holdings from KIS
	GetHoldings(ctx context.Context, accountID string) ([]*KISHolding, error)
}

// KISOrderRequest represents KIS order submission request
type KISOrderRequest struct {
	AccountID  string          // 계좌번호
	Symbol     string          // 종목 코드
	Side       string          // BUY, SELL
	OrderType  string          // MKT, LMT
	Qty        int64           // 수량
	LimitPrice *decimal.Decimal // 지정가 (LMT only)
}

// KISOrderResponse represents KIS order submission response
type KISOrderResponse struct {
	OrderID   string         // KIS 주문번호
	Timestamp time.Time      // 주문 시각
	Raw       map[string]any // 원본 응답
}

// KISUnfilledOrder represents unfilled order from KIS
type KISUnfilledOrder struct {
	OrderID   string         // 주문번호
	Symbol    string         // 종목 코드
	Qty       int64          // 주문 수량
	OpenQty   int64          // 미체결 수량
	FilledQty int64          // 체결 수량
	Status    string         // 주문 상태
	Raw       map[string]any // 원본 응답
}

// KISFill represents fill from KIS
type KISFill struct {
	ExecID    string          // 체결번호
	OrderID   string          // 주문번호
	Symbol    string          // 종목 코드
	Qty       int64           // 체결 수량
	Price     decimal.Decimal // 체결 가격
	Fee       decimal.Decimal // 수수료
	Tax       decimal.Decimal // 세금
	Timestamp time.Time       // 체결 시각
	Seq       int             // 순번
	Raw       map[string]any  // 원본 응답
}

// KISHolding represents holding from KIS
type KISHolding struct {
	AccountID    string          // 계좌번호
	Symbol       string          // 종목 코드
	Qty          int64           // 보유 수량
	AvgPrice     decimal.Decimal // 평균 단가
	CurrentPrice decimal.Decimal // 현재가
	Pnl          decimal.Decimal // 평가손익
	PnlPct       float64         // 수익률 (%)
	Raw          map[string]any  // 원본 응답
}

// Side
const (
	SideBuy  = "BUY"  // 매수
	SideSell = "SELL" // 매도
)
