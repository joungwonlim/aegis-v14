# Audit 모듈 설계 (v13 참조)

> S7: 성과 분석 및 리스크 모니터링

---

## 📌 개요

Audit 모듈은 포트폴리오 성과를 분석하고 리스크를 모니터링합니다. 수익률, 변동성, 최대 낙폭 등 핵심 지표를 계산하고 벤치마크 대비 성과를 평가합니다.

### 핵심 기능

1. **성과 분석**: 수익률, 샤프비율, 소르티노 비율 등
2. **리스크 지표**: 변동성, 최대 낙폭 (MDD)
3. **트레이딩 지표**: 승률, 평균 손익, 수익 팩터
4. **벤치마크 비교**: Alpha, Beta 계산
5. **귀속 분석**: 팩터별/섹터별/종목별 기여도

---

## 📊 성과 지표

### 1. 수익률 지표

#### Total Return (누적 수익률)
```go
func calculateTotalReturn(dailyReturns []float64) float64 {
    cumReturn := 1.0
    for _, r := range dailyReturns {
        cumReturn *= (1.0 + r)
    }
    return cumReturn - 1.0
}
```

#### Annualized Return (연환산 수익률)
```go
func annualize(totalReturn float64, days int) float64 {
    if days == 0 {
        return 0
    }
    // 252 = 연간 거래일 수
    return math.Pow(1.0+totalReturn, 252.0/float64(days)) - 1.0
}
```

---

### 2. 리스크 지표

#### Volatility (연환산 변동성)
```go
func calculateVolatility(dailyReturns []float64) float64 {
    if len(dailyReturns) < 2 {
        return 0
    }

    // 평균
    var sum float64
    for _, r := range dailyReturns {
        sum += r
    }
    mean := sum / float64(len(dailyReturns))

    // 분산
    var variance float64
    for _, r := range dailyReturns {
        diff := r - mean
        variance += diff * diff
    }
    variance /= float64(len(dailyReturns) - 1)

    // 연환산 변동성 = 일간 표준편차 × √252
    return math.Sqrt(variance) * math.Sqrt(252)
}
```

#### Sharpe Ratio (샤프 비율)
```go
func calculateSharpe(annualReturn, volatility float64) float64 {
    if volatility == 0 {
        return 0
    }
    riskFreeRate := 0.03 // 3% 무위험 수익률
    return (annualReturn - riskFreeRate) / volatility
}
```

**해석**:
| Sharpe | 평가 |
|--------|------|
| < 0 | 손실 |
| 0 ~ 1.0 | 평균 이하 |
| 1.0 ~ 2.0 | 양호 |
| > 2.0 | 우수 |

#### Sortino Ratio (소르티노 비율)
```go
func calculateSortino(dailyReturns []float64) float64 {
    // Downside deviation (음수 수익률만 사용)
    var sumSquaredNegative float64
    var countNegative int
    for _, r := range dailyReturns {
        if r < 0 {
            sumSquaredNegative += r * r
            countNegative++
        }
    }

    if countNegative == 0 {
        return 0
    }

    downsideVol := math.Sqrt(sumSquaredNegative/float64(countNegative)) * math.Sqrt(252)
    return (annualReturn - riskFreeRate) / downsideVol
}
```

**Sharpe vs Sortino**:
- Sharpe: 전체 변동성 기준
- Sortino: 하방 위험만 고려 (투자자 관점에서 더 적절)

#### Maximum Drawdown (최대 낙폭)
```go
func calculateMaxDrawdown(dailyReturns []float64) float64 {
    if len(dailyReturns) == 0 {
        return 0
    }

    cumValue := 1.0
    peak := 1.0
    maxDD := 0.0

    for _, r := range dailyReturns {
        cumValue *= (1.0 + r)
        if cumValue > peak {
            peak = cumValue
        }
        dd := (cumValue - peak) / peak
        if dd < maxDD {
            maxDD = dd
        }
    }

    return maxDD // 음수로 반환
}
```

**해석**:
| MDD | 평가 |
|-----|------|
| > -10% | 안정적 |
| -10% ~ -20% | 보통 |
| -20% ~ -30% | 변동성 큼 |
| < -30% | 고위험 |

---

### 3. 트레이딩 지표

#### Win Rate (승률)
```go
func calculateWinRate(trades []Trade) float64 {
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
```

#### Average Win/Loss (평균 손익)
```go
func calculateAvgWinLoss(trades []Trade) (float64, float64) {
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

    avgWin := sumWin / float64(countWin)
    avgLoss := sumLoss / float64(countLoss)

    return avgWin, avgLoss
}
```

#### Profit Factor (수익 팩터)
```go
func calculateProfitFactor(trades []Trade) float64 {
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

**해석**:
| Profit Factor | 평가 |
|---------------|------|
| < 1.0 | 손실 |
| 1.0 ~ 1.5 | 보통 |
| 1.5 ~ 2.0 | 양호 |
| > 2.0 | 우수 |

---

### 4. 벤치마크 비교

#### Alpha (초과 수익률)
```go
alpha := portfolioReturn - benchmarkReturn
```

#### Beta (시장 민감도)
```go
func calculateBeta(portfolioReturns, benchmarkReturns []float64) float64 {
    covariance := calculateCovariance(portfolioReturns, benchmarkReturns)
    benchmarkVariance := calculateVariance(benchmarkReturns)

    if benchmarkVariance == 0 {
        return 0
    }

    return covariance / benchmarkVariance
}
```

**해석**:
| Beta | 의미 |
|------|------|
| β = 1.0 | 시장과 동일 |
| β > 1.0 | 시장보다 변동성 큼 |
| β < 1.0 | 시장보다 변동성 작음 |
| β < 0 | 시장과 반대 |

---

## 📈 귀속 분석 (Attribution Analysis)

### 팩터별 기여도

```go
type AttributionAnalysis struct {
    TotalReturn       float64 `json:"total_return"`
    MomentumContrib   float64 `json:"momentum_contrib"`
    TechnicalContrib  float64 `json:"technical_contrib"`
    ValueContrib      float64 `json:"value_contrib"`
    QualityContrib    float64 `json:"quality_contrib"`
    FlowContrib       float64 `json:"flow_contrib"`
    EventContrib      float64 `json:"event_contrib"`
    SectorContrib     map[string]float64 `json:"sector_contrib"`
    StockContrib      map[string]float64 `json:"stock_contrib"`
}
```

### 계산 방법

1. **팩터별 기여도**: 각 팩터 점수와 종목 수익률의 상관관계 분석
2. **섹터별 기여도**: 섹터 비중 × 섹터 수익률
3. **종목별 기여도**: 종목 비중 × 종목 수익률

---

## 🗄️ 데이터베이스 스키마

### audit.performance_reports

```sql
CREATE TABLE audit.performance_reports (
    report_date       DATE PRIMARY KEY,
    period_start      DATE NOT NULL,
    period_end        DATE NOT NULL,
    total_return      NUMERIC(10,6),
    benchmark_return  NUMERIC(10,6),
    alpha             NUMERIC(10,6),
    beta              NUMERIC(10,6),
    sharpe_ratio      NUMERIC(10,6),
    sortino_ratio     NUMERIC(10,6),
    volatility        NUMERIC(10,6),
    max_drawdown      NUMERIC(10,6),
    win_rate          NUMERIC(5,4),
    avg_win           NUMERIC(10,6),
    avg_loss          NUMERIC(10,6),
    profit_factor     NUMERIC(10,6),
    total_trades      INT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

### audit.attribution_analysis

```sql
CREATE TABLE audit.attribution_analysis (
    analysis_date     DATE PRIMARY KEY,
    period_start      DATE NOT NULL,
    period_end        DATE NOT NULL,
    total_return      NUMERIC(10,6),
    -- 팩터별 기여도
    momentum_contrib  NUMERIC(10,6),
    technical_contrib NUMERIC(10,6),
    value_contrib     NUMERIC(10,6),
    quality_contrib   NUMERIC(10,6),
    flow_contrib      NUMERIC(10,6),
    event_contrib     NUMERIC(10,6),
    -- 섹터별 기여도
    sector_contrib    JSONB,
    -- 종목별 기여도
    stock_contrib     JSONB,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

### audit.benchmark_data

```sql
CREATE TABLE audit.benchmark_data (
    benchmark_date DATE NOT NULL,
    benchmark_code VARCHAR(20) NOT NULL,  -- KOSPI, KOSDAQ
    close_price    NUMERIC(12,2) NOT NULL,
    daily_return   NUMERIC(10,6),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (benchmark_date, benchmark_code)
);
```

### audit.daily_pnl

```sql
CREATE TABLE audit.daily_pnl (
    pnl_date          DATE PRIMARY KEY,
    realized_pnl      BIGINT DEFAULT 0,
    unrealized_pnl    BIGINT DEFAULT 0,
    total_pnl         BIGINT,
    daily_return      NUMERIC(10,6),
    cumulative_return NUMERIC(10,6),
    portfolio_value   BIGINT,
    cash_balance      BIGINT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 🔌 API 엔드포인트

### 성과 분석 조회

```
GET /api/v1/audit/performance?period={1M|3M|6M|1Y|YTD}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "period": "3M",
    "start_date": "2025-10-17",
    "end_date": "2026-01-17",
    "total_return": 0.0856,
    "annual_return": 0.4124,
    "volatility": 0.1825,
    "sharpe": 2.09,
    "sortino": 2.45,
    "max_drawdown": -0.0632,
    "win_rate": 0.58,
    "avg_win": 1250000,
    "avg_loss": -780000,
    "profit_factor": 1.85,
    "benchmark": 0.0512,
    "alpha": 0.0344,
    "beta": 0.92
  }
}
```

### 일별 손익 조회

```
GET /api/v1/audit/daily-pnl?start_date={YYYY-MM-DD}&end_date={YYYY-MM-DD}
```

### 귀속 분석 조회

```
GET /api/v1/audit/attribution?period={1M|3M|6M|1Y|YTD}
```

---

## 📊 리포트 기간

| 코드 | 기간 | 설명 |
|------|------|------|
| 1M | 1개월 | 최근 1개월 |
| 3M | 3개월 | 최근 3개월 |
| 6M | 6개월 | 최근 6개월 |
| 1Y | 1년 | 최근 1년 |
| YTD | Year-to-Date | 올해 1월 1일부터 현재까지 |

---

## 🔗 v14 마이그레이션 매핑

| v13 위치 | v14 위치 | 상태 |
|----------|----------|------|
| `internal/audit/performance.go` | `internal/domain/audit/model.go` | ✅ 완료 |
| `internal/audit/attribution.go` | `internal/service/audit/calculator.go` | ✅ 완료 |
| `internal/audit/repository.go` | `internal/infra/database/postgres/audit/repository.go` | ✅ 완료 |
| - | `internal/service/audit/service.go` | ✅ 완료 |
| - | `internal/service/audit/trading_metrics.go` | ✅ 완료 |
| - | `internal/api/handlers/audit/handler.go` | ✅ 완료 |
| - | `internal/api/routes/audit_routes.go` | ✅ 완료 |

---

## 📁 v14 구현 구조

```
internal/
├── domain/audit/
│   ├── model.go           # PerformanceReport, DailyPnL, Attribution 등 도메인 모델
│   └── repository.go      # Repository 인터페이스 정의
├── service/audit/
│   ├── calculator.go      # 수익률/리스크 계산 로직
│   ├── trading_metrics.go # 트레이딩 지표 계산
│   └── service.go         # Audit 서비스 (비즈니스 로직)
├── infra/database/postgres/audit/
│   └── repository.go      # PostgreSQL Repository 구현
└── api/
    ├── handlers/audit/
    │   └── handler.go     # REST API 핸들러
    └── routes/
        └── audit_routes.go # 라우트 등록

migrations/
└── 103_create_audit_tables.sql # DB 스키마
```

---

## 📋 PerformanceReport 구조

```go
type PerformanceReport struct {
    Period      string    `json:"period"`
    StartDate   time.Time `json:"start_date"`
    EndDate     time.Time `json:"end_date"`

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
```

---

## ⚠️ 주의사항

1. **데이터 없음 처리**: 신규 시스템은 성과 데이터가 없을 수 있음 → 빈 리포트 반환
2. **벤치마크 데이터**: KOSPI/KOSDAQ 벤치마크 데이터 별도 수집 필요
3. **거래일 기준**: 252 거래일 기준 연환산 (한국 시장)
4. **무위험 수익률**: 3% 고정 (한국 국채 금리 참고)

---

---

## ✅ 구현 완료 항목

- [x] Domain Layer: 모델 및 Repository 인터페이스
- [x] Service Layer: 성과 계산, 트레이딩 지표, 리스크 계산
- [x] Infrastructure Layer: PostgreSQL Repository 구현
- [x] API Layer: REST API 핸들러 및 라우트
- [x] Migration: audit 스키마 테이블 정의

---

**Version**: v14.0.0 (구현 완료)
**Last Updated**: 2026-01-17
