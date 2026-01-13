# Exit Engine 운영 플레이북 (If–Then)

> **목적**: Exit Engine을 실제 운영하면서 마주하는 상황별 대응 절차
> **원칙**: 전역 제어로 사고를 막고 → 종목 오버라이드로 국소 조정 → 기본 프로파일을 마지막에 조정

**Last Updated**: 2026-01-13

---

## 📋 목차

1. [조정 우선순위 (항상 이 순서)](#0-조정-우선순위-항상-이-순서)
2. [손실 상황 대응](#1-손실-상황loss--더-큰-손실-방지-중심)
3. [수익 상황 대응](#2-수익-상황profit--수익-확정--추세-최대화-중심)
4. [숫자 조정 규칙](#3-숫자-조정에-대한-실무-규칙과도한-튜닝-방지)
5. [실제 운영 명령 예시](#4-운영자가-실제로-하는-명령-예시)
6. [가장 중요한 안전 규칙](#5-가장-중요한-안전-규칙캐시redis-포함)
7. [긴급 상황 매뉴얼](#6-긴급-상황-매뉴얼)
8. [모니터링 지표](#7-모니터링-지표)
9. [롤백 절차](#8-롤백-절차)
10. [변경 이력 기록](#9-변경-이력-기록-필수)

---

## 0) 조정 우선순위 (항상 이 순서)

**⚠️ CRITICAL**: 역순으로 하면 전체 시스템이 흔들립니다.

```
1순위: exit_control.mode로 즉시 리스크 차단
       ↓
2순위: symbol override(특정 종목만)로 미세 조정
       ↓
3순위: default profile(전체) 변경은 마지막
```

### 왜 이 순서인가?

| 순서 | 도구 | 영향 범위 | 롤백 난이도 | 리스크 |
|------|------|----------|------------|--------|
| 1순위 | exit_control.mode | 전체 | 쉬움 (1줄 UPDATE) | 낮음 |
| 2순위 | symbol override | 종목 단위 | 보통 (해당 종목만) | 보통 |
| 3순위 | profile 수정 | 전체 또는 그룹 | 어려움 (숫자 복원) | 높음 |

**금지 패턴**:
- ❌ profile부터 수정 → 전체 시스템 동작 변경 → 부작용 추적 어려움
- ❌ override 남발 → 종목별 설정 파편화 → 관리 불가

---

## 1) 손실 상황(Loss) – "더 큰 손실 방지" 중심

### 1.1. 가격 데이터 불안정 (Stale)

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 가격 stale/WS 불안정이 30초+ 지속 | `SELECT COUNT(*) FROM market.prices_best WHERE best_ts < NOW() - INTERVAL '30 seconds'` | **PAUSE_PROFIT**로 전환<br>(익절/트레일 차단, 손절만 유지)<br><br>심각하면 **PAUSE_ALL** + HardStop만 우회 허용 | 프로파일 변경보다 데이터 복구 우선<br><br>stale 동안은 평가 스킵(Fail-Closed) 유지 | stale 때 트레이드 의사결정은 오류 확률이 가장 큼<br><br>복구 후 RUNNING 복귀 |

**운영 명령**:
```sql
-- PAUSE_PROFIT (손절만 유지)
UPDATE trade.exit_control
SET mode='PAUSE_PROFIT',
    reason='실시간 가격 stale 30초+ 지속',
    updated_by='operator'
WHERE id=1;

-- 심각한 경우: PAUSE_ALL
UPDATE trade.exit_control
SET mode='PAUSE_ALL',
    reason='WS 완전 불안정 - HardStop만 우회',
    updated_by='operator'
WHERE id=1;
```

**복구 조건**:
```sql
-- freshness 정상화 확인 (30초간 stale < 5)
SELECT symbol, best_ts, NOW() - best_ts AS age
FROM market.prices_best
WHERE best_ts < NOW() - INTERVAL '10 seconds'
ORDER BY age DESC
LIMIT 10;

-- 정상이면 RUNNING 복귀
UPDATE trade.exit_control
SET mode='RUNNING',
    reason='가격 데이터 정상화 확인',
    updated_by='operator'
WHERE id=1;
```

---

### 1.2. SL2(전량 손절) 빈번

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 최근 20 트레이드 중 4회 이상 SL2 | `SELECT symbol, COUNT(*) as sl2_count FROM trade.exit_events WHERE action='SL2' AND created_at > NOW() - INTERVAL '7 days' GROUP BY symbol HAVING COUNT(*) >= 4` | 전체가 아니라 **해당 종목군부터 오버라이드 적용** | **high_beta** 종목이면:<br>- `sl2.max_pct`를 더 넓히고 (휩쏘 방지)<br>- 대신 `sl1.qty_pct`를 늘려 1차 방어 강화<br><br>예: `0.50→0.60` | SL을 타이트하게만 당기면 "손절 빈도"가 늘 수 있음 |

**운영 명령**:
```sql
-- 특정 종목만 high_beta로 전환 (휩쏘 방지)
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('012450', 'high_beta', 'SL2 빈번 - 변동성 확대 구간', 'operator')
ON CONFLICT(symbol) DO UPDATE
SET profile_id=EXCLUDED.profile_id,
    reason=EXCLUDED.reason,
    updated_ts=NOW();
```

**검증**:
```sql
-- 1주일 후 효과 검증
SELECT
    symbol,
    COUNT(*) FILTER (WHERE action='SL2') as sl2_count,
    AVG((exit_price - avg_price) / avg_price) as avg_pnl
FROM trade.exit_events
WHERE symbol='012450'
  AND created_at > NOW() - INTERVAL '7 days'
GROUP BY symbol;
```

---

### 1.3. SL1이 너무 자주 맞고 다시 상승 (휩쏘)

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 손절 후 1~2일 내 재상승 반복 | `SELECT symbol, exit_ts, exit_price, (SELECT MAX(close) FROM market.daily_bars WHERE symbol=e.symbol AND date > e.exit_ts::date AND date <= e.exit_ts::date + 2) as rebound_price FROM trade.exit_events e WHERE action='SL1'` | 해당 종목에 **symbol override**로만 완화 | - `atr.factor_max↑` 또는<br>- `sl1.min_pct`를 덜 타이트하게<br><br>예: `-2.0%→-2.5%`<br><br>trailing은 그대로 | SL 완화는 손실 확대 가능성도 있으니 **종목 한정** |

**운영 명령**:
```sql
-- 해당 종목만 SL 완화 (custom profile 생성 권장)
-- 또는 high_beta 적용
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('005930', 'high_beta', 'SL1 휩쏘 빈번', 'operator')
ON CONFLICT(symbol) DO UPDATE
SET profile_id=EXCLUDED.profile_id,
    reason=EXCLUDED.reason;
```

---

### 1.4. 손실이 커질 때 청산이 늦는 느낌

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 체결 지연/intent 누락 의심 | `SELECT * FROM trade.order_intents WHERE status='PENDING' AND created_at < NOW() - INTERVAL '5 minutes'` | **PAUSE_PROFIT** 유지<br><br>손절 경로만 점검:<br>- Execution/intent/체결 파이프라인 | 수치 조정보다<br>- idempotency/action_key<br>- intent insert<br>- fills reconcile<br><br>확인이 먼저 | **"로직" 문제가 아니라 "실행" 문제일 가능성 큼** |

**체크리스트**:
```sql
-- Intent 생성은 되는가?
SELECT position_id, symbol, action, created_at
FROM trade.order_intents
WHERE created_at > NOW() - INTERVAL '10 minutes'
ORDER BY created_at DESC
LIMIT 20;

-- Intent → Order 전환은 되는가?
SELECT oi.intent_id, oi.action, o.order_id, o.status
FROM trade.order_intents oi
LEFT JOIN trade.orders o ON oi.intent_id = o.intent_id
WHERE oi.created_at > NOW() - INTERVAL '10 minutes'
ORDER BY oi.created_at DESC;

-- 체결 지연은 없는가?
SELECT order_id, symbol, status, submitted_ts, filled_ts
FROM trade.orders
WHERE status IN ('SUBMITTED', 'PARTIAL_FILLED')
  AND submitted_ts < NOW() - INTERVAL '5 minutes'
ORDER BY submitted_ts;
```

---

### 1.5. 평단/수량이 변하는 타이밍에 과다/과소 청산 위험

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 추매/부분체결 타이밍에 이상 청산 | `SELECT position_id, version, qty, avg_price, updated_ts FROM trade.positions WHERE updated_ts > NOW() - INTERVAL '1 minute'` | Exit는 Redis 값을 참고하더라도<br>**intent 생성 직전 DB 재확인 강제** | **version 불일치 시**:<br>- "재평가" 또는<br>- "이번 tick skip"<br><br>정책 고정 | **개인 시스템에서 가장 현실적인 사고 포인트** |

**v14 방어 코드** (설계 문서 참고):
```go
// Intent 생성 직전 DB 재확인
var latestVersion int
var latestAvgPrice decimal.Decimal
e.db.QueryRow(ctx, `
    SELECT version, avg_price
    FROM trade.positions
    WHERE position_id = $1
`, pos.PositionID).Scan(&latestVersion, &latestAvgPrice)

// Version mismatch 감지
if latestVersion != snapshot.Version {
    log.Warn("평단가/수량 변경 감지 - 재평가 필요",
        "position", pos.PositionID,
        "cached_version", snapshot.Version,
        "db_version", latestVersion)
    return ErrPositionChanged  // 재평가 또는 skip
}

// DB 최신 값으로 intent 생성
createIntent(ctx, latestAvgPrice, latestVersion)
```

---

## 2) 수익 상황(Profit) – "수익 확정 + 추세 최대화" 중심

### 2.1. 수익은 나는데 너무 빨리 팔림 (큰 상승을 못 먹음)

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| TP1에서 청산 후 큰 상승 놓침 | `SELECT symbol, exit_price, (SELECT MAX(high) FROM market.intraday_bars WHERE symbol=e.symbol AND ts > e.exit_ts AND ts <= e.exit_ts + INTERVAL '1 day') as peak_price FROM trade.exit_events e WHERE action='TP1'` | 전역 모드는 건드리지 말고<br>**default 또는 해당 종목만 조정** | **1순위**: `tp1.qty_pct↓`<br>예: `0.25→0.15~0.20`<br><br>**2순위**: `trailing.pct_trail↑` 또는 `atr_k↑`<br>예: `4%→5%` / `2.0→2.2` | TP 물량을 줄이면 **단기 확정 수익은 감소**<br><br>대신 큰 수익 확률 증가 |

**운영 명령**:
```sql
-- default profile의 tp1.qty_pct 조정
UPDATE trade.exit_profiles
SET params = jsonb_set(
    jsonb_set(params, '{tp1,qty_pct}', '0.15'::jsonb),
    '{trailing,pct_trail}', '0.05'::jsonb
),
    reason='TP1 조기 청산 방지 - 큰 상승 확보',
    updated_by='operator'
WHERE profile_id='default';
```

---

### 2.2. 수익 났다가 본전/손실로 역전 (되돌림 손실)

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| 수익 구간 진입 후 본전 이하 청산 | `SELECT symbol, MAX(pnl) as peak_pnl, final_pnl FROM trade.positions WHERE status='CLOSED' AND MAX(pnl) > 0.01 AND final_pnl < 0` | 우선 **PAUSE_PROFIT가 아니라 StopFloor 강화** | `tp1.stop_floor_profit↑`<br>예: `0.006→0.008~0.010`<br><br>StopFloor 트리거 우선순위 상향 | StopFloor가 너무 타이트하면<br>"큰 상승장"에서 너무 자주 잘릴 수 있음 |

**운영 명령**:
```sql
-- StopFloor 강화
UPDATE trade.exit_profiles
SET params = jsonb_set(
    params,
    '{tp1,stop_floor_profit}',
    '0.010'::jsonb
),
    reason='되돌림 손실 방지 - StopFloor 강화',
    updated_by='operator'
WHERE profile_id='default';
```

---

### 2.3. TP는 잘 되는데 트레일링에서 수익을 많이 반납

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| HWM 대비 하락 폭이 너무 큼 | `SELECT symbol, hwm_price, exit_price, (hwm_price - exit_price) / hwm_price as drawdown FROM trade.exit_events WHERE action='TRAIL'` | 종목 특성(고/저 변동성) 먼저 분리 | **저변동**:<br>`pct_trail↓` (예: `4%→3%`)<br>또는 `atr_k↓` (`2.0→1.8`)<br><br>**고변동**:<br>반대로 `pct_trail↑` (`5%`)<br>또는 `atr_k↑` (`2.3`) | 종목별로 **반대 방향**이 될 수 있어<br>override 우선 |

**운영 명령**:
```sql
-- 저변동성 종목: trailing gap 축소
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('005930', 'low_volatility', '트레일 반납 과다 - gap 축소', 'operator')
ON CONFLICT(symbol) DO UPDATE
SET profile_id=EXCLUDED.profile_id, reason=EXCLUDED.reason;

-- 또는 custom profile 생성
-- low_volatility: pct_trail=3%, atr_k=1.8
```

---

### 2.4. TP 체결이 잘 안 됨 (지정가 미체결)

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| TP intent 생성 후 미체결 장기화 | `SELECT o.order_id, o.symbol, o.status, o.submitted_ts FROM trade.orders o JOIN trade.order_intents oi ON o.intent_id = oi.intent_id WHERE oi.action IN ('TP1','TP2','TP3') AND o.status='SUBMITTED' AND o.submitted_ts < NOW() - INTERVAL '10 minutes'` | Execution 주문 타입 정책 점검<br>(슬리피지/지정가) | TP 주문을 **LMT**로 두면:<br>`limit_price` 버퍼를 확대<br>예: `0.2%→0.3~0.5%`<br><br>또는 TP는 **MKT**로 전환<br>(개인 운영 단순화) | **MKT**는 슬리피지 증가 가능<br><br>종목 유동성에 따라 선택 |

**Execution 정책 조정** (코드 레벨):
```go
// TP는 MKT 권장 (개인 운영 단순화)
if intent.Action == "TP1" || intent.Action == "TP2" || intent.Action == "TP3" {
    orderType = "MKT"
} else {
    orderType = "LMT"
}
```

---

### 2.5. 익절 신호가 나왔는데 stale 때문에 계속 스킵

| IF (관측) | 모니터링 지표 | THEN (즉시 운영 조치) | 프로파일 조정 | 리스크/롤백 |
|-----------|--------------|---------------------|--------------|------------|
| TP 가능 구간인데 가격 stale | 로그: `"TP 평가 스킵 - price stale"` | 이건 **"기회손실"** 성격이 큼<br><br>시스템 안정화가 우선 | 익절/트레일은<br>stale이면 **스킵 유지 권장**<br><br>(손절과 달리 계좌 손실로 직결되지 않음) | stale에서 익절 강행은<br>잘못된 가격으로 팔 위험 |

**정책**:
- TP/TRAIL: stale 시 스킵 유지 (안전)
- SL: stale이어도 HardStop은 우회 허용 (손실 차단 우선)

---

## 3) "숫자 조정"에 대한 실무 규칙(과도한 튜닝 방지)

### A. 손절(SL) 조정 규칙

**"손절이 너무 잦다"를 해결하려고 SL2를 무작정 넓히지 말 것**

**조정 순서**:
```
1. high_beta 적용 여부 확인
   ↓
2. ATR factor_max / min_pct로 "휩쏘 방지"
   ↓
3. 마지막에 sl2 범위 조정
```

**권장 안전 범위 (개인 운영 보수적)**:

| 파라미터 | 범위 | 비고 |
|---------|------|------|
| `sl1.min_pct` | -1.5% ~ -2.5% | 1차 방어선 |
| `sl1.max_pct` | -5% ~ -8% | ATR 기반 상한 |
| `sl2.min_pct` | -3.5% ~ -5% | 전량 손절 시작 |
| `sl2.max_pct` | -8% ~ -12% | HardStop 직전 |

**조정 예시**:
```sql
-- high_beta profile
{
  "sl1": {
    "qty_pct": 0.60,      -- 1차 방어 물량 증가 (휩쏘 대응)
    "min_pct": -0.020,    -- -2.0%
    "max_pct": -0.060     -- -6.0% (ATR 기반 확대)
  },
  "sl2": {
    "qty_pct": 1.00,
    "min_pct": -0.045,    -- -4.5%
    "max_pct": -0.100     -- -10.0%
  },
  "atr": {
    "lookback": 20,
    "factor_min": 1.5,
    "factor_max": 2.5     -- 휩쏘 방지 강화
  }
}
```

---

### B. 익절(TP) 조정 규칙

**"너무 빨리 팔림"은 TP 퍼센트를 올리기보다 물량을 줄이거나 트레일을 조정하는 편이 안정적합니다.**

**권장 조정 순서**:
```
1. tp1.qty_pct 감소 (예: 0.25→0.15)
   ↓
2. trailing gap 확대/축소 (종목 특성 반영)
   ↓
3. tp1/tp2/tp3 퍼센트는 마지막에
```

**조정 예시**:
```sql
-- "큰 상승 확보" 프로파일
{
  "tp1": {
    "qty_pct": 0.15,        -- 물량 감소 (큰 상승 기회 확보)
    "min_pct": 0.025,       -- +2.5%
    "max_pct": 0.050,       -- +5.0%
    "stop_floor_profit": 0.010  -- 본전 방어
  },
  "trailing": {
    "pct_trail": 0.05,      -- 5% (큰 변동 허용)
    "atr_k": 2.2            -- ATR 기반 gap 확대
  }
}
```

---

## 4) 운영자가 실제로 하는 "명령" 예시

### 4.1. 시장 급변/시스템 불안정: 익절만 멈추기

```sql
UPDATE trade.exit_control
SET mode='PAUSE_PROFIT',
    reason='실시간 가격 불안정',
    updated_by='operator'
WHERE id=1;
```

### 4.2. 특정 종목만 high_beta로 전환 (휩쏘 방지)

```sql
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('012450', 'high_beta', '변동성 확대 구간', 'operator')
ON CONFLICT(symbol) DO UPDATE
SET profile_id=EXCLUDED.profile_id,
    reason=EXCLUDED.reason,
    updated_ts=NOW();
```

### 4.3. "수익 확정 강화": StopFloor만 강화 (프로파일 수정)

```sql
UPDATE trade.exit_profiles
SET params = jsonb_set(
    params,
    '{tp1,stop_floor_profit}',
    '0.010'::jsonb
),
    reason='되돌림 손실 방지',
    updated_by='operator'
WHERE profile_id='default';
```

### 4.4. 전체 시스템 긴급 정지 (장애 의심)

```sql
UPDATE trade.exit_control
SET mode='PAUSE_ALL',
    reason='시스템 장애 의심 - 긴급 정지',
    updated_by='operator'
WHERE id=1;
```

---

## 5) 가장 중요한 안전 규칙(캐시/Redis 포함)

### ⚠️ CRITICAL: Redis는 "가속" 용도이고, SSOT는 DB입니다

특히 **qty/avg_price**는 캐시를 참고하더라도:

```
1. intent 생성 직전 DB의 qty/avg_price/version을 재확인
2. 불일치면 재평가 또는 이번 tick skip
```

**이 한 줄이 "개인 시스템 사고"의 대부분을 막습니다.**

### v14 방어 코드 (필수)

```go
// Redis에서 snapshot 조회 (빠른 평가)
snapshot, err := e.cache.GetPosition(ctx, pos.PositionID)

// 청산 조건 평가 (Redis 기반)
shouldExit, action := e.evaluateExit(ctx, snapshot, price)

if shouldExit {
    // ⚠️ CRITICAL: Intent 생성 직전 DB 재확인
    var latestVersion int
    var latestQty int64
    var latestAvgPrice decimal.Decimal

    err := e.db.QueryRow(ctx, `
        SELECT version, qty, avg_price
        FROM trade.positions
        WHERE position_id = $1
    `, pos.PositionID).Scan(&latestVersion, &latestQty, &latestAvgPrice)

    if err != nil {
        return nil, fmt.Errorf("DB 재확인 실패: %w", err)
    }

    // Version 불일치 감지
    if latestVersion != snapshot.Version {
        log.Warn("포지션 변경 감지 - 재평가 필요",
            "position", pos.PositionID,
            "cached_version", snapshot.Version,
            "db_version", latestVersion,
            "cached_qty", snapshot.Qty,
            "db_qty", latestQty)

        // 재평가 또는 skip
        return nil, ErrPositionChanged
    }

    // ✅ DB 최신 값으로 intent 생성
    intent := &OrderIntent{
        PositionID: pos.PositionID,
        Action:     action,
        Qty:        calculateQty(latestQty, action),  // DB 값 사용
        AvgPrice:   latestAvgPrice,                   // DB 값 사용
        Version:    latestVersion,                    // 낙관적 잠금
    }

    return e.createIntent(ctx, intent)
}
```

### 안전 체크리스트

- [ ] **Redis 캐시는 "힌트"**일 뿐, 결정 직전 DB 재확인
- [ ] **Version 불일치 시**: 재평가 또는 skip (절대 무시 금지)
- [ ] **Stale 가격**: TP/TRAIL은 스킵, SL은 HardStop만 우회
- [ ] **Intent 생성**: DB 최신 값(qty/avg_price/version) 사용

---

## 6) 긴급 상황 매뉴얼

### 🚨 Level 3: 시스템 장애 (즉시 - 30초 내)

**증상**:
- WS 완전 불통
- DB 연결 끊김
- Execution 응답 없음

**조치**:
```sql
-- 1. 전체 정지 (HardStop만 우회)
UPDATE trade.exit_control
SET mode='PAUSE_ALL',
    reason='시스템 장애 - 긴급 정지',
    updated_by='operator'
WHERE id=1;

-- 2. HardStop 설정 확인 (최후 방어선)
SELECT profile_id, params->'hard_stop'
FROM trade.exit_profiles;
```

**체크리스트**:
- [ ] PAUSE_ALL 설정 완료
- [ ] 모든 포지션 수동 모니터링 시작
- [ ] 장애 원인 파악 (WS/DB/Execution)
- [ ] KIS HTS에서 수동 청산 준비

---

### ⚠️ Level 2: 데이터 불안정 (1분 내)

**증상**:
- 가격 stale 30초+
- freshness 지표 급증
- WS 재연결 반복

**조치**:
```sql
-- 1. 익절만 차단 (손절은 유지)
UPDATE trade.exit_control
SET mode='PAUSE_PROFIT',
    reason='실시간 가격 stale 30초+ 지속',
    updated_by='operator'
WHERE id=1;

-- 2. Freshness 모니터링
SELECT
    symbol,
    best_ts,
    NOW() - best_ts AS age
FROM market.prices_best
WHERE best_ts < NOW() - INTERVAL '10 seconds'
ORDER BY age DESC
LIMIT 20;
```

**복구 조건**:
- [ ] Freshness stale < 5 (30초간 지속)
- [ ] WS 안정화 확인
- [ ] RUNNING 모드 복귀

---

### 📊 Level 1: 특정 종목 이상 (3분 내)

**증상**:
- 특정 종목만 SL 빈번
- 특정 종목만 체결 지연
- 특정 종목만 가격 이상

**조치**:
```sql
-- 1. 해당 종목만 오버라이드
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('012450', 'high_beta', '이상 징후 - 일시 조정', 'operator')
ON CONFLICT(symbol) DO UPDATE
SET profile_id=EXCLUDED.profile_id, reason=EXCLUDED.reason;

-- 또는 일시적 MANUAL 모드
UPDATE trade.positions
SET exit_mode='MANUAL'
WHERE symbol='012450' AND status='OPEN';
```

**원인 파악**:
- [ ] 종목 이벤트 확인 (공시/뉴스)
- [ ] 변동성 급증 확인
- [ ] 데이터 품질 확인

---

## 7) 모니터링 지표

### 7.1. 일일 체크리스트 (매일 장 마감 후)

```sql
-- 1. 오늘의 Exit 요약
SELECT
    action,
    COUNT(*) as count,
    AVG((exit_price - avg_price) / avg_price) as avg_pnl
FROM trade.exit_events
WHERE created_at::date = CURRENT_DATE
GROUP BY action
ORDER BY action;

-- 2. Stale 발생 횟수
SELECT
    COUNT(*) FILTER (WHERE best_ts < NOW() - INTERVAL '10 seconds') as stale_count,
    COUNT(*) as total_count,
    ROUND(100.0 * COUNT(*) FILTER (WHERE best_ts < NOW() - INTERVAL '10 seconds') / COUNT(*), 2) as stale_pct
FROM market.prices_best;

-- 3. Intent → Order 전환율
SELECT
    COUNT(DISTINCT oi.intent_id) as total_intents,
    COUNT(DISTINCT o.intent_id) as converted_orders,
    ROUND(100.0 * COUNT(DISTINCT o.intent_id) / COUNT(DISTINCT oi.intent_id), 2) as conversion_rate
FROM trade.order_intents oi
LEFT JOIN trade.orders o ON oi.intent_id = o.intent_id
WHERE oi.created_at::date = CURRENT_DATE;

-- 4. 미체결 주문 (5분 이상)
SELECT
    order_id,
    symbol,
    status,
    submitted_ts,
    NOW() - submitted_ts AS pending_duration
FROM trade.orders
WHERE status IN ('SUBMITTED', 'PARTIAL_FILLED')
  AND submitted_ts < NOW() - INTERVAL '5 minutes'
ORDER BY submitted_ts;
```

### 7.2. 주간 리뷰 (매주 금요일)

```sql
-- 1. 주간 Exit 성과
SELECT
    action,
    COUNT(*) as count,
    AVG((exit_price - avg_price) / avg_price) as avg_pnl,
    STDDEV((exit_price - avg_price) / avg_price) as pnl_stddev
FROM trade.exit_events
WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE)
GROUP BY action;

-- 2. SL 빈번 종목 (튜닝 대상)
SELECT
    symbol,
    COUNT(*) FILTER (WHERE action='SL1') as sl1_count,
    COUNT(*) FILTER (WHERE action='SL2') as sl2_count,
    AVG((exit_price - avg_price) / avg_price) as avg_pnl
FROM trade.exit_events
WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE)
GROUP BY symbol
HAVING COUNT(*) FILTER (WHERE action IN ('SL1', 'SL2')) >= 3
ORDER BY sl2_count DESC;

-- 3. Override 효과 검증
SELECT
    seo.symbol,
    seo.profile_id,
    seo.reason,
    COUNT(e.exit_id) as exit_count,
    AVG((e.exit_price - e.avg_price) / e.avg_price) as avg_pnl
FROM trade.symbol_exit_overrides seo
LEFT JOIN trade.exit_events e
    ON seo.symbol = e.symbol
    AND e.created_at >= seo.created_ts
GROUP BY seo.symbol, seo.profile_id, seo.reason;
```

---

## 8) 롤백 절차

### 8.1. 긴급 조치 롤백

#### A. exit_control.mode를 PAUSE_ALL로 한 경우

```sql
-- 1. 원인 해결 확인
-- - WS 안정화
-- - DB 연결 정상
-- - Execution 응답 정상

-- 2. Freshness 점검 (30초간)
SELECT
    COUNT(*) FILTER (WHERE best_ts < NOW() - INTERVAL '10 seconds') as stale_count
FROM market.prices_best;
-- stale_count < 5 확인

-- 3. RUNNING 복귀
UPDATE trade.exit_control
SET mode='RUNNING',
    reason='시스템 정상화 확인 - 운영 재개',
    updated_by='operator'
WHERE id=1;
```

**복귀 조건**:
- [ ] 원인 해결 완료
- [ ] Freshness stale < 5 (30초간)
- [ ] 최소 3개 종목 가격 정상 갱신 확인

---

#### B. PAUSE_PROFIT로 전환한 경우

```sql
-- 1. 데이터 정상화 확인
SELECT symbol, best_ts, NOW() - best_ts AS age
FROM market.prices_best
WHERE best_ts < NOW() - INTERVAL '10 seconds'
ORDER BY age DESC
LIMIT 10;
-- 모두 10초 이내 확인

-- 2. RUNNING 복귀
UPDATE trade.exit_control
SET mode='RUNNING',
    reason='가격 데이터 정상화 - 익절 재개',
    updated_by='operator'
WHERE id=1;
```

---

### 8.2. Symbol Override 롤백

```sql
-- 1. Override 효과 검증 (1주일 후)
SELECT
    symbol,
    COUNT(*) as exit_count,
    COUNT(*) FILTER (WHERE action='SL2') as sl2_count,
    AVG((exit_price - avg_price) / avg_price) as avg_pnl
FROM trade.exit_events
WHERE symbol='012450'
  AND created_at > (SELECT created_ts FROM trade.symbol_exit_overrides WHERE symbol='012450')
GROUP BY symbol;

-- 2. 효과 없으면 override 삭제
DELETE FROM trade.symbol_exit_overrides
WHERE symbol='012450'
  AND reason='SL2 빈번 - 변동성 확대 구간';
```

**효과 검증 기준**:
- SL2 빈도 감소 (예: 주 4회 → 1회)
- 평균 P&L 개선 또는 유지
- 큰 손실 방지 확인

---

### 8.3. Profile 수정 롤백

**⚠️ CRITICAL**: Profile 수정은 **변경 전 값을 반드시 기록**

```sql
-- 변경 전 백업
SELECT profile_id, params
FROM trade.exit_profiles
WHERE profile_id='default';
-- 결과를 별도 파일에 저장

-- 변경 적용
UPDATE trade.exit_profiles
SET params = '...',
    reason='...',
    updated_by='operator'
WHERE profile_id='default';

-- 롤백 (변경 전 값으로 복원)
UPDATE trade.exit_profiles
SET params='<백업한 JSON>',
    reason='롤백 - 효과 없음',
    updated_by='operator'
WHERE profile_id='default';
```

**권장 방법**:
- A/B 테스트: 일부 종목만 override로 적용
- 2주간 효과 측정
- 효과 있으면 default에 반영

---

## 9) 변경 이력 기록 (필수)

### 변경 로그 템플릿

| 시각 | 조치 | 이유 | 영향 범위 | 롤백 계획 | 담당자 |
|------|------|------|----------|----------|--------|
| 2026-01-13 14:30 | PAUSE_PROFIT | WS 불안정 | 전체 | stale < 5로 복구 시 RUNNING | operator |
| 2026-01-13 15:00 | 012450 → high_beta | 휩쏘 3회 | 종목 1개 | 1주일 후 검증 | operator |
| 2026-01-14 10:00 | default.tp1.qty_pct: 0.25→0.15 | 조기 청산 방지 | 전체 | 2주 후 효과 없으면 복원 | operator |

### 기록 위치

1. **DB 필드** (자동):
   - `trade.exit_control.reason`
   - `trade.symbol_exit_overrides.reason`
   - `trade.exit_profiles.reason`

2. **별도 운영 일지** (수동):
   - Notion / Google Docs
   - Git: `docs/operations/changelog.md`

### 운영 일지 예시

```markdown
## 2026-01-13

### 14:30 - 긴급 조치: PAUSE_PROFIT

**증상**:
- 실시간 가격 30초+ stale 지속
- WS 재연결 5회 반복

**조치**:
```sql
UPDATE trade.exit_control
SET mode='PAUSE_PROFIT', reason='WS 불안정', updated_by='operator'
WHERE id=1;
```

**결과**:
- 15:10 - 가격 정상화 확인
- 15:12 - RUNNING 복귀

---

### 15:00 - 종목 조정: 012450 → high_beta

**이유**:
- 최근 3일간 SL1 휩쏘 3회
- 1~2일 내 재상승 반복

**조치**:
```sql
INSERT INTO trade.symbol_exit_overrides(symbol, profile_id, reason, created_by)
VALUES ('012450', 'high_beta', 'SL1 휩쏘 빈번', 'operator');
```

**검증 계획**:
- 1주일 후 (2026-01-20) 효과 측정
- SL1 빈도 감소 확인
```

---

## 📖 관련 문서

| 문서 | 설명 |
|------|------|
| `docs/modules/exit-engine.md` | Exit Engine 설계 문서 (아키텍처, 알고리즘, SSOT) |
| `docs/database/schema.md` | 테이블 스키마 (positions, exit_control, exit_profiles, exit_events 등) |
| `docs/database/access-control.md` | 권한 설계 (컬럼별 소유권) |
| `docs/architecture/architecture-improvements.md` | 아키텍처 개선안 (P0~P2, Redis 캐싱 안전 원칙) |

---

## ⚠️ 최종 체크리스트

운영 조치 전:
- [ ] **조정 우선순위** 확인 (전역 → 종목 → 프로파일)
- [ ] **현재 상태** 확인 (exit_control.mode, override 목록)
- [ ] **변경 전 값** 기록 (롤백용)
- [ ] **영향 범위** 파악 (전체 vs 종목)

운영 조치 후:
- [ ] **변경 이력** 기록 (DB reason + 운영 일지)
- [ ] **모니터링 지표** 확인 (즉시)
- [ ] **롤백 계획** 수립 (복귀 조건 명시)
- [ ] **검증 일정** 설정 (1주/2주 후)

---

**Remember**:
- 전역 제어 → 종목 오버라이드 → 프로파일 (이 순서 필수)
- Intent 생성 직전 DB 재확인 (Redis는 힌트일 뿐)
- 숫자 조정은 단계적으로 (과도한 튜닝 금지)
- 모든 변경은 기록하고, 효과를 검증하고, 필요 시 롤백

