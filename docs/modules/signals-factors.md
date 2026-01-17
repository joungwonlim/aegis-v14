# Signals & Factors 모듈 설계

> 6가지 팩터 시그널 계산 및 종합 점수 산출

**Status**: ✅ 구현 완료
**Version**: 1.1.0
**Last Updated**: 2026-01-17

---

## 📌 개요

Signals 모듈은 유니버스에 포함된 각 종목에 대해 6가지 팩터(Momentum, Technical, Value, Quality, Flow, Event)를 계산하고 종합 점수를 산출합니다.

### 핵심 원칙

- 모든 팩터 점수는 **-1.0 ~ 1.0** 범위로 정규화
- 각 팩터는 독립적으로 계산 후 가중 합산
- **tanh 정규화**를 통해 극단값 억제
- 매일 시장 마감 후 배치 실행

---

## 🎯 6가지 팩터 상세

### 1. Momentum (모멘텀) 팩터

**목적**: 가격 추세와 거래량 성장률 측정

**입력 데이터**:
- 일봉 가격 데이터 (최소 60일)
- 거래량 데이터 (최소 40일)

**계산 로직**:

```go
type MomentumCalculator struct {
    logger *logger.Logger
}

// 계산 요소
// 1. Return1M: 20 거래일 수익률 (40%)
// 2. Return3M: 60 거래일 수익률 (40%)
// 3. VolumeRate: 거래량 성장률 (20%)

func (c *MomentumCalculator) Calculate(prices []PricePoint) float64 {
    return1M := calculateReturn(prices, 20)
    return3M := calculateReturn(prices, 60)
    volumeRate := calculateVolumeGrowth(prices, 20)

    // 가중 합산
    score := return1M*0.4 + return3M*0.4 + volumeRate*0.2

    // tanh 정규화 (-1 ~ 1)
    return math.Tanh(score * 2)
}
```

**가중치**:
| 요소 | 비중 | 설명 |
|------|------|------|
| Return1M | 40% | 단기 모멘텀 |
| Return3M | 40% | 중기 모멘텀 |
| VolumeRate | 20% | 거래량 확인 |

---

### 2. Technical (기술적) 팩터

**목적**: RSI, MACD, MA 크로스 등 기술적 지표 종합

**입력 데이터**:
- 일봉 가격 데이터 (최소 120일 for MA120)

**계산 요소**:

#### RSI (14일)
```go
func calculateRSI(prices []PricePoint, period int) float64 {
    // Relative Strength Index
    // RSI < 30: 과매도 (긍정적)
    // RSI > 70: 과매수 (부정적)
    // RSI = 50: 중립

    avgGain := gains / float64(period)
    avgLoss := losses / float64(period)
    rs := avgGain / avgLoss
    rsi := 100 - (100 / (1 + rs))
    return rsi
}
```

#### MACD (12, 26, 9)
```go
func calculateMACD(prices []PricePoint) (float64, float64) {
    ema12 := calculateEMA(prices, 12)
    ema26 := calculateEMA(prices, 26)
    macd := ema12 - ema26
    signal := calculateEMA(macdValues, 9)
    return macd, signal
}
```

#### MA20 Cross
```go
func calculateMA20Cross(prices []PricePoint) int {
    ma20 := calculateMA(prices, 20)
    currentPrice := prices[0].Price
    priceDiff := (currentPrice - ma20) / ma20

    if priceDiff > 0.02 {
        return 1   // Golden Cross
    } else if priceDiff < -0.02 {
        return -1  // Death Cross
    }
    return 0       // Neutral
}
```

**가중치**:
| 요소 | 비중 | 점수 범위 |
|------|------|----------|
| RSI | 40% | -1 ~ 1 |
| MACD | 40% | -1 ~ 1 (tanh 정규화) |
| MA20 Cross | 20% | -1, 0, 1 |

---

### 3. Value (가치) 팩터

**목적**: PER, PBR, PSR 등 밸류에이션 지표 평가

**입력 데이터**:
- 재무 데이터 (분기별)

**계산 로직**:

```go
type ValueMetrics struct {
    PER float64 // Price to Earnings Ratio
    PBR float64 // Price to Book Ratio
    PSR float64 // Price to Sales Ratio
}

// 점수화 기준 (낮을수록 저평가 = 높은 점수)
// PER: 10 기준, 5 = +1.0, 20 = -0.5
// PBR: 1.0 기준, 0.5 = +1.0, 2.0 = -0.5
// PSR: 1.0 기준, 0.5 = +1.0, 3.0 = -0.5
```

**기준값**:

| 지표 | 저평가 | 중립 | 고평가 |
|------|--------|------|--------|
| PER | < 10 | 10~20 | > 20 |
| PBR | < 1.0 | 1.0~2.0 | > 2.0 |
| PSR | < 1.0 | 1.0~3.0 | > 3.0 |

**가중치**:
| 요소 | 비중 |
|------|------|
| PER | 50% |
| PBR | 30% |
| PSR | 20% |

---

### 4. Quality (퀄리티) 팩터

**목적**: ROE, 부채비율 등 기업 질적 지표 평가

**입력 데이터**:
- 재무 데이터 (분기별)

**계산 로직**:

```go
type QualityMetrics struct {
    ROE       float64 // Return on Equity (%)
    DebtRatio float64 // 부채비율 (%)
}

// ROE: 높을수록 좋음
// ROE > 15%: 우량 (양수)
// ROE < 5%: 저품질 (음수)

// DebtRatio: 낮을수록 좋음
// Debt < 50%: 저위험 (양수)
// Debt > 150%: 고위험 (음수)
```

**기준값**:

| 지표 | 우량 | 중립 | 저품질 |
|------|------|------|--------|
| ROE | > 15% | 5~15% | < 5% |
| DebtRatio | < 50% | 50~150% | > 150% |

**가중치**:
| 요소 | 비중 |
|------|------|
| ROE | 60% |
| DebtRatio | 40% |

---

### 5. Flow (수급) 팩터

**목적**: 외국인/기관 순매수 동향 분석

**입력 데이터**:
- 투자자별 순매수 데이터 (최소 20일)

**계산 로직**:

```go
type FlowData struct {
    ForeignNet  int64  // 외국인 순매수
    InstNet     int64  // 기관 순매수
    IndividualNet int64 // 개인 순매수
}

// 5일/20일 누적 순매수 계산
foreignNet5D := sum(flowData[:5], "foreign")
foreignNet20D := sum(flowData[:20], "foreign")
instNet5D := sum(flowData[:5], "inst")
instNet20D := sum(flowData[:20], "inst")

// tanh 정규화
// 기준: 5D = 50만주, 20D = 200만주
foreignScore5D := math.Tanh(float64(foreignNet5D) / 500_000)
foreignScore20D := math.Tanh(float64(foreignNet20D) / 2_000_000)
```

**가중치**:
| 요소 | 비중 | 시간 가중치 |
|------|------|------------|
| 외국인 | 60% | 5D: 70%, 20D: 30% |
| 기관 | 40% | 5D: 70%, 20D: 30% |

**수급 스마트머니 원칙**:
- 외국인/기관 = Smart Money (높은 비중)
- 개인 = 역지표 (참고용)

---

### 6. Event (이벤트) 팩터

**목적**: 공시, 뉴스, 실적 등 이벤트 영향도 평가

**입력 데이터**:
- DART 공시
- 뉴스 이벤트
- 실적 발표

**이벤트 유형 및 영향도**:

#### 긍정적 이벤트 (0.3 ~ 1.0)
| 이벤트 | 영향도 |
|--------|--------|
| 실적 개선 (earnings_positive) | +1.0 |
| 인수합병 긍정 (merger_positive) | +0.9 |
| 자사주 매입 (share_buyback) | +0.8 |
| 신제품 출시 (new_product) | +0.7 |
| 배당 증가 (dividend_increase) | +0.6 |
| 파트너십 체결 (partnership) | +0.6 |
| 설비 투자 (capex_increase) | +0.5 |
| 특허 취득 (patent) | +0.5 |

#### 부정적 이벤트 (-0.3 ~ -1.0)
| 이벤트 | 영향도 |
|--------|--------|
| 실적 악화 (earnings_negative) | -1.0 |
| 감사 의견 (audit_opinion) | -0.9 |
| 인수합병 부정 (merger_negative) | -0.8 |
| 제품 리콜 (recall) | -0.8 |
| 소송 (lawsuit) | -0.7 |
| 규제 이슈 (regulatory) | -0.7 |
| 배당 감소 (dividend_decrease) | -0.6 |
| 경영진 교체 (management_change) | -0.5 |

**시간 가중치 (Exponential Decay)**:

```go
// 최근 이벤트일수록 높은 가중치
// 감쇠율 k = 0.023
// 7일 이내: ~100%
// 30일 이내: ~50%
// 90일 이내: ~25%
// 90일 초과: 10% (floor)

func calculateTimeWeight(daysSince float64) float64 {
    const decayRate = 0.023
    weight := math.Exp(-decayRate * daysSince)
    if weight < 0.1 {
        weight = 0.1
    }
    return weight
}
```

---

## 📊 종합 점수 계산

### 팩터별 가중치 (기본값)

| 팩터 | 기본 비중 | 설명 |
|------|----------|------|
| Momentum | 20% | 추세 추종 |
| Technical | 15% | 기술적 분석 |
| Value | 20% | 가치 평가 |
| Quality | 15% | 기업 품질 |
| Flow | 20% | 수급 분석 |
| Event | 10% | 이벤트 영향 |
| **합계** | **100%** | |

### 종합 점수 공식

```go
func calculateTotalScore(factors SignalFactors, weights FactorWeights) float64 {
    score := factors.Momentum * weights.Momentum +
             factors.Technical * weights.Technical +
             factors.Value * weights.Value +
             factors.Quality * weights.Quality +
             factors.Flow * weights.Flow +
             factors.Event * weights.Event

    // 최종 정규화 (-1 ~ 1)
    return math.Tanh(score * 1.5)
}
```

---

## 🗄️ 데이터베이스 스키마

### signals.factor_scores

```sql
CREATE TABLE signals.factor_scores (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    momentum     NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    technical    NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    value        NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    quality      NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    flow         NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    event        NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    total_score  NUMERIC(5,4),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(calc_date);
CREATE INDEX idx_factor_scores_total ON signals.factor_scores(total_score DESC);
```

### signals.flow_details

```sql
CREATE TABLE signals.flow_details (
    stock_code        VARCHAR(20) NOT NULL,
    calc_date         DATE NOT NULL,
    foreign_net_5d    BIGINT DEFAULT 0,
    inst_net_5d       BIGINT DEFAULT 0,
    indiv_net_5d      BIGINT DEFAULT 0,
    foreign_net_10d   BIGINT DEFAULT 0,
    inst_net_10d      BIGINT DEFAULT 0,
    indiv_net_10d     BIGINT DEFAULT 0,
    foreign_net_20d   BIGINT DEFAULT 0,
    inst_net_20d      BIGINT DEFAULT 0,
    indiv_net_20d     BIGINT DEFAULT 0,
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);
```

### signals.technical_details

```sql
CREATE TABLE signals.technical_details (
    stock_code   VARCHAR(20) NOT NULL,
    calc_date    DATE NOT NULL,
    ma5          NUMERIC(12,2),
    ma10         NUMERIC(12,2),
    ma20         NUMERIC(12,2),
    ma60         NUMERIC(12,2),
    ma120        NUMERIC(12,2),
    rsi14        NUMERIC(5,2),
    macd         NUMERIC(12,4),
    macd_signal  NUMERIC(12,4),
    macd_hist    NUMERIC(12,4),
    bb_upper     NUMERIC(12,2),
    bb_middle    NUMERIC(12,2),
    bb_lower     NUMERIC(12,2),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, calc_date)
);
```

### signals.event_signals

```sql
CREATE TABLE signals.event_signals (
    id            SERIAL PRIMARY KEY,
    stock_code    VARCHAR(20) NOT NULL,
    event_date    DATE NOT NULL,
    event_type    VARCHAR(50) NOT NULL,
    event_subtype VARCHAR(50),
    title         TEXT,
    description   TEXT,
    impact_score  NUMERIC(5,4) DEFAULT 0.0,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 🔌 API 엔드포인트

### 팩터 점수 조회

```
GET /api/v1/signals/factors?stock_code={code}&date={YYYY-MM-DD}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "stock_code": "005930",
    "calc_date": "2026-01-17",
    "factors": {
      "momentum": 0.72,
      "technical": 0.45,
      "value": 0.38,
      "quality": 0.85,
      "flow": 0.62,
      "event": 0.25
    },
    "total_score": 0.68
  }
}
```

### 랭킹 조회 (점수 순)

```
GET /api/v1/signals/ranking?market={KOSPI|KOSDAQ|ALL}&date={YYYY-MM-DD}&limit=100
```

---

## 🔗 v14 구현 매핑

| v13 위치 | v14 위치 |
|----------|----------|
| `internal/s2_signals/momentum.go` | `internal/service/signals/momentum.go` |
| `internal/s2_signals/technical.go` | `internal/service/signals/technical.go` |
| `internal/s2_signals/value.go` | `internal/service/signals/value.go` |
| `internal/s2_signals/quality.go` | `internal/service/signals/quality.go` |
| `internal/s2_signals/flow.go` | `internal/service/signals/flow.go` |
| `internal/s2_signals/event.go` | `internal/service/signals/event.go` |
| `internal/s2_signals/builder.go` | `internal/service/signals/builder.go` |
| `internal/s2_signals/repository.go` | `internal/infra/database/postgres/signals/factor_repository.go` |
| `internal/contracts/signals.go` | `internal/domain/signals/model.go` |

### 주요 구현 파일

**Domain Layer**:
- `internal/domain/signals/model.go` - 6팩터 도메인 모델
- `internal/domain/signals/repository.go` - 리포지토리 인터페이스
- `internal/domain/signals/errors.go` - 에러 정의

**Service Layer**:
- `internal/service/signals/momentum.go` - 모멘텀 Calculator
- `internal/service/signals/technical.go` - 기술적 Calculator
- `internal/service/signals/value.go` - 가치 Calculator
- `internal/service/signals/quality.go` - 품질 Calculator
- `internal/service/signals/flow.go` - 수급 Calculator
- `internal/service/signals/event.go` - 이벤트 Calculator
- `internal/service/signals/builder.go` - 6팩터 오케스트레이터

**Infrastructure Layer**:
- `internal/infra/database/postgres/signals/factor_repository.go` - 팩터 리포지토리
- `internal/infra/database/postgres/signals/signal_repository.go` - 신호 리포지토리

**API Layer**:
- `internal/api/handlers/signals/handler.go` - REST API 핸들러
- `internal/api/routes/signals_routes.go` - 라우트 등록

---

**Version**: v14.1.1.0
**Last Updated**: 2026-01-17
