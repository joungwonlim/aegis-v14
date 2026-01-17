# v13 → v14 마이그레이션 구현 가이드

> Sonnet을 위한 상세 개발 가이드

---

## 📋 개요

이 문서는 v13의 핵심 모듈(Fetcher, Signals, Audit)을 v14 아키텍처로 마이그레이션하기 위한 상세 구현 가이드입니다.

### 마이그레이션 대상

| 모듈 | v13 위치 | v14 위치 | 우선순위 |
|------|----------|----------|----------|
| Fetcher | `internal/s0_data/` | `internal/domain/fetcher/` + `internal/service/fetcher/` | P0 |
| Signals (6팩터) | `internal/s2_signals/` | `internal/domain/signals/` + `internal/service/signals/` | P1 |
| Audit | `internal/audit/` | `internal/domain/audit/` + `internal/service/audit/` | P2 |

### v14 아키텍처 패턴 (필수 준수)

```
internal/
├── domain/           # 도메인 모델, 리포지토리 인터페이스, 에러
│   └── {module}/
│       ├── model.go      # 도메인 모델 (struct)
│       ├── repository.go # 리포지토리 인터페이스
│       └── errors.go     # 도메인 에러
├── service/          # 비즈니스 로직
│   └── {module}/
│       └── service.go    # 서비스 구현
├── infrastructure/   # 리포지토리 구현체
│   └── postgres/
│       └── {module}/
│           └── repository.go
└── api/              # HTTP 핸들러
    ├── handlers/
    │   └── {module}/
    │       └── handler.go
    └── routes/
        └── {module}_routes.go
```

---

## 🔴 P0: Fetcher 모듈 구현

### Step 1: 도메인 모델 생성

**파일**: `internal/domain/fetcher/model.go`

```go
package fetcher

import "time"

// Stock 종목 마스터
type Stock struct {
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Market       string    `json:"market"` // KOSPI, KOSDAQ
	Sector       string    `json:"sector"`
	ListingDate  time.Time `json:"listing_date"`
	DelistingDate *time.Time `json:"delisting_date"`
	Status       string    `json:"status"` // active, delisted, suspended
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DailyPrice 일봉 데이터
type DailyPrice struct {
	StockCode    string    `json:"stock_code"`
	TradeDate    time.Time `json:"trade_date"`
	OpenPrice    float64   `json:"open_price"`
	HighPrice    float64   `json:"high_price"`
	LowPrice     float64   `json:"low_price"`
	ClosePrice   float64   `json:"close_price"`
	Volume       int64     `json:"volume"`
	TradingValue int64     `json:"trading_value"`
}

// InvestorFlow 투자자별 수급
type InvestorFlow struct {
	StockCode      string    `json:"stock_code"`
	TradeDate      time.Time `json:"trade_date"`
	ForeignNetQty  int64     `json:"foreign_net_qty"`
	ForeignNetValue int64    `json:"foreign_net_value"`
	InstNetQty     int64     `json:"inst_net_qty"`
	InstNetValue   int64     `json:"inst_net_value"`
	IndivNetQty    int64     `json:"indiv_net_qty"`
	IndivNetValue  int64     `json:"indiv_net_value"`
}

// Fundamentals 재무 데이터
type Fundamentals struct {
	StockCode       string    `json:"stock_code"`
	ReportDate      time.Time `json:"report_date"`
	PER             float64   `json:"per"`
	PBR             float64   `json:"pbr"`
	ROE             float64   `json:"roe"`
	DebtRatio       float64   `json:"debt_ratio"`
	Revenue         int64     `json:"revenue"`
	OperatingProfit int64     `json:"operating_profit"`
	NetProfit       int64     `json:"net_profit"`
}

// Disclosure DART 공시
type Disclosure struct {
	ID          int64     `json:"id"`
	StockCode   string    `json:"stock_code"`
	DisclosedAt time.Time `json:"disclosed_at"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
}

// FetchResult 수집 결과
type FetchResult struct {
	Source      string    `json:"source"` // naver, dart, krx, kis
	Target      string    `json:"target"` // prices, flow, fundamentals, disclosures
	Count       int       `json:"count"`
	Duration    int64     `json:"duration_ms"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	CompletedAt time.Time `json:"completed_at"`
}
```

### Step 2: 리포지토리 인터페이스

**파일**: `internal/domain/fetcher/repository.go`

```go
package fetcher

import (
	"context"
	"time"
)

// StockRepository 종목 저장소
type StockRepository interface {
	// 종목 저장 (upsert)
	UpsertStock(ctx context.Context, stock *Stock) error
	UpsertStocks(ctx context.Context, stocks []*Stock) error

	// 종목 조회
	GetStock(ctx context.Context, code string) (*Stock, error)
	GetStocksByMarket(ctx context.Context, market string) ([]*Stock, error)
	GetActiveStocks(ctx context.Context) ([]*Stock, error)
}

// PriceRepository 가격 저장소
type PriceRepository interface {
	// 가격 저장 (upsert)
	UpsertPrice(ctx context.Context, price *DailyPrice) error
	UpsertPrices(ctx context.Context, prices []*DailyPrice) error

	// 가격 조회
	GetPrice(ctx context.Context, stockCode string, date time.Time) (*DailyPrice, error)
	GetPriceRange(ctx context.Context, stockCode string, from, to time.Time) ([]*DailyPrice, error)
	GetLatestPrice(ctx context.Context, stockCode string) (*DailyPrice, error)
}

// FlowRepository 수급 저장소
type FlowRepository interface {
	// 수급 저장 (upsert)
	UpsertFlow(ctx context.Context, flow *InvestorFlow) error
	UpsertFlows(ctx context.Context, flows []*InvestorFlow) error

	// 수급 조회
	GetFlowRange(ctx context.Context, stockCode string, from, to time.Time) ([]*InvestorFlow, error)
}

// FundamentalsRepository 재무 저장소
type FundamentalsRepository interface {
	// 재무 저장 (upsert)
	UpsertFundamentals(ctx context.Context, fund *Fundamentals) error

	// 재무 조회
	GetLatestFundamentals(ctx context.Context, stockCode string) (*Fundamentals, error)
}

// DisclosureRepository 공시 저장소
type DisclosureRepository interface {
	// 공시 저장
	SaveDisclosure(ctx context.Context, disc *Disclosure) error
	SaveDisclosures(ctx context.Context, discs []*Disclosure) error

	// 공시 조회
	GetDisclosures(ctx context.Context, stockCode string, from, to time.Time) ([]*Disclosure, error)
}
```

### Step 3: 외부 API 클라이언트

**파일**: `internal/infra/external/naver/client.go`

```go
package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

const (
	baseURL = "https://finance.naver.com"
)

// Client 네이버 금융 클라이언트
type Client struct {
	httpClient *http.Client
}

// NewClient 클라이언트 생성
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchDailyPrices 일봉 데이터 수집
func (c *Client) FetchDailyPrices(ctx context.Context, stockCode string, days int) ([]*fetcher.DailyPrice, error) {
	url := fmt.Sprintf("%s/item/sise_day.naver?code=%s&page=1", baseURL, stockCode)

	// HTTP 요청
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// HTML 파싱 및 데이터 추출
	// TODO: 실제 파싱 로직 구현

	return nil, nil
}

// FetchInvestorFlow 투자자별 수급 수집
func (c *Client) FetchInvestorFlow(ctx context.Context, stockCode string, days int) ([]*fetcher.InvestorFlow, error) {
	// TODO: 구현
	return nil, nil
}
```

**파일**: `internal/infra/external/dart/client.go`

```go
package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

const (
	baseURL = "https://opendart.fss.or.kr/api"
)

// Client DART 클라이언트
type Client struct {
	httpClient *http.Client
	apiKey     string
}

// NewClient 클라이언트 생성
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// FetchDisclosures 공시 수집
func (c *Client) FetchDisclosures(ctx context.Context, corpCode string, from, to time.Time) ([]*fetcher.Disclosure, error) {
	url := fmt.Sprintf("%s/list.json?crtfc_key=%s&corp_code=%s&bgn_de=%s&end_de=%s",
		baseURL, c.apiKey, corpCode,
		from.Format("20060102"), to.Format("20060102"))

	// TODO: 구현
	return nil, nil
}

// FetchFundamentals 재무 데이터 수집
func (c *Client) FetchFundamentals(ctx context.Context, corpCode string, year int, quarter int) (*fetcher.Fundamentals, error) {
	// TODO: 구현
	return nil, nil
}
```

### Step 4: 서비스 구현

**파일**: `internal/service/fetcher/service.go`

```go
package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/infra/external/dart"
	"github.com/wonny/aegis/v14/internal/infra/external/naver"
)

// Service Fetcher 서비스
type Service struct {
	// Repositories
	stockRepo  fetcher.StockRepository
	priceRepo  fetcher.PriceRepository
	flowRepo   fetcher.FlowRepository
	fundRepo   fetcher.FundamentalsRepository
	discRepo   fetcher.DisclosureRepository

	// External clients
	naverClient *naver.Client
	dartClient  *dart.Client
}

// NewService 서비스 생성
func NewService(
	stockRepo fetcher.StockRepository,
	priceRepo fetcher.PriceRepository,
	flowRepo fetcher.FlowRepository,
	fundRepo fetcher.FundamentalsRepository,
	discRepo fetcher.DisclosureRepository,
	dartAPIKey string,
) *Service {
	return &Service{
		stockRepo:   stockRepo,
		priceRepo:   priceRepo,
		flowRepo:    flowRepo,
		fundRepo:    fundRepo,
		discRepo:    discRepo,
		naverClient: naver.NewClient(),
		dartClient:  dart.NewClient(dartAPIKey),
	}
}

// CollectAll 전체 데이터 수집
func (s *Service) CollectAll(ctx context.Context) ([]*fetcher.FetchResult, error) {
	var results []*fetcher.FetchResult

	// 1. 가격 수집
	priceResult := s.collectPrices(ctx)
	results = append(results, priceResult)

	// 2. 수급 수집
	flowResult := s.collectFlow(ctx)
	results = append(results, flowResult)

	// 3. 재무 수집
	fundResult := s.collectFundamentals(ctx)
	results = append(results, fundResult)

	// 4. 공시 수집
	discResult := s.collectDisclosures(ctx)
	results = append(results, discResult)

	return results, nil
}

// collectPrices 가격 수집
func (s *Service) collectPrices(ctx context.Context) *fetcher.FetchResult {
	start := time.Now()
	result := &fetcher.FetchResult{
		Source: "naver",
		Target: "prices",
	}

	// 활성 종목 조회
	stocks, err := s.stockRepo.GetActiveStocks(ctx)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result
	}

	count := 0
	for _, stock := range stocks {
		prices, err := s.naverClient.FetchDailyPrices(ctx, stock.Code, 5)
		if err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to fetch prices")
			continue
		}

		if err := s.priceRepo.UpsertPrices(ctx, prices); err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to save prices")
			continue
		}

		count += len(prices)
	}

	result.Count = count
	result.Duration = time.Since(start).Milliseconds()
	result.Success = true
	result.CompletedAt = time.Now()

	return result
}

// collectFlow 수급 수집
func (s *Service) collectFlow(ctx context.Context) *fetcher.FetchResult {
	// TODO: 구현
	return &fetcher.FetchResult{
		Source:      "naver",
		Target:      "flow",
		Success:     true,
		CompletedAt: time.Now(),
	}
}

// collectFundamentals 재무 수집
func (s *Service) collectFundamentals(ctx context.Context) *fetcher.FetchResult {
	// TODO: 구현
	return &fetcher.FetchResult{
		Source:      "dart",
		Target:      "fundamentals",
		Success:     true,
		CompletedAt: time.Now(),
	}
}

// collectDisclosures 공시 수집
func (s *Service) collectDisclosures(ctx context.Context) *fetcher.FetchResult {
	// TODO: 구현
	return &fetcher.FetchResult{
		Source:      "dart",
		Target:      "disclosures",
		Success:     true,
		CompletedAt: time.Now(),
	}
}
```

### Step 5: PostgreSQL 리포지토리 구현

**파일**: `internal/infrastructure/postgres/fetcher/stock_repository.go`

```go
package fetcher

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

// StockRepository PostgreSQL 종목 저장소
type StockRepository struct {
	pool *pgxpool.Pool
}

// NewStockRepository 저장소 생성
func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

// UpsertStock 종목 저장 (upsert)
func (r *StockRepository) UpsertStock(ctx context.Context, stock *fetcher.Stock) error {
	query := `
		INSERT INTO data.stocks (code, name, market, sector, listing_date, delisting_date, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			market = EXCLUDED.market,
			sector = EXCLUDED.sector,
			delisting_date = EXCLUDED.delisting_date,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		stock.Code, stock.Name, stock.Market, stock.Sector,
		stock.ListingDate, stock.DelistingDate, stock.Status,
	)
	if err != nil {
		return fmt.Errorf("upsert stock: %w", err)
	}

	return nil
}

// UpsertStocks 종목 일괄 저장
func (r *StockRepository) UpsertStocks(ctx context.Context, stocks []*fetcher.Stock) error {
	for _, stock := range stocks {
		if err := r.UpsertStock(ctx, stock); err != nil {
			return err
		}
	}
	return nil
}

// GetStock 종목 조회
func (r *StockRepository) GetStock(ctx context.Context, code string) (*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE code = $1
	`

	var stock fetcher.Stock
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
		&stock.ListingDate, &stock.DelistingDate, &stock.Status,
		&stock.CreatedAt, &stock.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}

	return &stock, nil
}

// GetStocksByMarket 시장별 종목 조회
func (r *StockRepository) GetStocksByMarket(ctx context.Context, market string) ([]*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE market = $1 AND status = 'active'
		ORDER BY code
	`

	rows, err := r.pool.Query(ctx, query, market)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*fetcher.Stock
	for rows.Next() {
		var stock fetcher.Stock
		if err := rows.Scan(
			&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
			&stock.ListingDate, &stock.DelistingDate, &stock.Status,
			&stock.CreatedAt, &stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, &stock)
	}

	return stocks, nil
}

// GetActiveStocks 활성 종목 조회
func (r *StockRepository) GetActiveStocks(ctx context.Context) ([]*fetcher.Stock, error) {
	query := `
		SELECT code, name, market, sector, listing_date, delisting_date, status, created_at, updated_at
		FROM data.stocks
		WHERE status = 'active'
		ORDER BY code
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query stocks: %w", err)
	}
	defer rows.Close()

	var stocks []*fetcher.Stock
	for rows.Next() {
		var stock fetcher.Stock
		if err := rows.Scan(
			&stock.Code, &stock.Name, &stock.Market, &stock.Sector,
			&stock.ListingDate, &stock.DelistingDate, &stock.Status,
			&stock.CreatedAt, &stock.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		stocks = append(stocks, &stock)
	}

	return stocks, nil
}
```

### Step 6: API 핸들러

**파일**: `internal/api/handlers/fetcher/handler.go`

```go
package fetcher

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wonny/aegis/v14/internal/api/response"
	"github.com/wonny/aegis/v14/internal/service/fetcher"
)

// Handler Fetcher API 핸들러
type Handler struct {
	service *fetcher.Service
}

// NewHandler 핸들러 생성
func NewHandler(service *fetcher.Service) *Handler {
	return &Handler{service: service}
}

// Routes 라우트 등록
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/collect/all", h.CollectAll)
	r.Post("/collect/prices", h.CollectPrices)
	r.Post("/collect/flow", h.CollectFlow)
	r.Post("/collect/fundamentals", h.CollectFundamentals)
	r.Post("/collect/disclosures", h.CollectDisclosures)

	return r
}

// CollectAll 전체 수집
func (h *Handler) CollectAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.CollectAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"results": results,
	})
}

// CollectPrices 가격 수집
func (h *Handler) CollectPrices(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Price collection started",
	})
}

// CollectFlow 수급 수집
func (h *Handler) CollectFlow(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Flow collection started",
	})
}

// CollectFundamentals 재무 수집
func (h *Handler) CollectFundamentals(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Fundamentals collection started",
	})
}

// CollectDisclosures 공시 수집
func (h *Handler) CollectDisclosures(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Disclosures collection started",
	})
}
```

---

## 🟡 P1: Signals 모듈 구현 (6팩터)

### v14에 이미 존재하는 모델 확장

기존 `internal/domain/signals/model.go`에 v13의 6팩터 점수를 추가합니다.

**수정 필요 사항**:

1. `SignalBreakdown`에 `Flow`, `Event` 팩터 추가
2. 각 팩터별 상세 계산 로직 추가

**파일**: `internal/service/signals/factor_calculator.go` (신규)

```go
package signals

import (
	"context"
	"math"

	"github.com/wonny/aegis/v14/internal/domain/fetcher"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// FactorCalculator 팩터 계산기
type FactorCalculator struct {
	priceRepo fetcher.PriceRepository
	flowRepo  fetcher.FlowRepository
	fundRepo  fetcher.FundamentalsRepository
}

// NewFactorCalculator 팩터 계산기 생성
func NewFactorCalculator(
	priceRepo fetcher.PriceRepository,
	flowRepo fetcher.FlowRepository,
	fundRepo fetcher.FundamentalsRepository,
) *FactorCalculator {
	return &FactorCalculator{
		priceRepo: priceRepo,
		flowRepo:  flowRepo,
		fundRepo:  fundRepo,
	}
}

// CalculateMomentum 모멘텀 팩터 계산
// v13 로직: Return1M(40%) + Return3M(40%) + VolumeRate(20%)
func (c *FactorCalculator) CalculateMomentum(ctx context.Context, symbol string) (*signals.MomentumFactors, float64, error) {
	// 가격 데이터 조회 (60일)
	prices, err := c.priceRepo.GetPriceRange(ctx, symbol, /* 60일 전 */, /* 오늘 */)
	if err != nil {
		return nil, 0, err
	}

	if len(prices) < 60 {
		return nil, 0, nil
	}

	// 수익률 계산
	return1M := calculateReturn(prices, 20)
	return3M := calculateReturn(prices, 60)
	volumeRate := calculateVolumeGrowth(prices, 20)

	// 가중 합산
	rawScore := return1M*0.4 + return3M*0.4 + volumeRate*0.2

	// tanh 정규화 (-1 ~ 1)
	normalizedScore := math.Tanh(rawScore * 2)

	factors := &signals.MomentumFactors{
		Symbol:    symbol,
		Return5D:  calculateReturn(prices, 5),
		Return20D: return1M,
		Return60D: return3M,
		VolumeGrowth: volumeRate,
	}

	return factors, normalizedScore, nil
}

// CalculateTechnical 기술적 팩터 계산
// v13 로직: RSI(40%) + MACD(40%) + MA20Cross(20%)
func (c *FactorCalculator) CalculateTechnical(ctx context.Context, symbol string) (*signals.TechnicalFactors, float64, error) {
	prices, err := c.priceRepo.GetPriceRange(ctx, symbol, /* 120일 전 */, /* 오늘 */)
	if err != nil {
		return nil, 0, err
	}

	if len(prices) < 120 {
		return nil, 0, nil
	}

	rsi := calculateRSI(prices, 14)
	macd, macdSignal := calculateMACD(prices)
	ma20Cross := calculateMA20Cross(prices)

	// RSI 점수화
	rsiScore := 0.0
	if rsi < 30 {
		rsiScore = (30 - rsi) / 30
	} else if rsi > 70 {
		rsiScore = (70 - rsi) / 30
	} else {
		rsiScore = (50 - rsi) / 20
	}

	// MACD 점수화
	macdScore := math.Tanh(macd / 500)

	// 가중 합산
	rawScore := rsiScore*0.4 + macdScore*0.4 + float64(ma20Cross)*0.2

	// clamp
	if rawScore > 1.0 {
		rawScore = 1.0
	} else if rawScore < -1.0 {
		rawScore = -1.0
	}

	factors := &signals.TechnicalFactors{
		Symbol:     symbol,
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: macdSignal,
	}

	return factors, rawScore, nil
}

// CalculateValue 가치 팩터 계산
// v13 로직: PER(50%) + PBR(30%) + PSR(20%)
func (c *FactorCalculator) CalculateValue(ctx context.Context, symbol string) (*signals.ValueFactors, float64, error) {
	fund, err := c.fundRepo.GetLatestFundamentals(ctx, symbol)
	if err != nil {
		return nil, 0, err
	}

	// PER 점수화 (10 기준)
	perScore := 0.0
	if fund.PER > 0 {
		perScore = (15 - fund.PER) / 15
		if perScore > 1.0 {
			perScore = 1.0
		} else if perScore < -1.0 {
			perScore = -1.0
		}
	}

	// PBR 점수화 (1.0 기준)
	pbrScore := 0.0
	if fund.PBR > 0 {
		pbrScore = (1.5 - fund.PBR) / 1.5
		if pbrScore > 1.0 {
			pbrScore = 1.0
		} else if pbrScore < -1.0 {
			pbrScore = -1.0
		}
	}

	// 가중 합산
	rawScore := perScore*0.5 + pbrScore*0.3 // PSR 데이터 없으면 생략
	score := math.Tanh(rawScore * 1.5)

	factors := &signals.ValueFactors{
		Symbol: symbol,
		PER:    fund.PER,
		PBR:    fund.PBR,
	}

	return factors, score, nil
}

// CalculateQuality 퀄리티 팩터 계산
// v13 로직: ROE(60%) + DebtRatio(40%)
func (c *FactorCalculator) CalculateQuality(ctx context.Context, symbol string) (*signals.QualityFactors, float64, error) {
	fund, err := c.fundRepo.GetLatestFundamentals(ctx, symbol)
	if err != nil {
		return nil, 0, err
	}

	// ROE 점수화
	roeScore := (fund.ROE - 10) / 15
	if roeScore > 1.0 {
		roeScore = 1.0
	} else if roeScore < -1.0 {
		roeScore = -1.0
	}

	// DebtRatio 점수화
	debtScore := (100 - fund.DebtRatio) / 100
	if debtScore > 1.0 {
		debtScore = 1.0
	} else if debtScore < -1.0 {
		debtScore = -1.0
	}

	// 가중 합산
	rawScore := roeScore*0.6 + debtScore*0.4
	score := math.Tanh(rawScore * 1.5)

	factors := &signals.QualityFactors{
		Symbol:    symbol,
		ROE:       fund.ROE,
		DebtRatio: fund.DebtRatio,
	}

	return factors, score, nil
}

// CalculateFlow 수급 팩터 계산
// v13 로직: 외국인(60%) + 기관(40%), 5D(70%) + 20D(30%)
func (c *FactorCalculator) CalculateFlow(ctx context.Context, symbol string) (float64, error) {
	flows, err := c.flowRepo.GetFlowRange(ctx, symbol, /* 20일 전 */, /* 오늘 */)
	if err != nil {
		return 0, err
	}

	if len(flows) < 20 {
		return 0, nil
	}

	// 5일/20일 누적
	var foreignNet5D, foreignNet20D, instNet5D, instNet20D int64
	for i, flow := range flows {
		if i < 5 {
			foreignNet5D += flow.ForeignNetQty
			instNet5D += flow.InstNetQty
		}
		foreignNet20D += flow.ForeignNetQty
		instNet20D += flow.InstNetQty
	}

	// 정규화 (기준: 5D=50만주, 20D=200만주)
	foreignScore5D := math.Tanh(float64(foreignNet5D) / 500_000)
	foreignScore20D := math.Tanh(float64(foreignNet20D) / 2_000_000)
	instScore5D := math.Tanh(float64(instNet5D) / 500_000)
	instScore20D := math.Tanh(float64(instNet20D) / 2_000_000)

	// 가중 합산
	foreignScore := foreignScore5D*0.7 + foreignScore20D*0.3
	instScore := instScore5D*0.7 + instScore20D*0.3

	score := foreignScore*0.6 + instScore*0.4

	return score, nil
}

// 헬퍼 함수들
func calculateReturn(prices []*fetcher.DailyPrice, days int) float64 {
	if len(prices) < days+1 {
		return 0
	}
	current := prices[0].ClosePrice
	past := prices[days].ClosePrice
	if past == 0 {
		return 0
	}
	return (current - past) / past
}

func calculateVolumeGrowth(prices []*fetcher.DailyPrice, days int) float64 {
	if len(prices) < days*2 {
		return 0
	}

	var recentSum, pastSum int64
	for i := 0; i < days; i++ {
		recentSum += prices[i].Volume
		pastSum += prices[days+i].Volume
	}

	if pastSum == 0 {
		return 0
	}

	recentAvg := float64(recentSum) / float64(days)
	pastAvg := float64(pastSum) / float64(days)

	return (recentAvg - pastAvg) / pastAvg
}

func calculateRSI(prices []*fetcher.DailyPrice, period int) float64 {
	if len(prices) < period+1 {
		return 50.0
	}

	var gains, losses float64
	for i := 0; i < period; i++ {
		change := prices[i].ClosePrice - prices[i+1].ClosePrice
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	if losses == 0 {
		return 100.0
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	rs := avgGain / avgLoss

	return 100 - (100 / (1 + rs))
}

func calculateMACD(prices []*fetcher.DailyPrice) (float64, float64) {
	if len(prices) < 26 {
		return 0, 0
	}

	ema12 := calculateEMA(prices, 12)
	ema26 := calculateEMA(prices, 26)
	macd := ema12 - ema26

	return macd, macd // signal 간단화
}

func calculateEMA(prices []*fetcher.DailyPrice, period int) float64 {
	if len(prices) < period {
		return 0
	}

	var sum float64
	for i := 0; i < period; i++ {
		sum += prices[len(prices)-period+i].ClosePrice
	}
	sma := sum / float64(period)

	multiplier := 2.0 / (float64(period) + 1.0)
	ema := sma

	for i := len(prices) - period - 1; i >= 0; i-- {
		ema = prices[i].ClosePrice*multiplier + ema*(1-multiplier)
	}

	return ema
}

func calculateMA20Cross(prices []*fetcher.DailyPrice) int {
	if len(prices) < 20 {
		return 0
	}

	var sum float64
	for i := 0; i < 20; i++ {
		sum += prices[i].ClosePrice
	}
	ma20 := sum / 20.0

	currentPrice := prices[0].ClosePrice
	priceDiff := (currentPrice - ma20) / ma20

	if priceDiff > 0.02 {
		return 1 // Golden Cross
	} else if priceDiff < -0.02 {
		return -1 // Death Cross
	}
	return 0
}
```

---

## 🟢 P2: Audit 모듈 구현

### Step 1: 도메인 모델

**파일**: `internal/domain/audit/model.go`

```go
package audit

import "time"

// PerformanceReport 성과 보고서
type PerformanceReport struct {
	Period    string    `json:"period"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// 수익률
	TotalReturn  float64 `json:"total_return"`
	AnnualReturn float64 `json:"annual_return"`

	// 리스크 지표
	Volatility  float64 `json:"volatility"`
	Sharpe      float64 `json:"sharpe"`
	Sortino     float64 `json:"sortino"`
	MaxDrawdown float64 `json:"max_drawdown"`

	// 트레이딩 지표
	WinRate      float64 `json:"win_rate"`
	AvgWin       float64 `json:"avg_win"`
	AvgLoss      float64 `json:"avg_loss"`
	ProfitFactor float64 `json:"profit_factor"`
	TotalTrades  int     `json:"total_trades"`

	// 비교
	Benchmark float64 `json:"benchmark"`
	Alpha     float64 `json:"alpha"`
	Beta      float64 `json:"beta"`
}

// DailyPnL 일별 손익
type DailyPnL struct {
	Date             time.Time `json:"date"`
	RealizedPnL      int64     `json:"realized_pnl"`
	UnrealizedPnL    int64     `json:"unrealized_pnl"`
	TotalPnL         int64     `json:"total_pnl"`
	DailyReturn      float64   `json:"daily_return"`
	CumulativeReturn float64   `json:"cumulative_return"`
	PortfolioValue   int64     `json:"portfolio_value"`
	CashBalance      int64     `json:"cash_balance"`
}

// Trade 거래 기록
type Trade struct {
	Symbol     string    `json:"symbol"`
	EntryDate  time.Time `json:"entry_date"`
	ExitDate   time.Time `json:"exit_date"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Quantity   int64     `json:"quantity"`
	PnL        float64   `json:"pnl"`
	PnLPct     float64   `json:"pnl_pct"`
}
```

### Step 2: 서비스 구현

**파일**: `internal/service/audit/service.go`

```go
package audit

import (
	"context"
	"math"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/audit"
)

const riskFreeRate = 0.03 // 무위험 수익률 3%

// Service Audit 서비스
type Service struct {
	auditRepo audit.Repository
}

// NewService 서비스 생성
func NewService(auditRepo audit.Repository) *Service {
	return &Service{auditRepo: auditRepo}
}

// Analyze 성과 분석
func (s *Service) Analyze(ctx context.Context, period string) (*audit.PerformanceReport, error) {
	startDate, endDate := parsePeriod(period)

	// 일별 수익률 조회
	dailyReturns, err := s.auditRepo.GetDailyReturns(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(dailyReturns) == 0 {
		return &audit.PerformanceReport{
			Period:    period,
			StartDate: startDate,
			EndDate:   endDate,
		}, nil
	}

	// 수익률 계산
	totalReturn := calculateTotalReturn(dailyReturns)
	annualReturn := annualize(totalReturn, len(dailyReturns))

	// 리스크 지표
	volatility := calculateVolatility(dailyReturns)
	sharpe := calculateSharpe(annualReturn, volatility)
	sortino := calculateSortino(dailyReturns)
	maxDD := calculateMaxDrawdown(dailyReturns)

	// 거래 지표
	trades, _ := s.auditRepo.GetTrades(ctx, startDate, endDate)
	winRate := calculateWinRate(trades)
	avgWin, avgLoss := calculateAvgWinLoss(trades)
	profitFactor := calculateProfitFactor(trades)

	// 벤치마크
	benchmark := 0.05 // TODO: 실제 벤치마크 조회

	return &audit.PerformanceReport{
		Period:       period,
		StartDate:    startDate,
		EndDate:      endDate,
		TotalReturn:  totalReturn,
		AnnualReturn: annualReturn,
		Volatility:   volatility,
		Sharpe:       sharpe,
		Sortino:      sortino,
		MaxDrawdown:  maxDD,
		WinRate:      winRate,
		AvgWin:       avgWin,
		AvgLoss:      avgLoss,
		ProfitFactor: profitFactor,
		TotalTrades:  len(trades),
		Benchmark:    benchmark,
		Alpha:        totalReturn - benchmark,
		Beta:         1.0,
	}, nil
}

func parsePeriod(period string) (time.Time, time.Time) {
	endDate := time.Now()
	switch period {
	case "1M":
		return endDate.AddDate(0, -1, 0), endDate
	case "3M":
		return endDate.AddDate(0, -3, 0), endDate
	case "6M":
		return endDate.AddDate(0, -6, 0), endDate
	case "1Y":
		return endDate.AddDate(-1, 0, 0), endDate
	case "YTD":
		return time.Date(endDate.Year(), 1, 1, 0, 0, 0, 0, endDate.Location()), endDate
	default:
		return endDate.AddDate(0, -1, 0), endDate
	}
}

func calculateTotalReturn(returns []float64) float64 {
	cum := 1.0
	for _, r := range returns {
		cum *= (1.0 + r)
	}
	return cum - 1.0
}

func annualize(totalReturn float64, days int) float64 {
	if days == 0 {
		return 0
	}
	return math.Pow(1.0+totalReturn, 252.0/float64(days)) - 1.0
}

func calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance) * math.Sqrt(252)
}

func calculateSharpe(annualReturn, volatility float64) float64 {
	if volatility == 0 {
		return 0
	}
	return (annualReturn - riskFreeRate) / volatility
}

func calculateSortino(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	var sumSquaredNeg float64
	var countNeg int
	for _, r := range returns {
		if r < 0 {
			sumSquaredNeg += r * r
			countNeg++
		}
	}

	if countNeg == 0 {
		return 0
	}

	downsideVol := math.Sqrt(sumSquaredNeg/float64(countNeg)) * math.Sqrt(252)
	if downsideVol == 0 {
		return 0
	}

	totalReturn := calculateTotalReturn(returns)
	annualReturn := annualize(totalReturn, len(returns))

	return (annualReturn - riskFreeRate) / downsideVol
}

func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	cumValue := 1.0
	peak := 1.0
	maxDD := 0.0

	for _, r := range returns {
		cumValue *= (1.0 + r)
		if cumValue > peak {
			peak = cumValue
		}
		dd := (cumValue - peak) / peak
		if dd < maxDD {
			maxDD = dd
		}
	}

	return maxDD
}

func calculateWinRate(trades []*audit.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(trades))
}

func calculateAvgWinLoss(trades []*audit.Trade) (float64, float64) {
	if len(trades) == 0 {
		return 0, 0
	}

	var sumWin, sumLoss float64
	var countWin, countLoss int

	for _, t := range trades {
		if t.PnL > 0 {
			sumWin += t.PnL
			countWin++
		} else if t.PnL < 0 {
			sumLoss += t.PnL
			countLoss++
		}
	}

	avgWin := 0.0
	if countWin > 0 {
		avgWin = sumWin / float64(countWin)
	}

	avgLoss := 0.0
	if countLoss > 0 {
		avgLoss = sumLoss / float64(countLoss)
	}

	return avgWin, avgLoss
}

func calculateProfitFactor(trades []*audit.Trade) float64 {
	var totalWin, totalLoss float64

	for _, t := range trades {
		if t.PnL > 0 {
			totalWin += t.PnL
		} else if t.PnL < 0 {
			totalLoss += math.Abs(t.PnL)
		}
	}

	if totalLoss == 0 {
		return 0
	}

	return totalWin / totalLoss
}
```

---

## 📁 파일 생성 체크리스트

### Fetcher 모듈
- [ ] `internal/domain/fetcher/model.go`
- [ ] `internal/domain/fetcher/repository.go`
- [ ] `internal/domain/fetcher/errors.go`
- [ ] `internal/service/fetcher/service.go`
- [ ] `internal/infrastructure/postgres/fetcher/stock_repository.go`
- [ ] `internal/infrastructure/postgres/fetcher/price_repository.go`
- [ ] `internal/infrastructure/postgres/fetcher/flow_repository.go`
- [ ] `internal/infrastructure/postgres/fetcher/fundamentals_repository.go`
- [ ] `internal/infrastructure/postgres/fetcher/disclosure_repository.go`
- [ ] `internal/infra/external/naver/client.go`
- [ ] `internal/infra/external/dart/client.go`
- [ ] `internal/api/handlers/fetcher/handler.go`
- [ ] `internal/api/routes/fetcher_routes.go`

### Signals 모듈 (확장)
- [ ] `internal/service/signals/factor_calculator.go`
- [ ] `internal/infrastructure/postgres/signals/factor_repository.go`

### Audit 모듈
- [ ] `internal/domain/audit/model.go`
- [ ] `internal/domain/audit/repository.go`
- [ ] `internal/domain/audit/errors.go`
- [ ] `internal/service/audit/service.go`
- [ ] `internal/infrastructure/postgres/audit/repository.go`
- [ ] `internal/api/handlers/audit/handler.go`
- [ ] `internal/api/routes/audit_routes.go`

---

## 🗄️ DB 마이그레이션

마이그레이션 파일을 `backend/migrations/` 에 생성하세요.

```sql
-- migrations/100_create_data_schema.sql
CREATE SCHEMA IF NOT EXISTS data;

-- migrations/101_create_signals_schema.sql
CREATE SCHEMA IF NOT EXISTS signals;

-- migrations/102_create_audit_schema.sql
CREATE SCHEMA IF NOT EXISTS audit;

-- migrations/103_create_data_tables.sql
-- (상세 SQL은 docs/database/schema.md 참조)

-- migrations/104_create_signals_tables.sql
-- migrations/105_create_audit_tables.sql
```

---

## ✅ 구현 순서

1. **DB 스키마 생성** → 마이그레이션 실행
2. **Fetcher 모듈** → 데이터 수집 기반
3. **Signals 팩터 계산기** → 6팩터 점수 계산
4. **Audit 모듈** → 성과 분석

각 단계별로 테스트 코드도 함께 작성하세요.

---

**Version**: v14.0.0
**Last Updated**: 2026-01-17
