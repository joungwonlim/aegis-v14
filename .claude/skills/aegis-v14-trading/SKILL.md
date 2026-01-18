# Aegis v14 Trading System Development Skill

**Version**: 1.0.0
**Author**: Aegis Development Team
**Purpose**: 트레이딩 시스템 개발 시 금융 도메인 지식과 실시간 처리 Best Practices 적용

---

## 🎯 Skill 활성화 시점

다음 작업 시 자동으로 이 skill이 참조됩니다:
- Order 처리 로직 작성
- Position 관리 코드 작성
- Exit Engine 트리거 구현
- 실시간 가격 데이터 처리
- KIS API 연동 코드 작성
- 금융 계산 (PnL, 수익률 등)

---

## 📋 트레이딩 시스템 핵심 원칙

### 1. 멱등성 (Idempotency) 보장

**원칙**: 동일한 요청을 여러 번 실행해도 결과가 같아야 함

```go
// ❌ BAD - 중복 실행 시 데이터 중복
func CreateOrder(order *Order) error {
    return db.Insert(order)
}

// ✅ GOOD - Upsert로 멱등성 보장
func UpsertOrder(order *Order) error {
    query := `
        INSERT INTO orders (order_id, ...) VALUES ($1, ...)
        ON CONFLICT (order_id) DO UPDATE SET
            status = EXCLUDED.status,
            updated_ts = EXCLUDED.updated_ts
    `
    return db.Exec(query, order)
}
```

### 2. FK 제약 순서 준수

**원칙**: 외래키 제약을 위반하지 않도록 삽입 순서 보장

```go
// ❌ BAD - Fill 먼저 삽입 (FK 위반 가능)
func ProcessFill(fill *Fill) error {
    if err := fillRepo.Create(fill); err != nil {
        return err
    }
    return ensureOrderExists(fill.OrderID)
}

// ✅ GOOD - Order 먼저 확인/생성 후 Fill 삽입
func ProcessFill(fill *Fill) error {
    if err := ensureOrderExists(fill.OrderID); err != nil {
        return err
    }
    return fillRepo.UpsertFill(fill)
}
```

### 3. Null vs Zero Value 구분

**원칙**: 의미 없는 Zero Value는 NULL로 저장

```go
// ❌ BAD - Zero UUID를 그대로 삽입 (FK 위반)
order := &Order{
    OrderID:  "123",
    IntentID: uuid.UUID{}, // 00000000-0000-0000-0000-000000000000
}
db.Insert(order)

// ✅ GOOD - Zero UUID를 NULL로 변환
var intentID interface{}
if order.IntentID == (uuid.UUID{}) {
    intentID = nil
} else {
    intentID = order.IntentID
}
db.Exec(query, order.OrderID, intentID)
```

### 4. 재시도 로직에서 멱등성 보장

**원칙**: 재시도 시 중복 생성 방지

```go
// ❌ BAD - 재시도 시 중복 Intent 생성
func CreateExitIntent(positionID uuid.UUID) error {
    intent := &Intent{
        IntentID:   uuid.New(),
        PositionID: positionID,
        Type:       "EXIT_FULL",
    }
    return intentRepo.Create(intent)
}

// ✅ GOOD - 중복 체크 후 생성
func CreateExitIntent(positionID uuid.UUID, reasonCode string) error {
    // Check if intent already exists
    exists, err := intentRepo.ExistsForPosition(positionID, reasonCode)
    if err != nil {
        return err
    }
    if exists {
        return nil // Already created
    }

    intent := &Intent{
        IntentID:   uuid.New(),
        PositionID: positionID,
        Type:       "EXIT_FULL",
        ReasonCode: reasonCode,
    }
    return intentRepo.Create(intent)
}
```

---

## 🔒 금융 계산 정확성

### 1. Decimal 사용 (Float 금지)

**원칙**: 금융 계산은 반드시 `decimal.Decimal` 사용 (정밀도 보장)

```go
// ❌ BAD - Float64 사용 (정밀도 손실)
price := 10500.5
qty := 10
totalValue := price * float64(qty) // 부정확

// ✅ GOOD - Decimal 사용
price := decimal.NewFromFloat(10500.5)
qty := decimal.NewFromInt(10)
totalValue := price.Mul(qty) // 정확
```

### 2. PnL 계산 공식

**원칙**: 매입가와 현재가 기준으로 정확한 PnL 계산

```go
// ✅ PnL 계산 표준 공식
func CalculatePnL(avgPrice, currentPrice decimal.Decimal, qty int64) (pnl decimal.Decimal, pnlPct float64) {
    // 매입 총액
    entryValue := avgPrice.Mul(decimal.NewFromInt(qty))

    // 현재 평가액
    currentValue := currentPrice.Mul(decimal.NewFromInt(qty))

    // 손익
    pnl = currentValue.Sub(entryValue)

    // 손익률 (%)
    if !entryValue.IsZero() {
        pnlPct, _ = pnl.Div(entryValue).Mul(decimal.NewFromInt(100)).Float64()
    }

    return pnl, pnlPct
}
```

### 3. 수수료 및 세금 반영

**원칙**: 실제 수익은 매도 시 수수료/세금 차감 후 계산

```go
// ✅ Real PnL (HTS-style with fees)
func CalculateRealPnL(holding *Holding) (decimal.Decimal, float64) {
    // Simple PnL (without fees)
    simplePnl := holding.CurrentPrice.Sub(holding.AvgPrice).Mul(decimal.NewFromInt(holding.Qty))

    // Sell amount
    sellAmount := holding.CurrentPrice.Mul(decimal.NewFromInt(holding.Qty))

    // Fee rate by market
    var feeRate decimal.Decimal
    switch holding.Market {
    case "KOSPI":
        feeRate = decimal.NewFromFloat(0.00315) // 0.315%
    case "KOSDAQ":
        feeRate = decimal.NewFromFloat(0.00245) // 0.245%
    }

    // Calculate fees
    fees := sellAmount.Mul(feeRate)

    // Real PnL = Simple PnL - Fees
    realPnl := simplePnl.Sub(fees)

    // Real PnL %
    realPnlPct, _ := realPnl.Div(holding.AvgPrice.Mul(decimal.NewFromInt(holding.Qty))).Mul(decimal.NewFromInt(100)).Float64()

    return realPnl, realPnlPct
}
```

---

## 🚨 실시간 처리 Best Practices

### 1. WebSocket 재연결 전략

**원칙**: 지수 백오프 + 구독 복원

```go
// ✅ GOOD - Exponential backoff with subscription restoration
func (c *WebSocketClient) reconnect() error {
    backoff := 2 * time.Second
    maxBackoff := 60 * time.Second
    maxAttempts := 20

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        if err := c.connect(); err == nil {
            // Restore subscriptions after successful reconnect
            c.restoreSubscriptions()
            return nil
        }

        time.Sleep(backoff)
        backoff = min(backoff*2, maxBackoff)
    }

    return fmt.Errorf("max reconnect attempts reached")
}

func (c *WebSocketClient) restoreSubscriptions() {
    // Wait for connection to stabilize
    time.Sleep(2 * time.Second)

    // Re-subscribe to all symbols
    for _, symbol := range c.symbols {
        c.Subscribe(symbol)
        time.Sleep(200 * time.Millisecond) // Throttle subscriptions
    }
}
```

### 2. 가격 데이터 경합 방지

**원칙**: Mutex로 동시 접근 보호

```go
// ✅ GOOD - Thread-safe price updates
type PriceManager struct {
    prices map[string]decimal.Decimal
    mu     sync.RWMutex
}

func (pm *PriceManager) UpdatePrice(symbol string, price decimal.Decimal) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    pm.prices[symbol] = price
}

func (pm *PriceManager) GetPrice(symbol string) (decimal.Decimal, bool) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    price, ok := pm.prices[symbol]
    return price, ok
}
```

### 3. Rate Limiting 준수

**원칙**: KIS API 호출 제한 (1분당 1회) 준수

```go
// ✅ GOOD - Token caching with rate limiting
type KISAdapter struct {
    token        string
    tokenExpiry  time.Time
    tokenMutex   sync.Mutex
    lastTokenReq time.Time
}

func (a *KISAdapter) GetAccessToken() (string, error) {
    a.tokenMutex.Lock()
    defer a.tokenMutex.Unlock()

    // Return cached token if valid
    if time.Now().Before(a.tokenExpiry) {
        return a.token, nil
    }

    // Rate limit: 1 request per minute
    elapsed := time.Since(a.lastTokenReq)
    if elapsed < time.Minute {
        return "", fmt.Errorf("rate limit: wait %v", time.Minute-elapsed)
    }

    // Fetch new token
    token, err := a.fetchTokenFromAPI()
    if err != nil {
        return "", err
    }

    a.token = token
    a.tokenExpiry = time.Now().Add(24 * time.Hour)
    a.lastTokenReq = time.Now()

    return token, nil
}
```

---

## 🏗️ 아키텍처 패턴

### 1. Repository 패턴에서 Nil 체크

**원칙**: 주입된 dependency는 반드시 사용해야 함

```go
// ❌ BAD - Repository를 주입받고도 사용 안 함
func NewProfileResolver(repo ProfileRepository) *ProfileResolver {
    return &ProfileResolver{repo: repo}
}

func (r *ProfileResolver) Resolve(profileID uuid.UUID) (*Profile, error) {
    // TODO: Implement later
    return defaultProfile, nil // 항상 default 반환 (repo 미사용)
}

// ✅ GOOD - Repository를 실제로 사용
func (r *ProfileResolver) Resolve(profileID uuid.UUID) (*Profile, error) {
    // Try to load from DB
    profile, err := r.repo.GetProfile(profileID)
    if err == nil {
        return profile, nil
    }

    // Fallback to default if not found
    if errors.Is(err, ErrProfileNotFound) {
        return defaultProfile, nil
    }

    return nil, fmt.Errorf("failed to resolve profile: %w", err)
}
```

### 2. Context 전파

**원칙**: 모든 blocking I/O 함수는 context를 첫 번째 인자로 받아야 함

```go
// ❌ BAD - Context 없음 (취소/타임아웃 불가)
func FetchPrices(symbols []string) (map[string]decimal.Decimal, error) {
    // ...
}

// ✅ GOOD - Context 전파
func FetchPrices(ctx context.Context, symbols []string) (map[string]decimal.Decimal, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // ...
}
```

---

## ⚠️ 금지 패턴

### 1. TODO로 안전장치 미루기

```go
// ❌ PROHIBITED - HardStop을 TODO로 남김
func evaluateTriggers() {
    // TODO: Implement HardStop later

    if controlMode == PAUSE_ALL {
        return nil
    }

    // Evaluate regular triggers...
}
```

**이유**: HardStop은 최후의 안전장치로, 구현을 미루면 급락 시 손실 확대

**해결**: 즉시 구현하거나 별도 이슈 생성

### 2. 인터페이스만 구현하고 내부는 빈 껍데기

```go
// ❌ PROHIBITED - 겉만 구현
type ProfileResolver struct {
    repo ProfileRepository // 주입받았지만 사용 안 함
}

func (r *ProfileResolver) Resolve(profileID uuid.UUID) (*Profile, error) {
    // For now, return default
    return defaultProfile, nil
}
```

**이유**: Repository를 주입받았으면 실제로 사용해야 함

**해결**: DB에서 실제로 로드하거나, 미구현 시 명시적 에러 반환

### 3. Quantity = 0인 Position 평가

```go
// ❌ BAD - qty=0인 position도 평가 시도
func EvaluateAllPositions() {
    for _, pos := range positions {
        trigger := evaluateTriggers(pos) // qty=0이어도 평가
        if trigger != nil {
            createIntent(pos, trigger)
        }
    }
}

// ✅ GOOD - qty=0은 스킵
func EvaluateAllPositions() {
    for _, pos := range positions {
        if pos.Qty == 0 {
            continue // Skip empty positions
        }

        trigger := evaluateTriggers(pos)
        if trigger != nil {
            createIntent(pos, trigger)
        }
    }
}
```

---

## 📊 로깅 Best Practices

### 1. 구조화된 로깅

```go
// ❌ BAD - 문자열 로깅
log.Printf("Order created: %s, qty: %d", orderID, qty)

// ✅ GOOD - 구조화된 로깅 (zerolog)
log.Info().
    Str("order_id", orderID).
    Int64("qty", qty).
    Str("status", order.Status).
    Msg("Order created")
```

### 2. 로그 레벨 적절히 사용

```go
// DEBUG: 상세 추적용 (프로덕션에서는 꺼짐)
log.Debug().Str("symbol", symbol).Msg("Evaluating triggers")

// INFO: 정상 동작 이벤트
log.Info().Str("order_id", orderID).Msg("Order filled")

// WARN: 예상 가능한 오류 (시스템 계속 작동)
log.Warn().Err(err).Msg("Failed to load profile, fallback to default")

// ERROR: 예상 못한 오류 (기능 실패)
log.Error().Err(err).Str("order_id", orderID).Msg("Failed to create order")
```

---

## 🧪 테스트 전략

### 1. 멱등성 테스트

```go
func TestUpsertOrder_Idempotent(t *testing.T) {
    order := &Order{OrderID: "TEST001", Qty: 100}

    // First insert
    err := repo.UpsertOrder(ctx, order)
    require.NoError(t, err)

    // Second insert (same order_id)
    order.Qty = 150
    err = repo.UpsertOrder(ctx, order)
    require.NoError(t, err)

    // Verify: should update, not duplicate
    result, err := repo.GetOrder(ctx, "TEST001")
    require.NoError(t, err)
    assert.Equal(t, int64(150), result.Qty)
}
```

### 2. 금융 계산 정확성 테스트

```go
func TestCalculatePnL_Accuracy(t *testing.T) {
    avgPrice := decimal.NewFromFloat(10000.0)
    currentPrice := decimal.NewFromFloat(11000.0)
    qty := int64(10)

    pnl, pnlPct := CalculatePnL(avgPrice, currentPrice, qty)

    // Expected: (11000 - 10000) * 10 = 10000
    assert.Equal(t, "10000", pnl.String())

    // Expected: 10000 / 100000 * 100 = 10%
    assert.InDelta(t, 10.0, pnlPct, 0.001)
}
```

---

## 🔍 디버깅 팁

### 1. FK 제약 위반 디버깅

```bash
# 1. 어떤 FK가 문제인지 확인
ERROR: insert or update on table "fills" violates foreign key constraint "fills_order_id_fkey"

# 2. 해당 order_id가 실제로 존재하는지 확인
SELECT * FROM trade.orders WHERE order_id = '0026604300';

# 3. 없으면 ensureOrderExists 호출 누락 확인
```

### 2. WebSocket 재연결 무한 루프 디버깅

```go
// 로그에 backoff 값 출력하여 패턴 확인
log.Info().
    Int("attempt", attempt).
    Dur("backoff", backoff).
    Msg("[WS] Attempting reconnect...")

// 장중 시간인지 확인
// 장마감 후(15:30~)는 정상적으로 연결 불가
```

---

## ✅ 체크리스트

새 코드 작성 시 확인:

- [ ] 멱등성 보장 (Upsert 패턴 사용)
- [ ] FK 제약 순서 준수 (부모 레코드 먼저 확인/생성)
- [ ] Null vs Zero Value 구분 (의미 없는 값은 NULL)
- [ ] Decimal 사용 (Float 금지)
- [ ] Thread-safe (Mutex 사용)
- [ ] Context 전파 (blocking I/O)
- [ ] Rate limiting 준수 (KIS API)
- [ ] 구조화된 로깅
- [ ] 에러 처리 (wrap with context)
- [ ] TODO로 안전장치 미루기 금지

---

**이 Skill을 사용하면 트레이딩 시스템 특유의 함정(FK 순서, 멱등성, 정밀도)을 자동으로 피할 수 있습니다.**
