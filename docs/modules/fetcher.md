# Fetcher 모듈 설계

> 외부 데이터 소스에서 시장 데이터를 수집하는 모듈

**Version**: 1.1.0 (v14 구현)
**Status**: ✅ 구현 완료
**Last Updated**: 2026-01-17

---

## 📋 개요

### 책임 (Responsibility)

외부 API에서 시장 데이터를 수집하여 데이터베이스에 저장합니다.

### 핵심 기능

1. **가격 데이터 수집**: 일봉, 거래량, 거래대금
2. **투자자 수급 수집**: 외국인/기관/개인 순매수
3. **시가총액 수집**: 시총, 상장주식수, 유동주식수
4. **공시 데이터 수집**: DART 공시
5. **재무 데이터 수집**: PER, PBR, ROE 등 기본 지표

### 구현 파일 위치

```
backend/internal/
├── domain/fetcher/
│   ├── model.go           # 도메인 모델 (Stock, DailyPrice, InvestorFlow, etc.)
│   ├── repository.go      # Repository/Client 인터페이스
│   └── errors.go          # 도메인 에러
├── service/fetcher/
│   └── service.go         # 서비스 (오케스트레이션, 스케줄링)
├── infra/external/
│   ├── naver/client.go    # Naver Finance 스크래핑 클라이언트
│   └── dart/client.go     # DART OpenAPI 클라이언트
├── infra/database/postgres/fetcher/
│   ├── stock_repository.go
│   ├── price_repository.go
│   ├── flow_repository.go
│   ├── fundamentals_repository.go
│   ├── marketcap_repository.go
│   └── disclosure_repository.go
└── api/
    ├── handlers/fetcher/handler.go
    └── routes/fetcher_routes.go
```

### 의존성

- `infra/database/postgres` (PostgreSQL Pool)
- `net/http` (HTTP 클라이언트)
- `github.com/PuerkitoBio/goquery` (HTML 파싱)
- 외부 API: Naver Finance, DART OpenAPI

---

## 🎯 설계 원칙

### 1. 데이터 소스별 분리

```
External Clients
├── NaverClient (가격, 수급, 시가총액, 재무)
└── DartClient (공시)
```

### 2. 멱등성 보장

같은 날짜 데이터를 여러 번 수집해도 중복 없이 UPSERT

### 3. 스케줄 기반 자동 수집

백그라운드에서 설정된 간격으로 데이터 자동 수집

### 4. 실패 격리

한 종목 실패가 전체 수집을 중단시키지 않음

---

## 🏗️ 구현 상세

### Domain Layer

#### domain/fetcher/model.go

```go
// Stock 종목 마스터 (data.stocks)
type Stock struct {
    Code          string     `json:"code" db:"code"`
    Name          string     `json:"name" db:"name"`
    Market        string     `json:"market" db:"market"`         // KOSPI, KOSDAQ
    Sector        *string    `json:"sector,omitempty" db:"sector"`
    ListingDate   *time.Time `json:"listing_date,omitempty" db:"listing_date"`
    DelistingDate *time.Time `json:"delisting_date,omitempty" db:"delisting_date"`
    Status        string     `json:"status" db:"status"`         // active, suspended, delisted
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// DailyPrice 일봉 데이터 (data.daily_prices)
type DailyPrice struct {
    StockCode    string    `json:"stock_code" db:"stock_code"`
    TradeDate    time.Time `json:"trade_date" db:"trade_date"`
    OpenPrice    int64     `json:"open_price" db:"open_price"`
    HighPrice    int64     `json:"high_price" db:"high_price"`
    LowPrice     int64     `json:"low_price" db:"low_price"`
    ClosePrice   int64     `json:"close_price" db:"close_price"`
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

// Fundamentals 재무 지표 (data.fundamentals)
type Fundamentals struct {
    StockCode       string    `json:"stock_code" db:"stock_code"`
    ReportDate      time.Time `json:"report_date" db:"report_date"`
    PER             *float64  `json:"per,omitempty" db:"per"`
    PBR             *float64  `json:"pbr,omitempty" db:"pbr"`
    PSR             *float64  `json:"psr,omitempty" db:"psr"`
    ROE             *float64  `json:"roe,omitempty" db:"roe"`
    DebtRatio       *float64  `json:"debt_ratio,omitempty" db:"debt_ratio"`
    Revenue         *int64    `json:"revenue,omitempty" db:"revenue"`
    OperatingProfit *int64    `json:"operating_profit,omitempty" db:"operating_profit"`
    NetProfit       *int64    `json:"net_profit,omitempty" db:"net_profit"`
    EPS             *int64    `json:"eps,omitempty" db:"eps"`
    BPS             *int64    `json:"bps,omitempty" db:"bps"`
    DPS             *int64    `json:"dps,omitempty" db:"dps"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// MarketCap 시가총액 (data.market_cap)
type MarketCap struct {
    StockCode   string    `json:"stock_code" db:"stock_code"`
    TradeDate   time.Time `json:"trade_date" db:"trade_date"`
    MarketCap   int64     `json:"market_cap" db:"market_cap"`
    SharesOut   int64     `json:"shares_out" db:"shares_out"`
    FloatShares *int64    `json:"float_shares,omitempty" db:"float_shares"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Disclosure 공시 (data.disclosures)
type Disclosure struct {
    ID          int64      `json:"id" db:"id"`
    StockCode   string     `json:"stock_code" db:"stock_code"`
    DisclosedAt time.Time  `json:"disclosed_at" db:"disclosed_at"`
    Title       string     `json:"title" db:"title"`
    Category    string     `json:"category" db:"category"`
    Subcategory *string    `json:"subcategory,omitempty" db:"subcategory"`
    Content     *string    `json:"content,omitempty" db:"content"`
    URL         *string    `json:"url,omitempty" db:"url"`
    DartRceptNo *string    `json:"dart_rcept_no,omitempty" db:"dart_rcept_no"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}
```

#### domain/fetcher/repository.go

```go
// StockRepository 종목 저장소 (data.stocks)
type StockRepository interface {
    Upsert(ctx context.Context, stock *Stock) error
    UpsertBatch(ctx context.Context, stocks []*Stock) (int, error)
    GetByCode(ctx context.Context, code string) (*Stock, error)
    GetByMarket(ctx context.Context, market string) ([]*Stock, error)
    GetActive(ctx context.Context) ([]*Stock, error)
    List(ctx context.Context, filter *StockFilter) ([]*Stock, error)
    Count(ctx context.Context, filter *StockFilter) (int, error)
}

// PriceRepository 가격 저장소 (data.daily_prices)
type PriceRepository interface {
    Upsert(ctx context.Context, price *DailyPrice) error
    UpsertBatch(ctx context.Context, prices []*DailyPrice) (int, error)
    GetByDate(ctx context.Context, stockCode string, date time.Time) (*DailyPrice, error)
    GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*DailyPrice, error)
    GetLatest(ctx context.Context, stockCode string) (*DailyPrice, error)
    GetLatestN(ctx context.Context, stockCode string, n int) ([]*DailyPrice, error)
}

// FlowRepository 수급 저장소 (data.investor_flow)
type FlowRepository interface {
    Upsert(ctx context.Context, flow *InvestorFlow) error
    UpsertBatch(ctx context.Context, flows []*InvestorFlow) (int, error)
    GetByDate(ctx context.Context, stockCode string, date time.Time) (*InvestorFlow, error)
    GetRange(ctx context.Context, stockCode string, from, to time.Time) ([]*InvestorFlow, error)
    GetLatest(ctx context.Context, stockCode string) (*InvestorFlow, error)
}

// NaverClient 네이버 금융 클라이언트
type NaverClient interface {
    FetchDailyPrices(ctx context.Context, stockCode string, days int) ([]*DailyPrice, error)
    FetchInvestorFlow(ctx context.Context, stockCode string, days int) ([]*InvestorFlow, error)
    FetchMarketCap(ctx context.Context, stockCode string) (*MarketCap, error)
    FetchStockInfo(ctx context.Context, stockCode string) (*Stock, error)
    FetchFundamentals(ctx context.Context, stockCode string) (*Fundamentals, error)
    FetchMarketCapRanking(ctx context.Context, market string, limit int) ([]*Stock, error)
}

// DartClient DART 공시 클라이언트
type DartClient interface {
    FetchDisclosures(ctx context.Context, corpCode string, from, to time.Time) ([]*Disclosure, error)
    FetchAllDisclosures(ctx context.Context, from, to time.Time) ([]*Disclosure, error)
    FetchFinancials(ctx context.Context, corpCode string, year int, reportCode string) (*Fundamentals, error)
    HealthCheck(ctx context.Context) error
}
```

### Service Layer

#### service/fetcher/service.go

```go
// CollectorType 수집기 타입
type CollectorType string

const (
    CollectorPrice      CollectorType = "price"
    CollectorFlow       CollectorType = "flow"
    CollectorFundament  CollectorType = "fundamental"
    CollectorMarketCap  CollectorType = "marketcap"
    CollectorDisclosure CollectorType = "disclosure"
)

// Config 서비스 설정
type Config struct {
    PriceInterval       time.Duration  // 가격 수집 간격 (기본: 1시간)
    FlowInterval        time.Duration  // 수급 수집 간격 (기본: 1시간)
    FundamentalInterval time.Duration  // 재무 수집 간격 (기본: 24시간)
    MarketCapInterval   time.Duration  // 시가총액 수집 간격 (기본: 6시간)
    DisclosureInterval  time.Duration  // 공시 수집 간격 (기본: 30분)
    BatchSize           int            // 배치 크기 (기본: 100)
    MaxRetries          int            // 최대 재시도 (기본: 3)
    RetryBackoff        time.Duration  // 재시도 대기 (기본: 5초)
    MaxConcurrent       int            // 최대 동시 수집 (기본: 5)
}

// Service Fetcher 서비스
type Service struct {
    ctx    context.Context
    config *Config

    // External Clients
    naverClient fetcher.NaverClient
    dartClient  fetcher.DartClient

    // Repositories
    stockRepo       fetcher.StockRepository
    priceRepo       fetcher.PriceRepository
    flowRepo        fetcher.FlowRepository
    fundamentalRepo fetcher.FundamentalsRepository
    marketCapRepo   fetcher.MarketCapRepository
    disclosureRepo  fetcher.DisclosureRepository
}

// 주요 메서드
func (s *Service) Start() error                                    // 백그라운드 수집 시작
func (s *Service) Stop() error                                     // 수집 중지
func (s *Service) CollectNow(ctx, collectorType) error             // 즉시 수집
func (s *Service) CollectStock(ctx, stockCode) (*FetchResult, error)// 특정 종목 수집
func (s *Service) RefreshStockMaster(ctx) error                    // 종목 마스터 갱신
```

---

## 📊 External API Clients

### Naver Finance Client (infra/external/naver/client.go)

네이버 금융 페이지를 스크래핑하여 데이터 수집

```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    timeout    time.Duration
    userAgent  string
}

// 수집 가능 데이터
- 일봉 가격 (FetchDailyPrices)
- 투자자 수급 (FetchInvestorFlow)
- 시가총액 (FetchMarketCap)
- 종목 정보 (FetchStockInfo)
- 재무 지표 (FetchFundamentals)
- 시가총액 순위 (FetchMarketCapRanking)
```

### DART Client (infra/external/dart/client.go)

DART OpenAPI를 통한 공시 데이터 수집

```go
type Client struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
}

// 수집 가능 데이터
- 전체 공시 (FetchAllDisclosures)
- 종목별 공시 (FetchDisclosures)
- 재무제표 (FetchFinancials)
```

---

## 🗄️ Database Schema

### data.stocks
```sql
CREATE TABLE IF NOT EXISTS data.stocks (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    market VARCHAR(20) NOT NULL,
    sector VARCHAR(100),
    listing_date DATE,
    delisting_date DATE,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### data.daily_prices
```sql
CREATE TABLE IF NOT EXISTS data.daily_prices (
    stock_code VARCHAR(20) NOT NULL,
    trade_date DATE NOT NULL,
    open_price BIGINT,
    high_price BIGINT,
    low_price BIGINT,
    close_price BIGINT NOT NULL,
    volume BIGINT DEFAULT 0,
    trading_value BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);
```

### data.investor_flow
```sql
CREATE TABLE IF NOT EXISTS data.investor_flow (
    stock_code VARCHAR(20) NOT NULL,
    trade_date DATE NOT NULL,
    foreign_net_qty BIGINT DEFAULT 0,
    foreign_net_value BIGINT DEFAULT 0,
    inst_net_qty BIGINT DEFAULT 0,
    inst_net_value BIGINT DEFAULT 0,
    indiv_net_qty BIGINT DEFAULT 0,
    indiv_net_value BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);
```

### data.fundamentals
```sql
CREATE TABLE IF NOT EXISTS data.fundamentals (
    stock_code VARCHAR(20) NOT NULL,
    report_date DATE NOT NULL,
    per DECIMAL(10,2),
    pbr DECIMAL(10,2),
    psr DECIMAL(10,2),
    roe DECIMAL(10,2),
    debt_ratio DECIMAL(10,2),
    revenue BIGINT,
    operating_profit BIGINT,
    net_profit BIGINT,
    eps BIGINT,
    bps BIGINT,
    dps BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (stock_code, report_date)
);
```

### data.market_cap
```sql
CREATE TABLE IF NOT EXISTS data.market_cap (
    stock_code VARCHAR(20) NOT NULL,
    trade_date DATE NOT NULL,
    market_cap BIGINT NOT NULL,
    shares_out BIGINT NOT NULL,
    float_shares BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);
```

### data.disclosures
```sql
CREATE TABLE IF NOT EXISTS data.disclosures (
    id BIGSERIAL PRIMARY KEY,
    stock_code VARCHAR(20) NOT NULL,
    disclosed_at TIMESTAMP NOT NULL,
    title VARCHAR(500) NOT NULL,
    category VARCHAR(100) NOT NULL,
    subcategory VARCHAR(100),
    content TEXT,
    url TEXT,
    dart_rcept_no VARCHAR(50) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🔌 API Endpoints

### Stock Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fetcher/stocks` | 종목 목록 조회 |
| GET | `/api/v1/fetcher/stocks/{code}` | 종목 상세 조회 |
| GET | `/api/v1/fetcher/stocks/{code}/data` | 종목 종합 데이터 조회 |

### Price Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fetcher/prices/{code}` | 최신 가격 조회 |
| GET | `/api/v1/fetcher/prices/{code}/history` | 가격 이력 조회 |

### Flow Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fetcher/flows/{code}` | 최신 수급 조회 |
| GET | `/api/v1/fetcher/flows/{code}/history` | 수급 이력 조회 |

### Disclosure Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fetcher/disclosures` | 최근 공시 목록 |
| GET | `/api/v1/fetcher/disclosures/{code}` | 종목별 공시 목록 |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/fetcher/collect` | 수집 트리거 |
| POST | `/api/v1/fetcher/collect/{code}` | 특정 종목 수집 |
| POST | `/api/v1/fetcher/refresh-stocks` | 종목 마스터 갱신 |

### Request/Response Examples

**POST /api/v1/fetcher/collect**
```json
// Request
{
  "collector_type": "price"  // price, flow, fundamental, marketcap, disclosure
}

// Response
{
  "success": true,
  "collector_type": "price",
  "message": "Collection triggered successfully"
}
```

**GET /api/v1/fetcher/prices/{code}/history?from=2026-01-01&to=2026-01-17**
```json
{
  "stock_code": "005930",
  "prices": [
    {
      "stock_code": "005930",
      "trade_date": "2026-01-17",
      "open_price": 85000,
      "high_price": 86000,
      "low_price": 84500,
      "close_price": 85500,
      "volume": 12500000,
      "trading_value": 1068750000000
    }
  ],
  "count": 1
}
```

---

## 🧪 테스트 전략

### 단위 테스트

- [x] Domain 모델 테스트
- [ ] Naver 클라이언트 파싱 테스트
- [ ] DART 클라이언트 API 테스트
- [ ] Repository UPSERT 테스트

### 통합 테스트

- [ ] 전체 수집 흐름 테스트
- [ ] API 엔드포인트 테스트
- [ ] 스케줄러 테스트

### 성능 테스트

- [ ] 2,500개 종목 수집 시간
- [ ] 병렬 처리 효율
- [ ] 메모리 사용량

---

## 📝 Changelog

### v1.1.0 (2026-01-17)
- v14 아키텍처에 맞게 모듈 재구현
- Domain/Service/Infra/API 레이어 분리
- PostgreSQL Repository 패턴 적용
- Naver Finance, DART 클라이언트 구현
- REST API 엔드포인트 추가

### v1.0.0 (v13 이전)
- 초기 설계 문서

---

**Version**: 1.1.0
**Status**: ✅ 구현 완료
