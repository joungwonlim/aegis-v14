package audit

import (
	"errors"
	"time"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrReportNotFound    = errors.New("performance report not found")
	ErrSnapshotNotFound  = errors.New("daily snapshot not found")
	ErrInsufficientData  = errors.New("insufficient data for calculation")
	ErrInvalidPeriod     = errors.New("invalid period")
)

// =============================================================================
// Period Types
// =============================================================================

// Period 리포트 기간 타입
type Period string

const (
	Period1M  Period = "1M"
	Period3M  Period = "3M"
	Period6M  Period = "6M"
	Period1Y  Period = "1Y"
	PeriodYTD Period = "YTD"
)

// ValidPeriods 유효한 기간 목록
var ValidPeriods = []Period{Period1M, Period3M, Period6M, Period1Y, PeriodYTD}

// IsValid 유효한 기간인지 확인
func (p Period) IsValid() bool {
	for _, valid := range ValidPeriods {
		if p == valid {
			return true
		}
	}
	return false
}

// =============================================================================
// Performance Report
// =============================================================================

// PerformanceReport 성과 리포트
type PerformanceReport struct {
	Period    Period    `json:"period"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// 수익률 지표
	TotalReturn  float64 `json:"total_return"`   // 누적 수익률
	AnnualReturn float64 `json:"annual_return"`  // 연환산 수익률

	// 리스크 지표
	Volatility  float64 `json:"volatility"`   // 연환산 변동성
	Sharpe      float64 `json:"sharpe"`       // 샤프 비율
	Sortino     float64 `json:"sortino"`      // 소르티노 비율
	MaxDrawdown float64 `json:"max_drawdown"` // 최대 낙폭 (MDD)

	// 트레이딩 지표
	WinRate      float64 `json:"win_rate"`      // 승률
	AvgWin       float64 `json:"avg_win"`       // 평균 수익
	AvgLoss      float64 `json:"avg_loss"`      // 평균 손실
	ProfitFactor float64 `json:"profit_factor"` // 수익 팩터
	TotalTrades  int     `json:"total_trades"`  // 총 거래 수

	// 벤치마크 비교
	Benchmark float64 `json:"benchmark"` // 벤치마크 수익률
	Alpha     float64 `json:"alpha"`     // 초과 수익률
	Beta      float64 `json:"beta"`      // 시장 민감도
}

// =============================================================================
// Daily PnL
// =============================================================================

// DailyPnL 일별 손익
type DailyPnL struct {
	Date             time.Time `json:"date"`
	RealizedPnL      int64     `json:"realized_pnl"`      // 실현 손익
	UnrealizedPnL    int64     `json:"unrealized_pnl"`    // 평가 손익
	TotalPnL         int64     `json:"total_pnl"`         // 총 손익
	DailyReturn      float64   `json:"daily_return"`      // 일별 수익률
	CumulativeReturn float64   `json:"cumulative_return"` // 누적 수익률
	PortfolioValue   int64     `json:"portfolio_value"`   // 포트폴리오 가치
	CashBalance      int64     `json:"cash_balance"`      // 현금 잔고
}

// =============================================================================
// Daily Snapshot
// =============================================================================

// DailySnapshot 일별 포트폴리오 스냅샷
type DailySnapshot struct {
	Date        time.Time          `json:"date"`
	TotalValue  int64              `json:"total_value"`
	Cash        int64              `json:"cash"`
	Positions   []PositionSnapshot `json:"positions"`
	DailyReturn float64            `json:"daily_return"`
	CumReturn   float64            `json:"cum_return"`
}

// PositionSnapshot 포지션 스냅샷
type PositionSnapshot struct {
	Symbol       string  `json:"symbol"`
	Quantity     int     `json:"quantity"`
	AvgPrice     int64   `json:"avg_price"`
	CurrentPrice int64   `json:"current_price"`
	MarketValue  int64   `json:"market_value"`
	UnrealizedPL int64   `json:"unrealized_pl"`
	Weight       float64 `json:"weight"`
}

// =============================================================================
// Attribution Analysis
// =============================================================================

// AttributionAnalysis 귀속 분석
type AttributionAnalysis struct {
	AnalysisDate time.Time          `json:"analysis_date"`
	PeriodStart  time.Time          `json:"period_start"`
	PeriodEnd    time.Time          `json:"period_end"`
	TotalReturn  float64            `json:"total_return"`
	Factors      []FactorAttribution `json:"factors"`
	Sectors      map[string]float64 `json:"sectors"`
	Stocks       map[string]float64 `json:"stocks"`
}

// FactorAttribution 팩터별 기여도
type FactorAttribution struct {
	Factor       string  `json:"factor"`
	Contribution float64 `json:"contribution"` // 수익 기여도
	Exposure     float64 `json:"exposure"`     // 평균 노출도
	ReturnPct    float64 `json:"return_pct"`   // 기여 수익률
}

// =============================================================================
// Trade
// =============================================================================

// Trade 거래 내역
type Trade struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // BUY or SELL
	Quantity   int       `json:"quantity"`
	Price      int64     `json:"price"`
	PnL        float64   `json:"pnl"`
	PnLPercent float64   `json:"pnl_percent"`
	EntryDate  time.Time `json:"entry_date"`
	ExitDate   time.Time `json:"exit_date"`
	HoldDays   int       `json:"hold_days"`
}

// =============================================================================
// Benchmark
// =============================================================================

// BenchmarkData 벤치마크 데이터
type BenchmarkData struct {
	Date        time.Time `json:"date"`
	Code        string    `json:"code"` // KOSPI, KOSDAQ
	ClosePrice  float64   `json:"close_price"`
	DailyReturn float64   `json:"daily_return"`
}

// =============================================================================
// Risk Metrics
// =============================================================================

// RiskMetrics 리스크 지표
type RiskMetrics struct {
	VaR95       float64 `json:"var_95"`       // 95% VaR
	VaR99       float64 `json:"var_99"`       // 99% VaR
	CVaR95      float64 `json:"cvar_95"`      // 95% CVaR
	CVaR99      float64 `json:"cvar_99"`      // 99% CVaR
	MaxDrawdown float64 `json:"max_drawdown"` // MDD
	Volatility  float64 `json:"volatility"`   // 변동성
}
