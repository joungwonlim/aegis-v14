# v14 아키텍처 개선점 (Architecture Improvements)

> 설계 검토 후 성능 및 안정성 향상을 위한 개선 제안

**작성일**: 2026-01-13
**우선순위**: P0 (최우선) ~ P2 (보통)

---

## 📋 개요

현재 v14 아키텍처는 SSOT 원칙과 모듈 독립성 측면에서 매우 우수합니다. 다음 개선점들은 **속도(Latency)**와 **동시성 제어(Concurrency)** 최적화에 초점을 맞춥니다.

---

## 🔴 P0: 최우선 개선 (반드시 구현)

### 1. Execution ↔ Exit 간 Locked Qty 계산 로직

**문제점**: Race Condition으로 인한 중복 주문 위험

Exit Engine이 부분 체결 후 잔량을 계산하여 추가 청산 intent를 생성하는 순간, Execution이 나머지 물량의 체결 정보를 수신할 수 있습니다. 이로 인해 초과 매도 (Short Position 진입) 주문이 발생할 수 있습니다.

**개선안**: Available Qty 계산 시 Pending Orders 차감

```go
// Exit Engine - 가용 수량 계산
func (e *ExitEngine) GetAvailableQty(positionID uuid.UUID) (int64, error) {
    // 1. 현재 포지션 수량
    position := e.store.GetPosition(ctx, positionID)

    // 2. SUBMITTED 상태 주문의 수량 합계 (Locked Qty)
    pendingOrders := e.store.ListOrders(ctx, ListOrdersFilter{
        PositionID: positionID,
        Status:     []string{"NEW", "SUBMITTED", "PARTIAL_FILLED"},
    })

    lockedQty := int64(0)
    for _, order := range pendingOrders {
        lockedQty += order.Qty - order.FilledQty
    }

    // 3. 가용 수량 = 포지션 수량 - 잠긴 수량
    availableQty := position.Qty - lockedQty

    return max(availableQty, 0), nil
}

// Intent 생성 전 체크
if availableQty <= 0 {
    log.Warn("no available qty for exit", "position_id", positionID, "locked_qty", lockedQty)
    return nil // Skip intent creation
}
```

**효과**:
- 중복 주문 방지
- Short Position 진입 위험 제거
- 의도치 않은 포지션 꼬임 방지

---

## 🟡 P1: 우선 개선 (조속 구현 권장)

### 2. DB Polling 방식을 보완하는 이벤트 트리거 도입

**문제점**: DB 폴링으로 인한 Latency 누적

현재 모든 모듈이 PostgreSQL 테이블을 통해 통신합니다:
- PriceSync → prices_best (쓰기) → Strategy (읽기, 1~3초 주기)
- Strategy → order_intents (쓰기) → Execution (읽기, 1~3초 주기)

급락장에서 PriceSync 쓰기 → Strategy 판단 → Intent 생성 → Execution 주문까지 **수십ms ~ 수백ms 지연**이 발생합니다.

**개선안**: PostgreSQL NOTIFY/LISTEN 또는 Redis Pub/Sub

#### 옵션 A: PostgreSQL NOTIFY/LISTEN (권장)

```sql
-- Intent 생성 시 자동 알림
CREATE OR REPLACE FUNCTION notify_new_intent()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'NEW' THEN
        PERFORM pg_notify('new_intent', json_build_object(
            'intent_id', NEW.intent_id,
            'intent_type', NEW.intent_type,
            'symbol', NEW.symbol
        )::text);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_new_intent
AFTER INSERT ON trade.order_intents
FOR EACH ROW
EXECUTE FUNCTION notify_new_intent();
```

```go
// Execution Service - LISTEN 대기
func (s *ExecutionService) StartIntentListener(ctx context.Context) {
    conn, _ := s.db.Conn(ctx)
    _, _ = conn.ExecContext(ctx, "LISTEN new_intent")

    for {
        notification := waitForNotification(conn) // Blocking
        intentID := parseIntentID(notification.Extra)

        // 즉시 처리 (10ms 이내)
        go s.ProcessIntent(ctx, intentID)
    }
}
```

**효과**:
- 주문 반응 속도: **1~3초 → 10ms 이내**
- DB 폴링 부하 감소
- 급락장 대응 속도 향상

#### 옵션 B: Redis Pub/Sub (향후 확장)

```go
// Strategy (Publisher)
func (s *ExitEngine) CreateIntent(ctx context.Context, intent *Intent) error {
    // 1. DB 저장 (영속성)
    s.store.InsertIntent(ctx, intent)

    // 2. Redis 발행 (신호)
    s.redis.Publish(ctx, "intents:new", intent.IntentID)

    return nil
}

// Execution (Subscriber)
func (s *ExecutionService) SubscribeIntents(ctx context.Context) {
    pubsub := s.redis.Subscribe(ctx, "intents:new")

    for msg := range pubsub.Channel() {
        intentID := msg.Payload
        go s.ProcessIntent(ctx, intentID)
    }
}
```

**DB vs Redis 비교**:

| 방식 | Latency | 복잡도 | 영속성 | 권장 |
|------|---------|--------|--------|------|
| PostgreSQL NOTIFY/LISTEN | 10~50ms | 낮음 | DB 보장 | ✅ 초기 구현 |
| Redis Pub/Sub | 1~10ms | 중간 | 별도 필요 | 향후 확장 |

---

### 3. Morning Rush Mode (가변 주기 루프)

**문제점**: 장 시작 시간대 대응 부족

한국 시장 09:00~09:01은 1초 사이에 ±3%가 움직입니다. Exit Engine 루프가 1~5초 주기이면 GAP_DOWN 대응이 늦습니다.

**개선안**: 시간대별 가변 주기

```go
func (e *ExitEngine) GetLoopInterval() time.Duration {
    now := time.Now()
    hour, min := now.Hour(), now.Minute()

    // 장 시작 구간 (08:55 ~ 09:10): 고속 모드
    if (hour == 8 && min >= 55) || (hour == 9 && min <= 10) {
        return 200 * time.Millisecond // 200ms
    }

    // 장 마감 구간 (15:15 ~ 15:30): 고속 모드
    if hour == 15 && min >= 15 && min <= 30 {
        return 500 * time.Millisecond // 500ms
    }

    // 일반 구간: 표준 모드
    return 2 * time.Second // 2초
}
```

**Pre-Queueing**: 09:00 이전 GAP_DOWN 계산

```go
// 08:59:50 시점에 전일 종가 대비 GAP 예측
func (e *ExitEngine) PreComputeGapDownCandidates(ctx context.Context) {
    positions := e.store.ListOpenPositions(ctx)

    for _, pos := range positions {
        lastClose := e.getPreviousClose(pos.Symbol)
        gapThreshold := lastClose * -0.05 // -5%

        // 09:00:00 첫 틱에서 즉시 발동 가능하도록 준비
        e.gapDownQueue[pos.PositionID] = GapDownCandidate{
            PositionID: pos.PositionID,
            Threshold:  gapThreshold,
            PreComputed: true,
        }
    }
}
```

**효과**:
- 시가 급변동 대응 속도 향상
- CPU 부하는 10분간만 증가 (허용 가능)

---

### 4. Redis 읽기 가속 (신중한 접근)

**문제점**: 고빈도 DB 읽기로 인한 병목

현재 PriceSync, Exit Engine, Execution Service가 PostgreSQL에서 반복적으로 읽기 작업을 수행합니다:
- `prices_best` 조회 (Strategy 모듈, 1~3초마다)
- `positions` 조회 (Exit/Reentry, 2초마다)

**⚠️ SSOT 원칙**: Redis는 SSOT가 아니다

- **SSOT**: PostgreSQL의 사실 테이블 (prices_best, positions, fills)
- **Redis**: 성능을 위한 파생/복제/뷰(view) 레이어
- Redis는 "정답"이 아니라 **"빠른 힌트"** 또는 **"읽기 가속"**

**개선안**: 안전한 값만 제한적으로 캐싱

#### A. 적극 권장: prices_best 읽기 가속 (비교적 안전)

**안전한 이유**:
- 가격은 고빈도/실시간 값이라 "최신 1개만" 의미
- freshness 게이트로 Fail-Closed 가능
- 타임스탬프 기반 검증 가능

**패턴 1: 단일 Writer (PriceSync만 갱신)**

```go
// PriceSync가 prices_best 업데이트 시 Redis 동시 갱신 (단일 Writer)
func (ps *PriceSync) UpdateBestPrice(ctx context.Context, symbol string, price *BestPrice) error {
    // 1. DB 업데이트 (SSOT)
    ps.db.UpsertBestPrice(ctx, symbol, price)

    // 2. Redis 캐시 동시 갱신 (타임스탬프 포함!)
    key := fmt.Sprintf("price:best:%s", symbol)
    cacheData := map[string]interface{}{
        "bid":       price.Bid,
        "ask":       price.Ask,
        "best_ts":   price.BestTs.Unix(), // 타임스탬프 필수!
        "source":    price.Source,
        "is_stale":  price.IsStale,
    }
    ps.redis.HSet(ctx, key, cacheData)
    ps.redis.Expire(ctx, key, 5*time.Second) // TTL 5초

    return nil
}
```

**패턴 2: Exit Engine은 읽기만 + 타임스탬프 검증**

```go
// Exit Engine - Redis에서 가격 조회 (검증 포함)
func (e *ExitEngine) GetBestPrice(ctx context.Context, symbol string) (*BestPrice, error) {
    key := fmt.Sprintf("price:best:%s", symbol)
    cached := e.redis.HGetAll(ctx, key).Val()

    if len(cached) == 0 {
        // Cache Miss → DB 조회 (Fallback)
        log.Warn("price cache miss, fallback to DB", "symbol", symbol)
        return e.db.GetBestPrice(ctx, symbol)
    }

    bestTs := time.Unix(parseInt64(cached["best_ts"]), 0)
    age := time.Since(bestTs)

    // 검증: 10초 이상 오래되면 stale 판정
    if age > 10*time.Second || cached["is_stale"] == "true" {
        log.Warn("stale price detected", "symbol", symbol, "age", age)
        return nil, ErrStalePrice // Fail-Closed (청산 보류)
    }

    return &BestPrice{
        Bid:     parseFloat64(cached["bid"]),
        Ask:     parseFloat64(cached["ask"]),
        BestTs:  bestTs,
        Source:  cached["source"],
        IsStale: cached["is_stale"] == "true",
    }, nil
}
```

---

#### B. 신중: positions.qty/avg_price 캐싱 (위험도 높음)

**⚠️ 위험한 이유**:
- qty/avg_price는 **Execution reconcile 기반 SSOT**
- 부분 체결/정정/취소/수동 매매 시 Redis ↔ DB 불일치 위험
- Exit Engine이 잘못된 qty로 과다청산 intent 생성 가능

**v10 실제 사고 사례** (자세한 내용은 `exit-engine.md` 참고):
1. **평단가 캐시 불일치**: 추가 매수로 평단가 71,667원 → 캐시는 70,000원 → 손익률 착오로 조기 청산
2. **가격 캐시 Stale**: 실제 -5.0% 손실인데 캐시는 -0.5% → SL 미발동 → 손실 확대
3. **부분 체결 중 수량 불일치**: 50주 체결 후 100주 재청산 Intent → 과다 청산 (숏 진입!)

**안전 패턴: Intent 생성 직전 DB 재확인 필수**

```go
// Exit Engine - 루프에서는 Redis 조회, Intent 생성 직전 DB 재확인
func (e *ExitEngine) EvaluatePosition(ctx context.Context, positionID uuid.UUID) error {
    // 1. Redis에서 빠르게 후보 평가 (힌트)
    cached := e.redis.HGetAll(ctx, fmt.Sprintf("position:%s", positionID)).Val()
    cachedQty := parseInt64(cached["qty"])
    cachedVersion := parseInt64(cached["version"])

    if cachedQty <= 0 {
        return nil // 이미 청산된 것으로 보임 (스킵)
    }

    // 2. 청산 조건 평가 (Redis 기반)
    shouldExit := e.evaluateExitCondition(ctx, positionID, cachedQty)
    if !shouldExit {
        return nil
    }

    // 3. ⚠️ Intent 생성 직전: DB에서 사실(SSOT) 재확인!
    dbPosition := e.db.GetPosition(ctx, positionID)

    // 4. 버전 불일치 또는 수량 변경 감지
    if dbPosition.Version != cachedVersion {
        log.Warn("position version mismatch, re-evaluating",
            "position_id", positionID,
            "cached_version", cachedVersion,
            "db_version", dbPosition.Version)

        // Redis 갱신 (Execution이 업데이트했을 가능성)
        e.refreshPositionCache(ctx, positionID, dbPosition)

        // 이번 tick에서는 스킵 (다음 tick에서 재평가)
        return nil
    }

    if dbPosition.Qty != cachedQty {
        log.Error("position qty mismatch - cache drift detected!",
            "position_id", positionID,
            "cached_qty", cachedQty,
            "db_qty", dbPosition.Qty)

        // 캐시 무효화 및 스킵
        e.redis.Del(ctx, fmt.Sprintf("position:%s", positionID))
        return ErrCacheDrift
    }

    // 5. 검증 통과 → DB 사실 기준으로 Intent 생성
    return e.createExitIntent(ctx, positionID, dbPosition.Qty, dbPosition.AvgPrice)
}
```

**Writer: Execution만 갱신 (holdings reconcile 결과)**

```go
// Execution Service - holdings reconcile 후 DB + Redis 동시 갱신
func (s *ExecutionService) ReconcilePosition(ctx context.Context, holding *KISHolding) error {
    // 1. DB 업데이트 (SSOT, version 증가)
    s.db.UpdatePosition(ctx, UpdatePositionParams{
        PositionID: holding.PositionID,
        Qty:        holding.Qty,
        AvgPrice:   holding.AvgPrice,
        // version은 DB에서 자동 증가
    })

    // 2. Redis 캐시 갱신 (타임스탬프 + 버전 포함)
    updated := s.db.GetPosition(ctx, holding.PositionID) // version 최신화
    key := fmt.Sprintf("position:%s", holding.PositionID)
    s.redis.HSet(ctx, key, map[string]interface{}{
        "qty":         updated.Qty,
        "avg_price":   updated.AvgPrice,
        "version":     updated.Version,    // 버전 필수!
        "updated_ts":  updated.UpdatedTs.Unix(),
    })
    s.redis.Expire(ctx, key, 10*time.Second) // TTL 10초

    return nil
}
```

---

#### C. 적극 권장: 기타 안전한 캐싱 대상

| 데이터 | 안전도 | TTL | Writer | 효과 |
|--------|--------|-----|--------|------|
| `prices_best` + best_ts | ✅ 높음 | 5초 | PriceSync만 | DB 읽기 90% 감소 |
| `freshness` | ✅ 높음 | 5초 | PriceSync만 | DB 읽기 90% 감소 |
| OPEN 포지션 리스트 | ✅ 높음 | 10초 | Exit/Execution | DB 읽기 80% 감소 |
| `exit_profiles` | ✅ 매우높음 | 1시간 | Admin만 | DB 읽기 99% 감소 |
| `exit_control`, `reentry_control` | ✅ 매우높음 | 10초 | Admin만 | DB 읽기 99% 감소 |
| `positions.qty/avg_price` + version | ⚠️ 중간 | 10초 | Execution만 | **Intent 직전 DB 재확인 필수** |

---

#### D. 비추천: 사실 로그성 데이터

| 데이터 | 이유 |
|--------|------|
| `fills`, `orders` | 정합성 요구 높고, 읽기 패턴 낮음 |
| `exit_events` | Execution 생성 SSOT, 캐싱 불필요 |
| `reentry_candidates` | Reentry 생성 SSOT, 캐싱 불필요 |

---

#### E. 안전 원칙 4가지 (필수 준수)

**원칙 1: 단일 Writer 강제**
- 가격 캐시: PriceSync만 Redis 갱신
- 포지션 캐시: Execution만 Redis 갱신 (holdings reconcile 결과로만)
- Exit Engine은 읽기만 (그리고 결정 직전에 DB로 재확인)

**원칙 2: 캐시는 반드시 버전/타임스탬프 동반**
- `best_price`, `best_ts`, `source`, `is_stale`
- `pos_qty`, `pos_avg_price`, `pos_version`, `updated_ts`
- Exit Engine은 `best_ts`/`pos_version`이 충분히 최신인지 확인 후 사용

**원칙 3: 결정 직전 "DB 사실 재확인" 최소 1회**
- 루프에서 Redis로 빠르게 후보 평가
- **Intent 생성 직전에만 DB에서 positions.qty/avg_price/version 재조회**
- Mismatch면 이번 tick에서 스킵하거나 재평가

**원칙 4: TTL/Fail-Closed**
- 가격 캐시 TTL: 5초 (또는 best_ts 기반 stale 판정)
- 포지션 캐시 TTL: 10초 (또는 version 기반)
- TTL 초과/검증 실패 시 Fail-Closed (청산 생성 보류)

---

#### F. 효과 및 주의사항

**효과** (안전한 패턴 준수 시):
- DB 읽기 부하: **60~80% 감소** (prices_best, freshness, profiles 중심)
- 응답 속도: PostgreSQL 1~3ms → Redis 0.1~0.3ms (10배 향상)
- DB max_connections 여유 확보

**⚠️ 주의사항**:
- **SSOT는 PostgreSQL** (Redis는 파생/복제/뷰 레이어)
- **qty/avg_price 캐싱은 신중** (Intent 직전 DB 재확인 필수)
- **TTL 설정 필수** (stale data 방지)
- **단일 Writer 패턴** (레이스/드리프트 방지)
- **타임스탬프/버전 없이 캐싱 금지** (최신성 검증 불가능)

**오류 확률을 올리는 금지 패턴**:
- ❌ Exit Engine이 Redis의 qty/avg_price를 사실로 믿고 계산
- ❌ DB보다 Redis가 앞서는 구조
- ❌ 여러 곳에서 동일 캐시 갱신 (레이스 발생)
- ❌ TTL 없는 캐시 (죽은 값 영구 보존)
- ❌ 타임스탬프/버전 없는 값만 캐시 (stale 판단 불가)

---

### 5. Pick Pipeline Event-Driven Router

**문제점**: Router 스케줄러(1분) 지연

뉴스/LLM(3001) 전략은 속보 발생 시 즉시 진입해야 알파가 있습니다. 1분 대기는 기회 상실입니다.

**개선안**: POST /ingest/picks 호출 시 즉시 Router 트리거

```go
// POST /api/picks/ingest
func (h *PicksHandler) IngestPicks(c *gin.Context) {
    var picks []Pick
    c.ShouldBindJSON(&picks)

    // 1. DB 저장
    h.store.InsertPicks(ctx, picks)

    // 2. 즉시 Router 트리거 (async)
    go h.router.ProcessNewPicks(ctx, picks)

    c.JSON(200, gin.H{"status": "queued"})
}

// Router - On-Demand 처리
func (r *Router) ProcessNewPicks(ctx context.Context, picks []Pick) {
    for _, pick := range picks {
        // Gate 검증
        if !r.gate1.Check(pick) { continue }
        if !r.gate2.Check(pick) { continue }
        if !r.gate3.Check(pick) { continue }

        // Intent 생성
        r.createIntent(ctx, pick)
    }
}
```

**스케줄러는 Fallback으로 유지**:

```go
// 1분마다 "놓친 picks" 처리 (안전장치)
func (r *Router) ScheduledFallback() {
    ticker := time.NewTicker(1 * time.Minute)

    for range ticker.C {
        orphanPicks := r.store.ListUnprocessedPicks(ctx, since=1*time.Minute)
        if len(orphanPicks) > 0 {
            log.Warn("found orphan picks", "count", len(orphanPicks))
            r.ProcessNewPicks(ctx, orphanPicks)
        }
    }
}
```

**효과**:
- 뉴스/이벤트 전략 반응 속도: **1분 → 즉시**
- 스케줄러는 안전장치로만 사용

---

## 🟢 P2: 보통 개선 (향후 검토)

### 6. KIS API Circuit Breaker & Fallback

**문제점**: KIS API 장애 시 시스템 전체 마비

KIS REST API가 5분 이상 장애 나면 시세 수신도, 주문도 불가능합니다.

**개선안**: Circuit Breaker 패턴

```go
type CircuitBreaker struct {
    state         string // CLOSED | OPEN | HALF_OPEN
    failureCount  int
    lastFailureTs time.Time
    threshold     int    // 연속 실패 임계값
    timeout       time.Duration // OPEN 상태 유지 시간
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == "OPEN" {
        if time.Since(cb.lastFailureTs) > cb.timeout {
            cb.state = "HALF_OPEN"
        } else {
            return ErrCircuitOpen
        }
    }

    err := fn()
    if err != nil {
        cb.failureCount++
        cb.lastFailureTs = time.Now()

        if cb.failureCount >= cb.threshold {
            cb.state = "OPEN"
            log.Error("circuit breaker opened", "service", "KIS_API")
            // 알람 발송
            sendAlert("KIS API 장애 감지 - Circuit Breaker OPEN")
        }
        return err
    }

    cb.failureCount = 0
    cb.state = "CLOSED"
    return nil
}
```

**Emergency Flatten 정책**:

```yaml
emergency_flatten:
  enabled: false
  trigger_conditions:
    - kis_api_down_minutes: 5
    - all_price_sources_stale: true
  action:
    - send_critical_alert: ["slack", "sms", "email"]
    - recommend_manual_action: "MTS로 수동 청산 필요"
    - auto_flatten: false  # 기본값: 사람 판단 대기
```

**Rate Limit 분산**: AppKey 분리

```yaml
kis_accounts:
  - app_key: "조회용_KEY_001"
    secret: "..."
    usage: ["price_sync", "holdings_sync"]

  - app_key: "주문용_KEY_002"
    secret: "..."
    usage: ["order_submit", "order_cancel"]
```

**효과**:
- KIS API 장애 시 즉시 감지 및 알람
- Rate Limit 분산으로 할당량 확보

---

### 7. PostgreSQL Connection Pooling 최적화

**문제점**: 모듈별 DB Connection Pool 관리

향후 스케일 아웃 시 max_connections 한계에 도달할 수 있습니다.

**개선안**: PgBouncer 도입 (향후)

```
Application Modules
    ↓
PgBouncer (Connection Pooler)
    ↓
PostgreSQL (10~20 connections)
```

**설정 예시** (pgbouncer.ini):

```ini
[databases]
aegis = host=localhost port=5432 dbname=aegis

[pgbouncer]
pool_mode = transaction  # 트랜잭션 종료 시 연결 반환
max_client_conn = 1000   # 애플리케이션 연결 수
default_pool_size = 20   # DB 연결 풀 크기
reserve_pool_size = 5    # 예비 연결
```

**효과**:
- 애플리케이션 연결 수: 무제한
- DB 실제 연결 수: 20개로 제한 (효율적)

---

## 📊 우선순위 요약

| 순위 | 개선점 | 예상 공수 | 효과 | 주의사항 |
|------|--------|----------|------|----------|
| **P0** | Locked Qty 계산 로직 | 1일 | 중복 주문 방지 (Critical) | - |
| **P1** | NOTIFY/LISTEN 이벤트 | 2일 | Latency 90% 감소 | PostgreSQL 트리거 |
| **P1** | Morning Rush Mode | 1일 | 시가 급변동 대응 | 10분간 CPU 증가 |
| **P1** | Redis 읽기 가속 (신중) | 3일 | DB 읽기 60~80% 감소 | **SSOT 원칙 준수, Intent 직전 DB 재확인 필수** |
| **P1** | Event-Driven Router | 1일 | 뉴스 전략 즉시 반응 | Fallback 필요 |
| **P2** | Circuit Breaker | 2일 | API 장애 대응 | 알람 필수 |
| **P2** | PgBouncer 도입 | 1일 | 향후 스케일링 대비 | Transaction Pool Mode |

---

## 🔗 관련 문서

- [system-overview.md](./architecture/system-overview.md) - 전체 시스템 아키텍처
- [execution-service.md](./modules/execution-service.md) - Execution Service 설계
- [exit-engine.md](./modules/exit-engine.md) - Exit Engine 설계
- [pick-to-execution-pipeline.md](./architecture/pick-to-execution-pipeline.md) - Pick Pipeline 설계

---

**Version**: v14.0.0-improvements
**Author**: Architecture Review
**Last Updated**: 2026-01-13
