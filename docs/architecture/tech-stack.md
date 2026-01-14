# 기술 스택 선정 (Tech Stack)

> v14 시스템의 기술 스택 및 선정 근거를 정의합니다.

**Last Updated**: 2026-01-14

---

## 📋 개요

이 문서는 **v14 기술 스택의 SSOT**입니다.

### 목적
- 각 레이어별 기술 스택 정의
- 선정 근거 및 트레이드오프
- 주요 라이브러리/프레임워크
- 대안 기술과의 비교

---

## 🎯 전체 기술 스택

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| **Backend** | **Go** | 1.21+ | BFF, Runtime Engine |
| **Database** | **PostgreSQL** | 15+ | SSOT, 트랜잭션 |
| **Cache** | **Redis** | 7.0+ | 읽기 가속 |
| **Frontend** | **Next.js** | 14+ | Web UI (App Router) |
| **UI Library** | **shadcn/ui** | Latest | 컴포넌트 |
| **Deployment** | **Docker** | Latest | 컨테이너화 |
| **Orchestration** | **Docker Compose** | Latest | 로컬 개발 |

---

## 🔷 Backend: Go 1.21+

### 선정 근거

#### 1. 성능 (Performance)
```
✅ 컴파일 언어 - 네이티브 바이너리
✅ 고루틴 - 경량 동시성 (수만 개 가능)
✅ 낮은 메모리 사용량
✅ 빠른 시작 시간 (< 1초)
```

**벤치마크** (vs. Python, Node.js):
| 메트릭 | Go | Python | Node.js |
|--------|-----|--------|---------|
| 요청 처리 속도 | **50k req/s** | 5k req/s | 30k req/s |
| 메모리 사용량 | **50MB** | 200MB | 150MB |
| Cold Start | **< 1초** | 3-5초 | 2-3초 |
| 동시성 | **고루틴** | 쓰레드/프로세스 | 이벤트 루프 |

---

#### 2. 동시성 모델 (Concurrency)

**고루틴 (Goroutine)**:
```go
// 수만 개의 동시 작업 가능
for _, symbol := range symbols {
    go func(sym string) {
        price := fetchPrice(sym)
        ch <- price
    }(symbol)
}
```

**채널 (Channel)**:
```go
// 안전한 데이터 공유
priceCh := make(chan Price, 1000)

// Producer
go func() {
    for price := range priceStream {
        priceCh <- price
    }
}()

// Consumer
go func() {
    for price := range priceCh {
        processPrice(price)
    }
}()
```

**v14 사용 사례**:
- 실시간 시세 수신 (WebSocket)
- 병렬 포지션 체크 (Exit Engine)
- 동시 주문 제출 (Execution)

---

#### 3. 타입 안전성 (Type Safety)

**컴파일 타임 타입 체크**:
```go
type Position struct {
    ID        string
    Symbol    string
    Quantity  int64
    AvgPrice  decimal.Decimal  // ✅ 정확한 소수점 계산
}

// ✅ 컴파일 에러 - 타입 불일치
var pos Position
pos.Quantity = "100"  // Cannot use type string as int64
```

**인터페이스**:
```go
type PriceProvider interface {
    GetCurrentPrice(ctx context.Context, symbol string) (*Price, error)
}

// ✅ 컴파일 타임에 인터페이스 구현 검증
```

---

#### 4. 표준 라이브러리 (Standard Library)

**풍부한 표준 라이브러리**:
```go
import (
    "context"           // Context 관리
    "net/http"          // HTTP 서버/클라이언트
    "encoding/json"     // JSON 직렬화
    "time"              // 시간 처리
    "database/sql"      // DB 인터페이스
)
```

**외부 의존성 최소화**:
- HTTP 서버: 표준 라이브러리만으로 가능
- JSON 처리: 내장
- 동시성: 언어 차원 지원

---

### 주요 라이브러리

#### 1. Web Framework

**선택: Gin** (https://github.com/gin-gonic/gin)

**선정 근거**:
```
✅ 빠른 성능 (Radix Tree 기반 라우팅)
✅ 풍부한 미들웨어
✅ 간단한 API
✅ 활발한 커뮤니티
```

**대안 비교**:
| Framework | 성능 | 학습 곡선 | 커뮤니티 |
|-----------|------|-----------|----------|
| **Gin** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Echo | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Fiber | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| Chi | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

**사용 예시**:
```go
// main.go
r := gin.Default()

r.GET("/api/stocks", handlers.ListStocks)
r.POST("/api/orders", handlers.SubmitOrder)

r.Run(":8080")
```

---

#### 2. PostgreSQL Driver

**선택: pgx** (https://github.com/jackc/pgx)

**선정 근거**:
```
✅ 네이티브 PostgreSQL 드라이버 (가장 빠름)
✅ Connection Pool 내장
✅ Prepared Statement 지원
✅ COPY 지원 (대량 데이터 삽입)
✅ Context 지원
```

**vs. database/sql + pq**:
| 기능 | pgx | database/sql + pq |
|------|-----|-------------------|
| 성능 | **30% 빠름** | 기준 |
| 타입 지원 | **PostgreSQL 전용** | 제한적 |
| Pool | **내장** | 별도 설정 |
| COPY | **지원** | 미지원 |

**사용 예시**:
```go
// Connection Pool
config, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
pool, _ := pgxpool.NewWithConfig(context.Background(), config)

// Query
var stock Stock
err := pool.QueryRow(ctx,
    "SELECT code, name FROM market.stocks WHERE code = $1",
    "005930",
).Scan(&stock.Code, &stock.Name)

// Batch Insert (COPY)
_, err = pool.CopyFrom(ctx,
    pgx.Identifier{"market", "prices"},
    []string{"symbol", "price", "volume", "traded_at"},
    pgx.CopyFromSlice(len(prices), func(i int) ([]any, error) {
        return []any{
            prices[i].Symbol,
            prices[i].Price,
            prices[i].Volume,
            prices[i].TradedAt,
        }, nil
    }),
)
```

---

#### 3. Redis Client

**선택: go-redis** (https://github.com/redis/go-redis)

**선정 근거**:
```
✅ 공식 Redis 클라이언트
✅ Redis 7.0+ 모든 기능 지원
✅ Connection Pool 내장
✅ Pipeline 지원
✅ Pub/Sub 지원
```

**사용 예시**:
```go
// Client
rdb := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    PoolSize: 10,
})

// Get/Set
err := rdb.Set(ctx, "price:005930", "70000", 60*time.Second).Err()
val, err := rdb.Get(ctx, "price:005930").Result()

// Pipeline (배치 처리)
pipe := rdb.Pipeline()
for _, price := range prices {
    pipe.Set(ctx, "price:"+price.Symbol, price.Price, 60*time.Second)
}
pipe.Exec(ctx)
```

---

#### 4. Decimal 계산

**선택: shopspring/decimal** (https://github.com/shopspring/decimal)

**선정 근거**:
```
✅ 정확한 소수점 계산 (float64 부정확성 회피)
✅ 금융 계산에 필수
✅ SQL DECIMAL 타입과 호환
```

**float64의 문제**:
```go
// ❌ float64 - 부정확
var price float64 = 0.1
var qty float64 = 0.2
total := price + qty
fmt.Println(total)  // 0.30000000000000004

// ✅ decimal - 정확
price := decimal.NewFromFloat(0.1)
qty := decimal.NewFromFloat(0.2)
total := price.Add(qty)
fmt.Println(total)  // 0.3
```

**v14 사용 사례**:
- 주문 가격 계산
- 포지션 평균 단가
- 손익 계산
- 비중 계산

---

#### 5. 로깅

**선택: zerolog** (https://github.com/rs/zerolog)

**선정 근거**:
```
✅ 제로 할당 (Zero Allocation)
✅ JSON 구조화 로그
✅ 빠른 성능
✅ 레벨 기반 필터링
```

**사용 예시**:
```go
import "github.com/rs/zerolog/log"

log.Info().
    Str("symbol", "005930").
    Dec("price", price).
    Msg("Price updated")

// Output (JSON):
// {"level":"info","symbol":"005930","price":"70000","time":"2024-01-14T10:00:00Z","message":"Price updated"}
```

---

#### 6. 의존성 주입

**선택: Wire** (https://github.com/google/wire)

**선정 근거**:
```
✅ 컴파일 타임 DI (런타임 리플렉션 없음)
✅ Google 공식 라이브러리
✅ 타입 안전성
✅ 성능 오버헤드 제로
```

**사용 예시**:
```go
// wire.go
//go:build wireinject
// +build wireinject

func InitializeApp() (*App, error) {
    wire.Build(
        // Infrastructure
        postgres.NewPool,
        redis.NewClient,
        kis.NewClient,

        // Repositories
        repository.NewStockRepository,
        repository.NewPositionRepository,

        // Services
        pricesync.NewService,
        exit.NewService,
        execution.NewService,

        // API
        handlers.NewStockHandler,
        router.NewRouter,

        // App
        NewApp,
    )
    return &App{}, nil
}
```

---

#### 7. 테스팅

**선택: testify** (https://github.com/stretchr/testify)

**선정 근거**:
```
✅ Assertion 라이브러리
✅ Mock 생성 (mockery)
✅ Test Suite 지원
```

**사용 예시**:
```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestExitEngine_CheckConditions(t *testing.T) {
    // Arrange
    mockPrice := mock.NewMockPriceProvider()
    mockPrice.On("GetCurrentPrice", mock.Anything, "005930").
        Return(&Price{Symbol: "005930", Price: decimal.NewFromInt(10000)}, nil)

    engine := exit.NewService(mockPrice, nil)

    // Act
    decision, err := engine.CheckPosition(context.Background(), "pos1")

    // Assert
    assert.NoError(t, err)
    assert.True(t, decision.ShouldExit)
    mockPrice.AssertExpectations(t)
}
```

---

## 🗄️ Database: PostgreSQL 15+

### 선정 근거

#### 1. ACID 트랜잭션
```
✅ Atomicity - 원자성
✅ Consistency - 일관성
✅ Isolation - 격리성
✅ Durability - 지속성
```

**v14 요구사항**:
- 주문/체결의 원자적 처리
- 포지션 상태 일관성
- ExitEvent 정합성

---

#### 2. 강력한 타입 시스템

**지원 타입**:
```sql
-- 숫자
DECIMAL(10, 2)          -- 정확한 소수점 (금융 필수)
BIGINT                  -- 대량 거래량

-- 시간
TIMESTAMP               -- 정확한 시간 기록
TIMESTAMPTZ             -- 타임존 포함

-- JSON
JSONB                   -- 유연한 메타데이터 저장
```

---

#### 3. 고급 기능

**Window Functions**:
```sql
-- 이동 평균
SELECT symbol, price,
    AVG(price) OVER (
        PARTITION BY symbol
        ORDER BY traded_at
        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
    ) AS ma_20
FROM market.prices;
```

**CTE (Common Table Expressions)**:
```sql
-- 복잡한 쿼리 가독성
WITH open_positions AS (
    SELECT * FROM trade.positions WHERE status = 'OPEN'
),
current_prices AS (
    SELECT DISTINCT ON (symbol) symbol, price
    FROM market.prices
    ORDER BY symbol, traded_at DESC
)
SELECT p.*, c.price AS current_price
FROM open_positions p
JOIN current_prices c ON p.symbol = c.symbol;
```

**Full Text Search**:
```sql
-- 종목명 검색
SELECT * FROM market.stocks
WHERE to_tsvector('simple', name) @@ to_tsquery('simple', '삼성');
```

---

#### 4. 확장성

**Partitioning**:
```sql
-- 날짜별 파티셔닝 (시계열 데이터)
CREATE TABLE market.prices (
    symbol VARCHAR(10),
    price DECIMAL(10, 2),
    traded_at TIMESTAMP
) PARTITION BY RANGE (traded_at);

CREATE TABLE market.prices_2024_01
    PARTITION OF market.prices
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

**Replication**:
```
Primary (Master) → Standby (Replica)
    ↓
읽기 부하 분산
장애 시 자동 Failover
```

---

### 대안 비교

| Database | ACID | 타입 | 성능 | 확장성 | 선택 |
|----------|------|------|------|--------|------|
| **PostgreSQL** | ✅ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ✅ |
| MySQL | ✅ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ❌ |
| MongoDB | ❌ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ |

**PostgreSQL 선택 이유**:
- 금융 데이터 정합성 필수 (ACID)
- DECIMAL 타입 지원
- 복잡한 쿼리 지원
- v10/v13에서 검증됨

---

## 🚀 Cache: Redis 7.0+

### 선정 근거

#### 1. In-Memory 성능
```
✅ 마이크로초 단위 응답 (< 1ms)
✅ 초당 수만 건 처리
```

#### 2. 다양한 자료구조
```
String    - 단순 Key-Value
Hash      - 객체 저장
List      - 시계열 데이터
Set       - 유니크 집합
Sorted Set - 순위 데이터
```

#### 3. TTL (Time To Live)
```redis
SET price:005930 "70000" EX 60  # 60초 후 자동 삭제
```

**v14 사용 사례**:
```
price:{symbol}          # 현재가 (TTL: 60초)
position:{id}           # 포지션 상태 (TTL: 30초)
config:{key}            # 설정 (TTL: 10분)
```

---

### 대안 비교

| Cache | 성능 | 기능 | 운영성 | 선택 |
|-------|------|------|--------|------|
| **Redis** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ✅ |
| Memcached | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ |
| Hazelcast | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ❌ |

---

## 🎨 Frontend: Next.js 14+

### 선정 근거

#### 1. App Router
```
✅ React Server Components
✅ Streaming SSR
✅ 향상된 라우팅
```

#### 2. 성능
```
✅ 자동 코드 스플리팅
✅ 이미지 최적화
✅ Font 최적화
```

#### 3. 개발 경험
```
✅ TypeScript 완벽 지원
✅ Hot Reload
✅ API Routes
```

---

### shadcn/ui

**선정 근거**:
```
✅ Radix UI 기반 (접근성)
✅ Tailwind CSS
✅ 복사해서 사용 (의존성 최소화)
✅ 커스터마이징 용이
```

---

## 🐳 Deployment: Docker

### 컨테이너화

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: aegis_v14
      POSTGRES_USER: aegis_v14
      POSTGRES_PASSWORD: aegis_v14_won
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    environment:
      DATABASE_URL: postgresql://aegis_v14:aegis_v14_won@postgres:5432/aegis_v14
      REDIS_URL: redis:6379

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - backend
    environment:
      NEXT_PUBLIC_API_URL: http://backend:8080

volumes:
  postgres_data:
```

---

## 📊 기술 스택 요약

### Backend

| 카테고리 | 선택 | 버전 |
|----------|------|------|
| Language | Go | 1.21+ |
| Web Framework | Gin | 1.9+ |
| DB Driver | pgx | 5.5+ |
| Redis Client | go-redis | 9.0+ |
| Decimal | shopspring/decimal | 1.3+ |
| Logging | zerolog | 1.31+ |
| DI | Wire | 0.5+ |
| Testing | testify | 1.8+ |

### Database

| 카테고리 | 선택 | 버전 |
|----------|------|------|
| RDBMS | PostgreSQL | 15+ |
| Cache | Redis | 7.0+ |
| Migration | golang-migrate | 4.16+ |

### Frontend

| 카테고리 | 선택 | 버전 |
|----------|------|------|
| Framework | Next.js | 14+ |
| UI Library | shadcn/ui | Latest |
| Styling | Tailwind CSS | 3.0+ |
| Language | TypeScript | 5.0+ |

---

## 🔍 참고 문서

- [레이어 구조 설계](./layer-design.md)
- [데이터 흐름 설계](./data-flow.md)
- [모듈 카탈로그](../modules/module-catalog.md)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
