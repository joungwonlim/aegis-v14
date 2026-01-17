package fetcher

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Data Models (data schema)
// =============================================================================

// Stock 종목 마스터 (data.stocks)
type Stock struct {
	Code          string     `json:"code" db:"code"`
	Name          string     `json:"name" db:"name"`
	Market        string     `json:"market" db:"market"`               // KOSPI, KOSDAQ, KONEX
	Sector        *string    `json:"sector" db:"sector"`
	ListingDate   time.Time  `json:"listing_date" db:"listing_date"`
	DelistingDate *time.Time `json:"delisting_date" db:"delisting_date"`
	Status        string     `json:"status" db:"status"`               // active, delisted, suspended
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// DailyPrice 일봉 데이터 (data.daily_prices)
type DailyPrice struct {
	StockCode    string    `json:"stock_code" db:"stock_code"`
	TradeDate    time.Time `json:"trade_date" db:"trade_date"`
	OpenPrice    float64   `json:"open_price" db:"open_price"`
	HighPrice    float64   `json:"high_price" db:"high_price"`
	LowPrice     float64   `json:"low_price" db:"low_price"`
	ClosePrice   float64   `json:"close_price" db:"close_price"`
	Volume       int64     `json:"volume" db:"volume"`
	TradingValue int64     `json:"trading_value" db:"trading_value"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// InvestorFlow 투자자별 수급 (data.investor_flow)
type InvestorFlow struct {
	StockCode       string    `json:"stock_code" db:"stock_code"`
	TradeDate       time.Time `json:"trade_date" db:"trade_date"`
	ForeignNetQty   int64     `json:"foreign_net_qty" db:"foreign_net_qty"`
	ForeignNetValue int64     `json:"foreign_net_value" db:"foreign_net_value"`
	InstNetQty      int64     `json:"inst_net_qty" db:"inst_net_qty"`
	InstNetValue    int64     `json:"inst_net_value" db:"inst_net_value"`
	IndivNetQty     int64     `json:"indiv_net_qty" db:"indiv_net_qty"`
	IndivNetValue   int64     `json:"indiv_net_value" db:"indiv_net_value"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Fundamentals 재무 데이터 (data.fundamentals)
type Fundamentals struct {
	StockCode       string    `json:"stock_code" db:"stock_code"`
	ReportDate      time.Time `json:"report_date" db:"report_date"`
	PER             *float64  `json:"per" db:"per"`
	PBR             *float64  `json:"pbr" db:"pbr"`
	PSR             *float64  `json:"psr" db:"psr"`
	ROE             *float64  `json:"roe" db:"roe"`
	DebtRatio       *float64  `json:"debt_ratio" db:"debt_ratio"`
	Revenue         *int64    `json:"revenue" db:"revenue"`
	OperatingProfit *int64    `json:"operating_profit" db:"operating_profit"`
	NetProfit       *int64    `json:"net_profit" db:"net_profit"`
	EPS             *float64  `json:"eps" db:"eps"`
	BPS             *float64  `json:"bps" db:"bps"`
	DPS             *float64  `json:"dps" db:"dps"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// MarketCap 시가총액 (data.market_cap)
type MarketCap struct {
	StockCode   string    `json:"stock_code" db:"stock_code"`
	TradeDate   time.Time `json:"trade_date" db:"trade_date"`
	MarketCap   int64     `json:"market_cap" db:"market_cap"`
	SharesOut   *int64    `json:"shares_out" db:"shares_out"`
	FloatShares *int64    `json:"float_shares" db:"float_shares"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Disclosure DART 공시 (data.disclosures)
type Disclosure struct {
	ID          int64     `json:"id" db:"id"`
	StockCode   string    `json:"stock_code" db:"stock_code"`
	DisclosedAt time.Time `json:"disclosed_at" db:"disclosed_at"`
	Title       string    `json:"title" db:"title"`
	Category    *string   `json:"category" db:"category"`
	Subcategory *string   `json:"subcategory" db:"subcategory"`
	Content     *string   `json:"content" db:"content"`
	URL         *string   `json:"url" db:"url"`
	DartRceptNo *string   `json:"dart_rcept_no" db:"dart_rcept_no"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// =============================================================================
// Job/Result Models
// =============================================================================

// FetchJobType 수집 작업 유형
type FetchJobType string

const (
	JobTypePrice      FetchJobType = "price"
	JobTypeFlow       FetchJobType = "flow"
	JobTypeMarketCap  FetchJobType = "marketcap"
	JobTypeDisclosure FetchJobType = "disclosure"
	JobTypeFundamental FetchJobType = "fundamental"
	JobTypeAll        FetchJobType = "all"
)

// FetchStatus 수집 상태
type FetchStatus string

const (
	StatusPending   FetchStatus = "pending"
	StatusRunning   FetchStatus = "running"
	StatusCompleted FetchStatus = "completed"
	StatusFailed    FetchStatus = "failed"
)

// FetchJob 수집 작업
type FetchJob struct {
	JobID       uuid.UUID    `json:"job_id"`
	JobType     FetchJobType `json:"job_type"`
	Status      FetchStatus  `json:"status"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Error       string       `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// FetchResult 수집 결과
type FetchResult struct {
	JobID        uuid.UUID `json:"job_id"`
	Source       string    `json:"source"`       // naver, dart, krx, kis
	Target       string    `json:"target"`       // prices, flow, fundamentals, disclosures
	TotalCount   int       `json:"total_count"`
	SuccessCount int       `json:"success_count"`
	FailedCount  int       `json:"failed_count"`
	Duration     float64   `json:"duration_sec"`
	Errors       []string  `json:"errors,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

// =============================================================================
// Config Models
// =============================================================================

// CollectorConfig 수집기 설정
type CollectorConfig struct {
	Workers     int           `json:"workers"`       // 병렬 작업자 수
	Timeout     time.Duration `json:"timeout"`       // 요청 타임아웃
	RetryCount  int           `json:"retry_count"`   // 재시도 횟수
	RateLimitMs int           `json:"rate_limit_ms"` // 요청 간격 (ms)
}

// DefaultCollectorConfig 기본 설정
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		Workers:     5,
		Timeout:     30 * time.Second,
		RetryCount:  3,
		RateLimitMs: 100,
	}
}

// =============================================================================
// Filter Models
// =============================================================================

// StockFilter 종목 조회 필터
type StockFilter struct {
	Market string // KOSPI, KOSDAQ, ALL
	Status string // active, delisted, suspended, all
	Limit  int
	Offset int
}

// PriceFilter 가격 조회 필터
type PriceFilter struct {
	StockCode string
	From      time.Time
	To        time.Time
	Limit     int
}
