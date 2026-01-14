# 레이어 구조 설계 (Layer Design)

> v14 시스템의 레이어 아키텍처 및 Go 프로젝트 구조를 정의합니다.

**Last Updated**: 2026-01-14

---

## 📋 개요

이 문서는 **v14 레이어 구조의 SSOT**입니다.

### 목적
- 5개 레이어 구조 상세 정의
- Go 프로젝트 디렉토리 구조
- 각 레이어의 책임과 경계
- 레이어 간 통신 규칙
- 패키지 의존성 방향

---

## 🏗️ 레이어 아키텍처 개요

### 5-Layer Architecture

```
┌─────────────────────────────────────────┐
│   API Layer (API 계층)                   │  ← HTTP/WS 엔드포인트
│   - BFF (Backend for Frontend)          │
└─────────────────────────────────────────┘
                   ↑
┌─────────────────────────────────────────┐
│   Control Layer (제어 계층)               │  ← 횡단 관심사
│   - Risk Management                      │
│   - Monitoring                           │
└─────────────────────────────────────────┘
                   ↑
┌─────────────────────────────────────────┐
│   Strategy Layer (전략 계층)              │  ← 전략 로직
│   - Universe, Signals, Ranking          │
│   - Portfolio                            │
└─────────────────────────────────────────┘
                   ↑
┌─────────────────────────────────────────┐
│   Core Runtime Layer (런타임 계층)        │  ← 실시간 실행
│   - PriceSync, Exit, Reentry            │
│   - Execution                            │
└─────────────────────────────────────────┘
                   ↑
┌─────────────────────────────────────────┐
│   Infrastructure Layer (인프라 계층)      │  ← 외부 연동
│   - External APIs, Database, Cache      │
└─────────────────────────────────────────┘
```

**의존성 규칙**:
- ✅ 상위 → 하위 (인터페이스 통해서만)
- ❌ 하위 → 상위 (절대 금지)
- ❌ 레이어 건너뛰기 (계층 순서 준수)

---

## 📁 Go 프로젝트 구조

### 전체 구조

```
backend/
├── cmd/                          # 실행 파일
│   ├── api/                      # BFF 서버
│   │   └── main.go
│   ├── runtime/                  # Runtime Engine
│   │   └── main.go
│   └── scheduler/                # 전략 스케줄러
│       └── main.go
│
├── internal/                     # 내부 패키지 (외부 노출 금지)
│   ├── api/                      # API Layer
│   │   ├── handlers/             # HTTP 핸들러
│   │   ├── middleware/           # 미들웨어
│   │   ├── router/               # 라우팅
│   │   └── websocket/            # WebSocket 핸들러
│   │
│   ├── control/                  # Control Layer
│   │   ├── risk/                 # Risk Management
│   │   └── monitoring/           # Monitoring & Alerting
│   │
│   ├── strategy/                 # Strategy Layer
│   │   ├── universe/             # Universe Selection
│   │   ├── signals/              # Signal Generation
│   │   ├── ranking/              # Ranking Engine
│   │   └── portfolio/            # Portfolio Construction
│   │
│   ├── runtime/                  # Core Runtime Layer
│   │   ├── pricesync/            # PriceSync
│   │   ├── exit/                 # Exit Engine
│   │   ├── reentry/              # Reentry Engine
│   │   └── execution/            # Execution Service
│   │
│   ├── infra/                    # Infrastructure Layer
│   │   ├── external/             # External APIs
│   │   │   ├── kis/              # KIS API Client
│   │   │   └── naver/            # Naver Finance Client
│   │   ├── database/             # Database Access
│   │   │   ├── postgres/         # PostgreSQL
│   │   │   └── repository/       # Repository 구현
│   │   └── cache/                # Redis Cache
│   │
│   ├── domain/                   # 도메인 모델 (공유)
│   │   ├── models/               # 엔티티 모델
│   │   ├── events/               # 도메인 이벤트
│   │   └── errors/               # 도메인 에러
│   │
│   └── pkg/                      # 내부 공유 라이브러리
│       ├── config/               # 설정 관리
│       ├── logger/               # 로깅
│       ├── validator/            # 검증
│       └── utils/                # 유틸리티
│
├── pkg/                          # 외부 노출 가능 라이브러리
│   └── contracts/                # 공개 인터페이스
│
├── migrations/                   # DB 마이그레이션
│   ├── 000001_create_stocks_table.up.sql
│   └── 000001_create_stocks_table.down.sql
│
├── configs/                      # 설정 파일
│   ├── config.yaml
│   ├── config.dev.yaml
│   └── config.prod.yaml
│
├── scripts/                      # 스크립트
│   └── db/                       # DB 초기화
│
├── tests/                        # 통합 테스트
│   ├── integration/
│   └── e2e/
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🔷 레이어별 상세 설계

### 1. Infrastructure Layer

**위치**: `internal/infra/`

**책임**:
- 외부 시스템 연동
- 데이터 저장/조회
- 캐싱

#### 1.1 External APIs (`internal/infra/external/`)

**구조**:
```
external/
├── kis/
│   ├── client.go           # KIS API 클라이언트
│   ├── websocket.go        # KIS WebSocket
│   ├── rest.go             # KIS REST API
│   ├── models.go           # KIS API 모델
│   └── mock/               # Mock 구현
│       └── mock_client.go
└── naver/
    ├── client.go           # Naver Finance 클라이언트
    ├── scraper.go          # HTML 파싱
    └── mock/
        └── mock_client.go
```

**인터페이스**:
```go
// kis/client.go
package kis

type Client interface {
    // WebSocket
    SubscribePrice(ctx context.Context, symbols []string) error
    UnsubscribePrice(ctx context.Context, symbols []string) error

    // REST API
    GetPrice(ctx context.Context, symbol string) (*Price, error)
    SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error)
}

type Config struct {
    AppKey    string
    SecretKey string
    BaseURL   string
}

func NewClient(config Config) Client {
    return &client{config: config}
}
```

---

#### 1.2 Database (`internal/infra/database/`)

**구조**:
```
database/
├── postgres/
│   ├── conn.go             # Connection Pool
│   ├── transaction.go      # Transaction 관리
│   └── health.go           # Health Check
└── repository/
    ├── stock_repo.go       # 종목 Repository
    ├── price_repo.go       # 가격 Repository
    ├── position_repo.go    # 포지션 Repository
    ├── order_repo.go       # 주문 Repository
    └── mock/               # Mock Repository
        └── mock_stock_repo.go
```

**인터페이스**:
```go
// repository/stock_repo.go
package repository

type StockRepository interface {
    // CRUD
    GetByCode(ctx context.Context, code string) (*domain.Stock, error)
    List(ctx context.Context, filter StockFilter) ([]*domain.Stock, error)
    Create(ctx context.Context, stock *domain.Stock) error
    Update(ctx context.Context, stock *domain.Stock) error

    // Batch
    BatchCreate(ctx context.Context, stocks []*domain.Stock) error
}

type StockFilter struct {
    Market   string
    Delisted bool
    Limit    int
    Offset   int
}
```

---

#### 1.3 Cache (`internal/infra/cache/`)

**구조**:
```
cache/
├── redis.go                # Redis 클라이언트
├── service.go              # Cache Service
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// cache/service.go
package cache

type Service interface {
    // Basic Operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) error

    // Cache Patterns
    GetOrLoad(ctx context.Context, key string, loader func() ([]byte, error), ttl time.Duration) ([]byte, error)
}
```

---

### 2. Core Runtime Layer

**위치**: `internal/runtime/`

**책임**:
- 실시간 시세 동기화
- 자동 청산/재진입
- 주문 실행

#### 2.1 PriceSync (`internal/runtime/pricesync/`)

**구조**:
```
pricesync/
├── service.go              # PriceSync Service
├── sync.go                 # 동기화 로직
├── fallback.go             # Fallback 전략
├── models.go               # 내부 모델
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// pricesync/service.go
package pricesync

type Service interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // Subscribe
    Subscribe(ctx context.Context, symbols []string) error
    Unsubscribe(ctx context.Context, symbols []string) error

    // Query
    GetCurrentPrice(ctx context.Context, symbol string) (*Price, error)
}

type Config struct {
    PrimarySource   string        // "kis" or "naver"
    FallbackEnabled bool
    SyncInterval    time.Duration
}
```

---

#### 2.2 Exit Engine (`internal/runtime/exit/`)

**구조**:
```
exit/
├── service.go              # Exit Engine Service
├── checker.go              # 청산 조건 체크
├── rules.go                # 청산 규칙 (Hybrid, ATR)
├── gate.go                 # Control Gate
├── profile.go              # Exit Profile
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// exit/service.go
package exit

type Service interface {
    // Check
    CheckExitConditions(ctx context.Context) ([]*ExitDecision, error)
    CheckPosition(ctx context.Context, positionID string) (*ExitDecision, error)

    // Control
    EnableGlobalExit(ctx context.Context, enabled bool) error
    SetSymbolOverride(ctx context.Context, symbol string, override SymbolOverride) error
    GetExitStatus(ctx context.Context) (*ExitStatus, error)
}

type ExitDecision struct {
    PositionID   string
    Symbol       string
    ShouldExit   bool
    ExitPrice    decimal.Decimal
    ExitReason   string
    RuleType     string  // "hybrid_pct", "atr", "hard_stop"
}
```

---

#### 2.3 Reentry Engine (`internal/runtime/reentry/`)

**구조**:
```
reentry/
├── service.go              # Reentry Engine Service
├── handler.go              # ExitEvent 핸들러
├── rules.go                # 재진입 규칙
├── gate.go                 # Control Gate
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// reentry/service.go
package reentry

type Service interface {
    // Event Handler
    OnExitEvent(ctx context.Context, event *domain.ExitEvent) error

    // Query
    GetReentryStatus(ctx context.Context, symbol string) (*ReentryStatus, error)

    // Control
    EnableGlobalReentry(ctx context.Context, enabled bool) error
    SetSymbolOverride(ctx context.Context, symbol string, override SymbolOverride) error
}
```

---

#### 2.4 Execution (`internal/runtime/execution/`)

**구조**:
```
execution/
├── service.go              # Execution Service
├── submitter.go            # 주문 제출
├── tracker.go              # 체결 추적
├── retry.go                # 재시도 로직
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// execution/service.go
package execution

type Service interface {
    // Submit
    SubmitOrder(ctx context.Context, order *domain.Order) (string, error)

    // Query
    GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)
    ListOrders(ctx context.Context, filter OrderFilter) ([]*domain.Order, error)

    // Cancel
    CancelOrder(ctx context.Context, orderID string) error
}
```

---

### 3. Strategy Layer

**위치**: `internal/strategy/`

**책임**:
- 투자 가능 종목 선정
- 시그널 생성
- 종합 점수 산출
- 포트폴리오 구성

#### 3.1 Universe (`internal/strategy/universe/`)

**구조**:
```
universe/
├── service.go              # Universe Service
├── filters.go              # 필터 (유동성, 시총 등)
├── screener.go             # 스크리닝 로직
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// universe/service.go
package universe

type Service interface {
    // Generate
    GenerateUniverse(ctx context.Context, date time.Time) ([]string, error)

    // Query
    GetCurrentUniverse(ctx context.Context) ([]string, error)
    GetUniverseHistory(ctx context.Context, startDate, endDate time.Time) (map[time.Time][]string, error)
}

type Config struct {
    MinMarketCap    decimal.Decimal
    MinAvgVolume    int64
    ExcludeMarkets  []string
}
```

---

#### 3.2 Signals (`internal/strategy/signals/`)

**구조**:
```
signals/
├── service.go              # Signal Service
├── factors/                # 팩터별 계산
│   ├── momentum.go
│   ├── value.go
│   └── quality.go
└── mock/
    └── mock_service.go
```

---

#### 3.3 Ranking (`internal/strategy/ranking/`)

**구조**:
```
ranking/
├── service.go              # Ranking Service
├── scorer.go               # 종합 점수 산출
└── mock/
    └── mock_service.go
```

---

#### 3.4 Portfolio (`internal/strategy/portfolio/`)

**구조**:
```
portfolio/
├── service.go              # Portfolio Service
├── builder.go              # 포트폴리오 구성
├── rebalancer.go           # 리밸런싱
└── mock/
    └── mock_service.go
```

---

### 4. Control Layer

**위치**: `internal/control/`

**책임**:
- 리스크 관리
- 모니터링/알람

#### 4.1 Risk Management (`internal/control/risk/`)

**구조**:
```
risk/
├── service.go              # Risk Service
├── checker.go              # 리스크 체크
├── limits.go               # 한도 관리
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// risk/service.go
package risk

type Service interface {
    // Check
    CheckRiskLimits(ctx context.Context, portfolio *domain.Portfolio) (*RiskReport, error)
    ApproveOrder(ctx context.Context, order *domain.Order) (bool, string, error)

    // Query
    GetCurrentRisk(ctx context.Context) (*RiskMetrics, error)
}

type RiskReport struct {
    Approved      bool
    ViolationCode string
    Message       string
    Metrics       *RiskMetrics
}
```

---

#### 4.2 Monitoring (`internal/control/monitoring/`)

**구조**:
```
monitoring/
├── service.go              # Monitoring Service
├── collector.go            # 메트릭 수집
├── alerter.go              # 알람
└── mock/
    └── mock_service.go
```

**인터페이스**:
```go
// monitoring/service.go
package monitoring

type Service interface {
    // Metrics
    RecordMetric(ctx context.Context, metric *Metric) error
    GetMetrics(ctx context.Context, filter MetricFilter) ([]*Metric, error)

    // Alerts
    SendAlert(ctx context.Context, alert *Alert) error

    // Health
    GetSystemHealth(ctx context.Context) (*HealthStatus, error)
}
```

---

### 5. API Layer

**위치**: `internal/api/`

**책임**:
- HTTP REST API 제공
- WebSocket 실시간 통신
- 인증/인가

#### 5.1 Handlers (`internal/api/handlers/`)

**구조**:
```
handlers/
├── health.go               # Health Check
├── stocks.go               # 종목 API
├── prices.go               # 가격 API
├── positions.go            # 포지션 API
├── orders.go               # 주문 API
├── portfolio.go            # 포트폴리오 API
├── signals.go              # 시그널 API
└── performance.go          # 성과 API
```

**예시**:
```go
// handlers/stocks.go
package handlers

type StockHandler struct {
    stockRepo repository.StockRepository
}

func NewStockHandler(stockRepo repository.StockRepository) *StockHandler {
    return &StockHandler{stockRepo: stockRepo}
}

// GET /api/stocks
func (h *StockHandler) ListStocks(c *gin.Context) {
    // Request Parsing
    var req ListStocksRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(400, ErrorResponse{Error: err.Error()})
        return
    }

    // Business Logic (Repository 호출)
    stocks, err := h.stockRepo.List(c.Request.Context(), repository.StockFilter{
        Market: req.Market,
        Limit:  req.Limit,
        Offset: req.Offset,
    })
    if err != nil {
        c.JSON(500, ErrorResponse{Error: err.Error()})
        return
    }

    // Response
    c.JSON(200, ListStocksResponse{
        Data: stocks,
        Pagination: Pagination{
            Page:  req.Page,
            Limit: req.Limit,
            Total: len(stocks),
        },
    })
}
```

---

#### 5.2 Router (`internal/api/router/`)

**구조**:
```
router/
├── router.go               # 라우터 설정
├── routes.go               # 라우트 정의
└── middleware.go           # 미들웨어 체인
```

**예시**:
```go
// router/router.go
package router

func NewRouter(handlers *Handlers) *gin.Engine {
    r := gin.Default()

    // Middleware
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())
    r.Use(middleware.Logger())

    // Health
    r.GET("/health", handlers.Health.Check)

    // API v1
    v1 := r.Group("/api/v1")
    {
        // Stocks
        v1.GET("/stocks", handlers.Stock.ListStocks)
        v1.GET("/stocks/:code", handlers.Stock.GetStock)

        // Prices
        v1.GET("/prices", handlers.Price.ListPrices)
        v1.GET("/prices/:symbol/current", handlers.Price.GetCurrentPrice)

        // Positions
        v1.GET("/positions", handlers.Position.ListPositions)
        v1.GET("/positions/:id", handlers.Position.GetPosition)

        // Orders
        v1.POST("/orders", handlers.Order.SubmitOrder)
        v1.GET("/orders", handlers.Order.ListOrders)
        v1.DELETE("/orders/:id", handlers.Order.CancelOrder)
    }

    // WebSocket
    r.GET("/ws/prices", handlers.WebSocket.PriceStream)

    return r
}
```

---

## 🔗 레이어 간 통신 규칙

### 1. 인터페이스 의존

```go
// ❌ 금지 - 구체 타입 의존
package exit

import "backend/internal/runtime/pricesync"  // 구체 패키지

type ExitEngine struct {
    priceSync *pricesync.Service  // 구체 타입
}

// ✅ 허용 - 인터페이스 의존
package exit

type PriceProvider interface {  // 인터페이스 정의
    GetCurrentPrice(ctx context.Context, symbol string) (*Price, error)
}

type ExitEngine struct {
    priceProvider PriceProvider  // 인터페이스
}
```

---

### 2. 의존성 주입 (DI)

```go
// main.go
func main() {
    // Infrastructure
    kisClient := kis.NewClient(kisConfig)
    db := postgres.NewPool(dbConfig)
    stockRepo := repository.NewStockRepository(db)

    // Runtime
    priceSync := pricesync.NewService(kisClient, stockRepo)
    exitEngine := exit.NewService(priceSync, positionRepo)
    execution := execution.NewService(kisClient, orderRepo)

    // API
    stockHandler := handlers.NewStockHandler(stockRepo)
    router := router.NewRouter(&Handlers{
        Stock: stockHandler,
    })

    router.Run(":8080")
}
```

---

### 3. 이벤트 기반 통신

**발행자** (Exit Engine):
```go
// exit/service.go
func (s *Service) ProcessExit(ctx context.Context) error {
    // 1. 청산 실행
    err := s.execution.SubmitOrder(ctx, exitOrder)

    // 2. ExitEvent 생성 (DB에 저장)
    event := &domain.ExitEvent{
        PositionID: position.ID,
        Symbol:     position.Symbol,
        ExitPrice:  exitPrice,
        ExitAt:     time.Now(),
    }
    return s.eventRepo.CreateExitEvent(ctx, event)
}
```

**구독자** (Reentry Engine):
```go
// reentry/service.go
func (s *Service) PollExitEvents(ctx context.Context) error {
    // DB에서 미처리 ExitEvent 조회
    events, err := s.eventRepo.GetUnprocessedExitEvents(ctx)
    if err != nil {
        return err
    }

    for _, event := range events {
        s.OnExitEvent(ctx, event)
    }
    return nil
}
```

---

## 📦 패키지 의존성 규칙

### 1. import 규칙

```go
// ✅ 허용 - 하위 레이어 import
package api  // API Layer

import (
    "backend/internal/runtime/exit"      // Runtime Layer
    "backend/internal/strategy/signals"  // Strategy Layer
    "backend/internal/infra/database"    // Infrastructure Layer
)

// ❌ 금지 - 상위 레이어 import
package infra  // Infrastructure Layer

import (
    "backend/internal/runtime/exit"  // 상위 레이어 - 금지!
)

// ❌ 금지 - 레이어 건너뛰기
package strategy  // Strategy Layer

import (
    "backend/internal/infra/external/kis"  // Infrastructure 직접 - 금지!
)
// 올바른 방법: Repository 인터페이스 사용
```

---

### 2. 도메인 모델 공유

```go
// domain/models/stock.go
package models

type Stock struct {
    Code   string
    Name   string
    Market string
}

// 모든 레이어에서 사용 가능
import "backend/internal/domain/models"
```

---

## 🧪 테스트 전략

### 1. 단위 테스트 (Unit Test)

**각 레이어별로 독립 테스트**:
```go
// exit/service_test.go
func TestExitEngine_CheckExitConditions(t *testing.T) {
    // Mock 의존성
    mockPriceSync := mock.NewMockPriceProvider()
    mockPriceSync.GetCurrentPriceFunc = func(ctx context.Context, symbol string) (*Price, error) {
        return &Price{Symbol: symbol, Price: decimal.NewFromInt(10000)}, nil
    }

    mockRepo := mock.NewMockPositionRepository()
    mockRepo.ListOpenPositionsFunc = func(ctx context.Context) ([]*Position, error) {
        return []*Position{
            {ID: "pos1", Symbol: "005930", AvgPrice: decimal.NewFromInt(9000)},
        }, nil
    }

    // Service 생성
    service := exit.NewService(mockPriceSync, mockRepo)

    // 테스트
    decisions, err := service.CheckExitConditions(context.Background())
    assert.NoError(t, err)
    assert.Len(t, decisions, 1)
    assert.True(t, decisions[0].ShouldExit)
}
```

---

### 2. 통합 테스트 (Integration Test)

**레이어 간 통합 테스트**:
```go
// tests/integration/exit_flow_test.go
func TestExitFlow_EndToEnd(t *testing.T) {
    // 실제 PostgreSQL (testcontainers)
    db := setupTestDB(t)
    defer db.Close()

    // 실제 Repository
    stockRepo := repository.NewStockRepository(db)
    positionRepo := repository.NewPositionRepository(db)
    orderRepo := repository.NewOrderRepository(db)

    // Mock External API
    mockKIS := mock.NewMockKISClient()

    // Services
    priceSync := pricesync.NewService(mockKIS, stockRepo)
    exitEngine := exit.NewService(priceSync, positionRepo)
    execution := execution.NewService(mockKIS, orderRepo)

    // 시나리오 테스트
    // 1. Position 생성
    // 2. Price 업데이트
    // 3. Exit 조건 체크
    // 4. Order 제출
    // 5. 체결 확인
}
```

---

## 🔍 참고 문서

- [모듈 카탈로그](../modules/module-catalog.md)
- [모듈 의존성 맵](./module-dependencies.md)
- [데이터 흐름 설계](./data-flow.md)
- [시스템 아키텍처 개요](./system-overview.md)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
