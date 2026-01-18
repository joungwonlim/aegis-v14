# PriceSync DB 보호 설계

> Redis 없이 PostgreSQL만으로 3000 종목 가격 동기화를 안정적으로 처리하기 위한 보호 장치

## 1. 개요

### 문제
- 3000 종목 × 다중 소스(WS/REST/Naver) = 초당 수백~수천 DB 쓰기 가능
- UI/전략이 DB를 직접 폴링하면 조회 부하도 폭증
- REST Tier 확장 순간 DB가 먼저 흔들림

### 해결 전략
```
┌─────────────────────────────────────────────────────────────────┐
│                        DB 보호 3대 장치                          │
├─────────────────────────────────────────────────────────────────┤
│  1. Coalescing: 쓰기 부하 ↓ (심볼별 1초 1회, 변화 없으면 스킵)    │
│  2. In-memory Cache: 조회 부하 ↓ (캐시 히트 시 DB 안 감)         │
│  3. Pub/Sub + SSE: 폴링 → 푸시 (DB 조회 자체를 줄임)            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 아키텍처

```
KIS WS ──┐
KIS REST ┼──▶ ProcessTick ──▶ Coalescer ──▶ In-memory Cache ──▶ Broker
Naver ───┘                        │                │              │
                                  │                │              │
                           (1초 debounce)     (조회 서빙)     (구독자 푸시)
                                  │                │              │
                                  ▼                │              ▼
                            PostgreSQL ◀───────────┘         SSE/WS Handler
                           (prices_best)                          │
                                                                  ▼
                                                              Frontend
```

---

## 3. Coalescing 레이어

### 3.1 목적
- **심볼별 1초에 1번 이하**만 DB에 쓰기
- **가격 변화 없으면 완전 스킵**
- REST Tier 확장해도 DB 부하 예측 가능

### 3.2 구조

```go
// backend/internal/service/pricesync/coalescer.go

type Coalescer struct {
    mu           sync.RWMutex
    pending      map[string]*CoalescedTick  // symbol → 대기 중인 최신 틱
    lastWritten  map[string]*WrittenState   // symbol → 마지막 DB 기록 상태

    flushInterval time.Duration  // 기본 1초
    minChangeRate float64        // 최소 변화율 (ex: 0.01% 미만이면 스킵)
}

type CoalescedTick struct {
    Tick       price.Tick
    ReceivedAt time.Time
}

type WrittenState struct {
    Price     int64
    Timestamp time.Time
}
```

### 3.3 Coalescing 규칙

| 조건 | 동작 |
|------|------|
| 마지막 쓰기로부터 1초 미경과 | pending에 저장, 바로 쓰지 않음 |
| 가격이 이전과 동일 | 스킵 (freshness만 캐시 갱신) |
| 변화율 < 0.01% | 스킵 (노이즈 필터링) |
| 1초 경과 + 가격 변화 있음 | DB에 쓰기 |

### 3.4 Flush 루프

```go
func (c *Coalescer) flushLoop(ctx context.Context) {
    ticker := time.NewTicker(c.flushInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.flush(ctx)
        }
    }
}

func (c *Coalescer) flush(ctx context.Context) {
    c.mu.Lock()
    toFlush := make(map[string]*CoalescedTick)

    for symbol, pending := range c.pending {
        last, exists := c.lastWritten[symbol]

        // Skip if price unchanged
        if exists && last.Price == pending.Tick.LastPrice {
            continue
        }

        // Skip if too recent (< 1 second)
        if exists && time.Since(last.Timestamp) < c.flushInterval {
            continue
        }

        toFlush[symbol] = pending
        delete(c.pending, symbol)
    }
    c.mu.Unlock()

    // Batch write to DB
    for symbol, pending := range toFlush {
        if err := c.writeToDB(ctx, pending.Tick); err != nil {
            log.Error().Err(err).Str("symbol", symbol).Msg("Flush failed")
            continue
        }

        c.mu.Lock()
        c.lastWritten[symbol] = &WrittenState{
            Price:     pending.Tick.LastPrice,
            Timestamp: time.Now(),
        }
        c.mu.Unlock()
    }
}
```

### 3.5 DB 쓰기 최적화

**기존 (매 틱마다 3번 쓰기)**
```
ProcessTick
  → InsertTick (prices_ticks)     ← 매번
  → UpsertFreshness (freshness)   ← 매번
  → UpsertBestPrice (prices_best) ← 매번
```

**개선 (Coalescing 적용)**
```
ProcessTick
  → Cache 업데이트 (in-memory)    ← 매번 (빠름)
  → Coalescer.Enqueue             ← 매번 (빠름)

Flush (1초마다)
  → 변화 있는 심볼만 배치로 UpsertBestPrice
  → prices_ticks는 별도 배치 INSERT (5초마다)
```

---

## 4. In-memory Cache

### 4.1 목적
- **조회 시 DB 안 감** (캐시 히트율 99%+)
- 전략/UI가 자주 조회해도 DB 부하 없음

### 4.2 구조

```go
// backend/internal/service/pricesync/cache.go

type PriceCache struct {
    mu     sync.RWMutex
    prices map[string]*CachedPrice  // symbol → 캐시된 가격
}

type CachedPrice struct {
    BestPrice   int64
    ChangePrice *int64
    ChangeRate  *float64
    Source      price.Source
    Timestamp   time.Time
    UpdatedAt   time.Time  // 캐시 갱신 시각
}
```

### 4.3 조회 흐름

```
GetBestPrice(symbol)
    │
    ├── Cache Hit → 바로 반환 (1µs)
    │
    └── Cache Miss → DB 조회 → 캐시 저장 → 반환 (1ms)
```

### 4.4 캐시 갱신 전략

| 이벤트 | 동작 |
|--------|------|
| ProcessTick | 캐시 즉시 갱신 (SelectBest 적용) |
| DB Flush | 캐시 상태와 동기 |
| Startup | DB에서 prices_best 전체 로드 |

---

## 5. Broker (In-memory Pub/Sub)

### 5.1 목적
- 가격 변경 시 **구독자에게 즉시 푸시**
- UI/전략이 DB를 폴링하지 않음

### 5.2 구조

```go
// backend/internal/service/pricesync/broker.go

type Broker struct {
    mu          sync.RWMutex
    subscribers map[string][]chan PriceUpdate  // symbol → 구독자 채널들
    allSubs     []chan PriceUpdate             // 전체 구독자 (모든 심볼)
}

type PriceUpdate struct {
    Symbol      string
    Price       int64
    ChangePrice *int64
    ChangeRate  *float64
    Source      price.Source
    Timestamp   time.Time
}
```

### 5.3 API

```go
// 특정 심볼 구독
func (b *Broker) Subscribe(symbol string) <-chan PriceUpdate

// 전체 심볼 구독 (모니터링용)
func (b *Broker) SubscribeAll() <-chan PriceUpdate

// 구독 해제
func (b *Broker) Unsubscribe(symbol string, ch <-chan PriceUpdate)

// 가격 변경 시 호출 (Coalescer에서)
func (b *Broker) Publish(update PriceUpdate)
```

### 5.4 Publish 흐름

```go
func (b *Broker) Publish(update PriceUpdate) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    // 심볼별 구독자에게 전송
    if subs, ok := b.subscribers[update.Symbol]; ok {
        for _, ch := range subs {
            select {
            case ch <- update:
            default:
                // 채널 가득 참 - 느린 구독자 스킵 (non-blocking)
            }
        }
    }

    // 전체 구독자에게 전송
    for _, ch := range b.allSubs {
        select {
        case ch <- update:
        default:
        }
    }
}
```

---

## 6. SSE Handler

### 6.1 목적
- **HTTP로 가격 스트리밍** (WebSocket보다 간단)
- Frontend가 EventSource로 연결

### 6.2 엔드포인트

```
GET /api/v1/prices/stream?symbols=005930,000660,035720
GET /api/v1/prices/stream/all  (전체 - 모니터링용)
```

### 6.3 구현

```go
// backend/internal/api/handlers/price_stream.go

func (h *PriceStreamHandler) StreamPrices(w http.ResponseWriter, r *http.Request) {
    // SSE 헤더 설정
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // 구독할 심볼 파싱
    symbols := parseSymbols(r.URL.Query().Get("symbols"))

    // 각 심볼 구독
    channels := make([]<-chan PriceUpdate, len(symbols))
    for i, symbol := range symbols {
        channels[i] = h.broker.Subscribe(symbol)
        defer h.broker.Unsubscribe(symbol, channels[i])
    }

    // 통합 채널
    merged := mergeChannels(channels)

    // 스트리밍 루프
    flusher := w.(http.Flusher)

    for {
        select {
        case <-r.Context().Done():
            return
        case update := <-merged:
            data, _ := json.Marshal(update)
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

### 6.4 Frontend 사용

```typescript
// frontend/hooks/usePriceStream.ts

export function usePriceStream(symbols: string[]) {
  const [prices, setPrices] = useState<Map<string, PriceUpdate>>(new Map())

  useEffect(() => {
    const symbolsParam = symbols.join(',')
    const eventSource = new EventSource(`/api/v1/prices/stream?symbols=${symbolsParam}`)

    eventSource.onmessage = (event) => {
      const update: PriceUpdate = JSON.parse(event.data)
      setPrices(prev => new Map(prev).set(update.symbol, update))
    }

    return () => eventSource.close()
  }, [symbols.join(',')])

  return prices
}
```

---

## 7. 통합 흐름

```
┌──────────────────────────────────────────────────────────────────────┐
│                           전체 데이터 흐름                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  KIS WS/REST ──▶ ProcessTick                                        │
│                      │                                               │
│                      ▼                                               │
│              ┌──────────────┐                                        │
│              │   Cache      │◀─── UI/전략 조회 (즉시 반환)            │
│              │ (in-memory)  │                                        │
│              └──────┬───────┘                                        │
│                     │                                                │
│                     ▼                                                │
│              ┌──────────────┐     ┌──────────┐                       │
│              │  Coalescer   │────▶│  Broker  │──▶ SSE ──▶ Frontend  │
│              │ (1초 debounce)│     │ (Pub/Sub)│                       │
│              └──────┬───────┘     └──────────┘                       │
│                     │                                                │
│                     ▼ (변화 있을 때만)                                │
│              ┌──────────────┐                                        │
│              │  PostgreSQL  │                                        │
│              │ prices_best  │                                        │
│              └──────────────┘                                        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 8. 기대 효과

### 8.1 DB 쓰기 부하

| 시나리오 | 기존 | 개선 후 |
|---------|------|--------|
| WS 40종목 × 1틱/초 | 120 writes/sec | 40 writes/sec (가격 변화 시만) |
| REST 340종목 × Tier별 | ~100 writes/sec | ~30 writes/sec |
| **합계** | **~220 writes/sec** | **~70 writes/sec** (68% ↓) |

### 8.2 DB 조회 부하

| 시나리오 | 기존 | 개선 후 |
|---------|------|--------|
| UI 조회 (5개 탭 × 1초 폴링) | 5 queries/sec | 0 (SSE 구독) |
| 전략 조회 (10 전략 × 40 심볼) | 400 queries/sec | 0 (캐시 히트) |
| **합계** | **~400 queries/sec** | **~0** (100% ↓) |

---

## 9. 파일 구조

```
backend/internal/service/pricesync/
├── service.go          # 기존 - ProcessTick 로직
├── manager.go          # 기존 - WS/REST 오케스트레이션
├── poller.go           # 기존 - REST Tier 폴링
├── priority_manager.go # 기존 - 우선순위 관리
├── coalescer.go        # 신규 - DB 쓰기 debounce
├── cache.go            # 신규 - In-memory 가격 캐시
├── broker.go           # 신규 - Pub/Sub
└── error.go            # 기존

backend/internal/api/handlers/
├── price_handler.go    # 기존 - REST API
└── price_stream.go     # 신규 - SSE 스트리밍
```

---

## 10. 구현 순서

| 순서 | 작업 | 우선순위 | 의존성 |
|------|------|---------|--------|
| 1 | Coalescer 구현 | P0 | 없음 |
| 2 | Cache 구현 | P0 | 없음 |
| 3 | Service에 Coalescer/Cache 통합 | P0 | 1, 2 |
| 4 | Broker 구현 | P1 | 3 |
| 5 | SSE Handler 구현 | P1 | 4 |
| 6 | Frontend usePriceStream 훅 | P1 | 5 |
| 7 | PriorityManager 어댑터 구현 | P2 | 3 |
