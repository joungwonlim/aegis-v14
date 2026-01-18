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

// FetchLog 수집 실행 로그 (data.fetch_logs)
type FetchLog struct {
	ID              int       `json:"id" db:"id"`
	JobType         string    `json:"job_type" db:"job_type"`
	Source          string    `json:"source" db:"source"`
	TargetTable     string    `json:"target_table" db:"target_table"`
	RecordsFetched  int       `json:"records_fetched" db:"records_fetched"`
	RecordsInserted int       `json:"records_inserted" db:"records_inserted"`
	RecordsUpdated  int       `json:"records_updated" db:"records_updated"`
	Status          string    `json:"status" db:"status"` // pending, running, completed, failed
	ErrorMessage    *string   `json:"error_message" db:"error_message"`
	StartedAt       time.Time `json:"started_at" db:"started_at"`
	FinishedAt      *time.Time `json:"finished_at" db:"finished_at"`
	DurationMs      *int      `json:"duration_ms" db:"duration_ms"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
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

// =============================================================================
// Stock Rankings (data.stock_rankings)
// =============================================================================

// RankingStock 순위 데이터 (data.stock_rankings)
type RankingStock struct {
	ID        int64     `json:"id" db:"id"`
	Category  string    `json:"category" db:"category"`   // volume, trading_value, gainers, etc
	Market    string    `json:"market" db:"market"`       // ALL, KOSPI, KOSDAQ
	Rank      int       `json:"rank" db:"rank"`
	StockCode string    `json:"stock_code" db:"stock_code"`
	StockName string    `json:"stock_name" db:"stock_name"`

	// 공통 가격 정보
	CurrentPrice  *float64 `json:"current_price,omitempty" db:"current_price"`
	ChangeRate    *float64 `json:"change_rate,omitempty" db:"change_rate"`
	Volume        *int64   `json:"volume,omitempty" db:"volume"`
	TradingValue  *int64   `json:"trading_value,omitempty" db:"trading_value"`
	HighPrice     *float64 `json:"high_price,omitempty" db:"high_price"`
	LowPrice      *float64 `json:"low_price,omitempty" db:"low_price"`

	// 카테고리별 특화 데이터
	ForeignNetValue  *int64   `json:"foreign_net_value,omitempty" db:"foreign_net_value"`
	InstNetValue     *int64   `json:"inst_net_value,omitempty" db:"inst_net_value"`
	VolumeSurgeRate  *float64 `json:"volume_surge_rate,omitempty" db:"volume_surge_rate"`
	High52Week       *float64 `json:"high_52week,omitempty" db:"high_52week"`
	MarketCap        *int64   `json:"market_cap,omitempty" db:"market_cap"`

	CollectedAt time.Time `json:"collected_at" db:"collected_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
