package fetcher

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Stock Repository
// =============================================================================

// StockRepository 종목 저장소 (data.stocks)
type StockRepository interface {
	// Upsert 종목 저장 (있으면 업데이트, 없으면 생성)
	Upsert(ctx context.Context, stock *Stock) error
	UpsertBatch(ctx context.Context, stocks []*Stock) (int, error)

	// Query 종목 조회
	GetByCode(ctx context.Context, code string) (*Stock, error)
	GetByMarket(ctx context.Context, market string) ([]*Stock, error)
	GetActive(ctx context.Context) ([]*Stock, error)
	List(ctx context.Context, filter *StockFilter) ([]*Stock, error)
	Count(ctx context.Context, filter *StockFilter) (int, error)
}

// =============================================================================
// Price Repository
// =============================================================================

// PriceRepository 가격 저장소 (data.daily_prices)
type PriceRepository interface {
	// Upsert 가격 저장
	Upsert(ctx context.Context, price *DailyPrice) error
	UpsertBatch(ctx context.Context, prices []*DailyPrice) (int, error)

	// Query 가격 조회
	GetByDate(ctx context.Context, stockCode string, date time.Time) (*DailyPrice, error)
	GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*DailyPrice, error)
	GetLatest(ctx context.Context, stockCode string) (*DailyPrice, error)
	GetLatestN(ctx context.Context, stockCode string, n int) ([]*DailyPrice, error)
}

// =============================================================================
// Flow Repository
// =============================================================================

// FlowRepository 수급 저장소 (data.investor_flow)
type FlowRepository interface {
	// Upsert 수급 저장
	Upsert(ctx context.Context, flow *InvestorFlow) error
	UpsertBatch(ctx context.Context, flows []*InvestorFlow) (int, error)

	// Query 수급 조회
	GetByDate(ctx context.Context, stockCode string, date time.Time) (*InvestorFlow, error)
	GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*InvestorFlow, error)
	GetLatest(ctx context.Context, stockCode string) (*InvestorFlow, error)
}

// =============================================================================
// Fundamentals Repository
// =============================================================================

// FundamentalsRepository 재무 저장소 (data.fundamentals)
type FundamentalsRepository interface {
	// Upsert 재무 저장
	Upsert(ctx context.Context, fund *Fundamentals) error
	UpsertBatch(ctx context.Context, funds []*Fundamentals) (int, error)

	// Query 재무 조회
	GetLatest(ctx context.Context, stockCode string) (*Fundamentals, error)
	GetByDate(ctx context.Context, stockCode string, date time.Time) (*Fundamentals, error)
	GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*Fundamentals, error)
}

// =============================================================================
// MarketCap Repository
// =============================================================================

// MarketCapRepository 시가총액 저장소 (data.market_cap)
type MarketCapRepository interface {
	// Upsert 시가총액 저장
	Upsert(ctx context.Context, mc *MarketCap) error
	UpsertBatch(ctx context.Context, mcs []*MarketCap) (int, error)

	// Query 시가총액 조회
	GetLatest(ctx context.Context, stockCode string) (*MarketCap, error)
	GetByDate(ctx context.Context, stockCode string, date time.Time) (*MarketCap, error)
	GetTopN(ctx context.Context, date time.Time, n int) ([]*MarketCap, error)
}

// =============================================================================
// Disclosure Repository
// =============================================================================

// DisclosureRepository 공시 저장소 (data.disclosures)
type DisclosureRepository interface {
	// Save 공시 저장 (새 공시만 추가, 중복 무시)
	Save(ctx context.Context, disc *Disclosure) error
	SaveBatch(ctx context.Context, discs []*Disclosure) (int, error)

	// Query 공시 조회
	GetByStock(ctx context.Context, stockCode string, from, to time.Time) ([]*Disclosure, error)
	GetByCategory(ctx context.Context, category string, from, to time.Time) ([]*Disclosure, error)
	GetRecent(ctx context.Context, limit int) ([]*Disclosure, error)
	ExistsByDartRceptNo(ctx context.Context, dartRceptNo string) (bool, error)
}

// =============================================================================
// Job Repository
// =============================================================================

// JobRepository 수집 작업 저장소
type JobRepository interface {
	// Job 관리
	Create(ctx context.Context, job *FetchJob) error
	Update(ctx context.Context, job *FetchJob) error
	GetByID(ctx context.Context, jobID uuid.UUID) (*FetchJob, error)
	GetLatest(ctx context.Context, jobType FetchJobType) (*FetchJob, error)
	List(ctx context.Context, from, to time.Time) ([]*FetchJob, error)
}

// =============================================================================
// FetchLog Repository
// =============================================================================

// FetchLogRepository 수집 실행 로그 저장소 (data.fetch_logs)
type FetchLogRepository interface {
	// Create 로그 생성 (실행 시작 시)
	Create(ctx context.Context, log *FetchLog) (*FetchLog, error)

	// Update 로그 업데이트 (실행 완료/실패 시)
	Update(ctx context.Context, log *FetchLog) error

	// Query 로그 조회
	GetByID(ctx context.Context, id int) (*FetchLog, error)
	GetRecent(ctx context.Context, limit int) ([]*FetchLog, error)
	GetByJobType(ctx context.Context, jobType string, limit int) ([]*FetchLog, error)
	GetByStatus(ctx context.Context, status string, limit int) ([]*FetchLog, error)
}

// =============================================================================
// External Client Interfaces
// =============================================================================

// NaverClient 네이버 금융 클라이언트
type NaverClient interface {
	// 가격 데이터 수집
	FetchDailyPrices(ctx context.Context, stockCode string, days int) ([]*DailyPrice, error)

	// 수급 데이터 수집
	FetchInvestorFlow(ctx context.Context, stockCode string, days int) ([]*InvestorFlow, error)

	// 시가총액 수집
	FetchMarketCap(ctx context.Context, stockCode string) (*MarketCap, error)

	// 종목 정보 수집
	FetchStockInfo(ctx context.Context, stockCode string) (*Stock, error)

	// 재무 데이터 수집
	FetchFundamentals(ctx context.Context, stockCode string) (*Fundamentals, error)

	// 시가총액 순위 수집
	FetchMarketCapRanking(ctx context.Context, market string, limit int) ([]*Stock, error)
}

// DartClient DART 공시 클라이언트
type DartClient interface {
	// 특정 종목 공시 수집
	FetchDisclosures(ctx context.Context, corpCode string, from, to time.Time) ([]*Disclosure, error)

	// 전체 공시 수집
	FetchAllDisclosures(ctx context.Context, from, to time.Time) ([]*Disclosure, error)

	// 재무제표 수집
	FetchFinancials(ctx context.Context, corpCode string, year int, reportCode string) (*Fundamentals, error)

	// 연결 확인
	HealthCheck(ctx context.Context) error
}
