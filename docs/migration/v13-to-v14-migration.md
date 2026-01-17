# v13 → v14 마이그레이션 가이드

> v13의 핵심 기능을 v14 아키텍처에 맞게 이전하는 가이드

**Version**: 1.0.0
**작성일**: 2026-01-17
**상태**: 🚧 진행 중

---

## 📋 개요

### 마이그레이션 대상

| v13 모듈 | v14 대상 | 우선순위 | 상태 |
|----------|----------|----------|------|
| s1_universe | service/universe | P1 | ⬜ 대기 |
| s2_signals | service/signals | P1 | ⬜ 대기 |
| audit | service/audit | P2 | ⬜ 대기 |
| s0_data/collector (Fetcher) | service/fetcher | P1 | ⬜ 대기 |

### 아키텍처 차이

```
v13 구조 (Stage-based)          v14 구조 (Layer-based)
├── internal/                   ├── internal/
│   ├── s0_data/               │   ├── domain/         # 도메인 모델
│   ├── s1_universe/           │   ├── service/        # 비즈니스 로직
│   ├── s2_signals/            │   ├── infrastructure/ # 저장소 구현
│   ├── audit/                 │   └── api/            # HTTP 핸들러
│   └── contracts/             │
```

### 핵심 원칙

1. **v14 기존 코드 수정 금지**: 새 파일만 추가
2. **v14 모듈 활용**: infra.database, domain 패턴 재사용
3. **점진적 이전**: 모듈 단위로 순차 적용
4. **테스트 우선**: 각 모듈 이전 후 테스트 검증

---

## 🔄 모듈별 이전 가이드

---

### 1. Fetcher (데이터 수집기)

#### v13 현재 구조

```
backend/internal/s0_data/
├── collector/
│   └── collector.go       # 통합 수집기
├── repository.go          # 데이터 저장
├── price_repository.go    # 가격 저장
├── investor_flow_repository.go  # 수급 저장
└── financial_repository.go      # 재무 저장

backend/internal/external/
├── naver/   # Naver Finance API
├── dart/    # DART 공시 API
├── krx/     # KRX 시장 API
└── kis/     # 한국투자증권 API

backend/cmd/quant/commands/fetcher.go  # CLI
```

#### v13 주요 기능

| 기능 | 소스 | 데이터 |
|------|------|--------|
| 가격 수집 | Naver | 일봉, 거래량, 거래대금 |
| 투자자 수급 | Naver | 외국인/기관/개인 순매수 |
| 시가총액 | Naver/KRX | 시총, 상장주식수 |
| 공시 | DART | 공시 제목, 날짜, 유형 |
| 시장 지표 | KRX | 시장 트렌드 |
| 실시간 시세 | KIS | 현재가, 체결 |

#### v14 이전 계획

##### 파일 구조

```
backend/internal/
├── domain/fetcher/
│   ├── model.go           # FetchJob, FetchResult
│   ├── repository.go      # 인터페이스
│   └── errors.go
├── service/fetcher/
│   ├── service.go         # 오케스트레이션
│   ├── price_collector.go # 가격 수집
│   ├── flow_collector.go  # 수급 수집
│   ├── disclosure_collector.go  # 공시 수집
│   └── marketcap_collector.go   # 시가총액 수집
├── infrastructure/postgres/fetcher/
│   └── repository.go
└── api/handlers/fetcher/
    └── handler.go
```

##### 코드 이전 체크리스트

```
⬜ 1. domain/fetcher/model.go 생성
   - FetchJob: 수집 작업 정의
   - FetchResult: 수집 결과
   - FetchStatus: 상태 (pending, running, completed, failed)

⬜ 2. external 클라이언트 이전
   - naver.Client → 그대로 복사 (변경 없음)
   - dart.Client → 그대로 복사
   - krx.Client → 그대로 복사
   - kis.Client → 그대로 복사

⬜ 3. service/fetcher 구현
   - v13 collector.go 로직 분리
   - 각 수집기를 별도 파일로

⬜ 4. infrastructure 구현
   - v13 repository 패턴 적용
   - v14 database 패키지 사용

⬜ 5. CLI 명령어 추가
   - go run ./cmd/quant fetcher collect all
   - go run ./cmd/quant fetcher collect naver
   - go run ./cmd/quant fetcher collect dart
```

##### v13 코드 참조 위치

```go
// 수집기 메인 로직
// v13: backend/internal/s0_data/collector/collector.go

// CLI 명령어
// v13: backend/cmd/quant/commands/fetcher.go

// 외부 API 클라이언트
// v13: backend/internal/external/naver/client.go
// v13: backend/internal/external/dart/client.go
// v13: backend/internal/external/krx/client.go
```

---

### 2. Universe (투자 유니버스)

#### v13 현재 구조

```
backend/internal/s1_universe/
├── builder.go      # 유니버스 생성
├── builder_test.go
└── repository.go   # 저장소
```

#### v13 핵심 로직

##### 필터링 기준 (Config)

```go
type Config struct {
    MinMarketCap   int64    // 최소 시가총액 (억원)
    MinVolume      int64    // 최소 거래대금 (백만원)
    MinListingDays int      // 최소 상장일수
    ExcludeAdmin   bool     // 관리종목 제외
    ExcludeHalt    bool     // 거래정지 제외
    ExcludeSPAC    bool     // SPAC 제외
    ExcludeSectors []string // 제외 섹터
}
```

##### 제외 사유

1. 거래정지
2. 관리종목
3. SPAC (스팩, 제N호 등)
4. 시가총액 미달
5. 거래대금 미달
6. 상장일수 미달
7. 제외 섹터

#### v14 이전 계획

v14에 이미 `docs/modules/universe.md` 설계가 있음. v13 로직을 v14 설계에 맞게 통합.

##### 매핑

| v13 | v14 |
|-----|-----|
| Builder.Build() | Service.GenerateSnapshot() |
| Builder.checkExclusion() | Service.passesFilter() |
| Config | FilterCriteria |
| Stock | UniverseStock |

##### 코드 이전 체크리스트

```
⬜ 1. v14 domain/universe/model.go 확인
   - v13 Stock 구조체와 비교
   - 누락 필드 추가 (IsSPAC, IsAdmin 등)

⬜ 2. service/universe/filter.go 구현
   - v13 checkExclusion() 로직 이전
   - SPAC 패턴 정규식 이전

⬜ 3. service/universe/builder.go 구현
   - v13 getAllStocks() 쿼리 이전
   - v14 repository 인터페이스 사용

⬜ 4. 테스트 이전
   - v13 builder_test.go → v14 형식으로
```

##### v13 코드 참조 위치

```go
// 유니버스 빌더
// v13: backend/internal/s1_universe/builder.go

// SPAC 패턴
// var spacPattern = regexp.MustCompile(`(?i)(스팩|SPAC|스펙|\d+호$|제\d+호)`)
```

---

### 3. Signals (매매 신호)

#### v13 현재 구조

```
backend/internal/s2_signals/
├── builder.go      # 신호 생성 오케스트레이션
├── momentum.go     # 모멘텀 팩터
├── technical.go    # 기술적 팩터
├── value.go        # 가치 팩터
├── quality.go      # 품질 팩터
├── flow.go         # 수급 팩터
├── event.go        # 이벤트 팩터
└── repository.go   # 저장소
```

#### v13 6개 팩터

| 팩터 | 설명 | 입력 데이터 |
|------|------|-------------|
| Momentum | 수익률 모멘텀 | 60일 가격 |
| Technical | RSI, MACD, MA | 120일 가격 |
| Value | PER, PBR, PSR | 재무 데이터 |
| Quality | ROE, 부채비율 | 재무 데이터 |
| Flow | 외국인/기관 순매수 | 수급 데이터 |
| Event | 공시 이벤트 | DART 공시 |

#### v13 신호 구조

```go
type StockSignals struct {
    Code      string
    Momentum  float64  // 모멘텀 점수 (0-100)
    Technical float64  // 기술적 점수
    Value     float64  // 가치 점수
    Quality   float64  // 품질 점수
    Flow      float64  // 수급 점수
    Event     float64  // 이벤트 점수
    Details   SignalDetails  // 상세 지표
}
```

#### v14 이전 계획

v14에 이미 `docs/modules/signals.md` 설계가 있음.

##### 매핑

| v13 | v14 |
|-----|-----|
| Builder | Service |
| *Calculator | scorer.go 내 함수 |
| StockSignals | Signal |
| SignalDetails | Breakdown |

##### 코드 이전 체크리스트

```
⬜ 1. 팩터 계산 로직 이전
   - momentum.go → service/signals/momentum_scorer.go
   - technical.go → service/signals/technical_scorer.go
   - value.go → service/signals/value_scorer.go
   - quality.go → service/signals/quality_scorer.go
   - flow.go → service/signals/flow_scorer.go
   - event.go → service/signals/event_scorer.go

⬜ 2. 이벤트 매핑 로직 이전
   - mapDisclosureToEventType() 함수
   - EventType 상수들
   - GetEventImpact() 함수

⬜ 3. 오케스트레이션 로직
   - v13 Builder.Build() → v14 Service.GenerateSignals()
   - 병렬 처리 유지

⬜ 4. Repository 이전
   - v13 쿼리들 v14 형식으로
```

##### v13 코드 참조 위치

```go
// 팩터 계산기들
// v13: backend/internal/s2_signals/momentum.go
// v13: backend/internal/s2_signals/technical.go
// v13: backend/internal/s2_signals/value.go
// v13: backend/internal/s2_signals/quality.go
// v13: backend/internal/s2_signals/flow.go
// v13: backend/internal/s2_signals/event.go

// 이벤트 매핑 (중요!)
// v13: backend/internal/s2_signals/builder.go (mapDisclosureToEventType 함수)
```

---

### 4. Audit (성과 분석)

#### v13 현재 구조

```
backend/internal/audit/
├── performance.go   # 성과 지표 계산
├── risk_report.go   # 리스크 리포트
├── attribution.go   # 성과 귀인 분석
├── snapshot.go      # 스냅샷 관리
└── repository.go    # 저장소
```

#### v13 성과 지표

##### PerformanceReport

```go
type PerformanceReport struct {
    // 수익률
    TotalReturn  float64  // 누적 수익률
    AnnualReturn float64  // 연환산 수익률

    // 리스크 지표
    Volatility  float64  // 변동성
    Sharpe      float64  // 샤프 비율
    Sortino     float64  // 소르티노 비율
    MaxDrawdown float64  // 최대 낙폭

    // 트레이딩 지표
    WinRate      float64  // 승률
    AvgWin       float64  // 평균 이익
    AvgLoss      float64  // 평균 손실
    ProfitFactor float64  // 손익비

    // 벤치마크 비교
    Benchmark float64  // KOSPI 수익률
    Alpha     float64  // 알파
    Beta      float64  // 베타
}
```

#### v14 이전 계획

##### 파일 구조

```
backend/internal/
├── domain/audit/
│   ├── model.go           # PerformanceReport, RiskReport
│   ├── repository.go
│   └── errors.go
├── service/audit/
│   ├── service.go         # 오케스트레이션
│   ├── performance.go     # 성과 계산
│   ├── risk.go            # 리스크 계산
│   └── attribution.go     # 귀인 분석
├── infrastructure/postgres/audit/
│   └── repository.go
└── api/handlers/audit/
    └── handler.go
```

##### 코드 이전 체크리스트

```
⬜ 1. domain/audit/model.go 생성
   - PerformanceReport 구조체
   - RiskReport 구조체
   - Trade 구조체

⬜ 2. service/audit/performance.go 이전
   - calculateTotalReturn()
   - calculateVolatility()
   - calculateSharpe()
   - calculateSortino()
   - calculateMaxDrawdown()
   - calculateWinRate()
   - calculateProfitFactor()

⬜ 3. service/audit/risk.go 이전
   - v13 risk_report.go 로직

⬜ 4. API 엔드포인트
   - GET /api/v1/audit/performance?period=1M
   - GET /api/v1/audit/risk
```

##### v13 코드 참조 위치

```go
// 성과 분석
// v13: backend/internal/audit/performance.go

// 리스크 리포트
// v13: backend/internal/audit/risk_report.go

// 성과 귀인
// v13: backend/internal/audit/attribution.go
```

---

## 🗄️ 데이터베이스 마이그레이션

### 필요한 테이블

v13에서 사용하는 테이블들을 v14 스키마에 맞게 생성:

```sql
-- 1. Fetcher 관련
CREATE TABLE IF NOT EXISTS data.fetch_jobs (
    job_id UUID PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,  -- price, flow, disclosure, marketcap
    status VARCHAR(20) NOT NULL,    -- pending, running, completed, failed
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 2. Universe 스냅샷
CREATE TABLE IF NOT EXISTS market.universe_snapshots (
    snapshot_id VARCHAR(20) PRIMARY KEY,
    generated_at TIMESTAMP NOT NULL,
    total_count INT NOT NULL,
    holdings JSONB NOT NULL,
    watchlist JSONB NOT NULL,
    rankings JSONB NOT NULL,
    filter_stats JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 3. Signal 스냅샷
CREATE TABLE IF NOT EXISTS signals.snapshots (
    snapshot_id VARCHAR(20) PRIMARY KEY,
    generated_at TIMESTAMP NOT NULL,
    total_count INT NOT NULL,
    buy_count INT NOT NULL,
    sell_count INT NOT NULL,
    signals JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 4. Ranking 스냅샷
CREATE TABLE IF NOT EXISTS ranking.snapshots (
    snapshot_id VARCHAR(20) PRIMARY KEY,
    signal_id VARCHAR(20) NOT NULL,
    generated_at TIMESTAMP NOT NULL,
    total_count INT NOT NULL,
    selected_count INT NOT NULL,
    rankings JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 5. Audit 관련
CREATE TABLE IF NOT EXISTS audit.performance_snapshots (
    snapshot_id VARCHAR(20) PRIMARY KEY,
    period VARCHAR(10) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_return DECIMAL(10,4),
    annual_return DECIMAL(10,4),
    volatility DECIMAL(10,4),
    sharpe DECIMAL(10,4),
    max_drawdown DECIMAL(10,4),
    win_rate DECIMAL(10,4),
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 📡 API 엔드포인트

### 추가할 엔드포인트

```
# Fetcher
POST /api/v1/fetcher/collect           # 데이터 수집 트리거
GET  /api/v1/fetcher/status            # 수집 상태 조회
GET  /api/v1/fetcher/jobs              # 수집 작업 목록

# Universe
GET  /api/v1/universe/latest           # 최신 유니버스
GET  /api/v1/universe/snapshots/:id    # 특정 스냅샷
GET  /api/v1/universe/symbols          # 종목 코드 목록

# Signals
GET  /api/v1/signals/latest            # 최신 신호
GET  /api/v1/signals/snapshots/:id     # 특정 스냅샷
GET  /api/v1/signals/stock/:symbol     # 종목별 신호

# Ranking
GET  /api/v1/ranking/latest            # 최신 랭킹
GET  /api/v1/ranking/selected          # 선정 종목만
GET  /api/v1/ranking/snapshots/:id     # 특정 스냅샷

# Audit
GET  /api/v1/audit/performance         # 성과 리포트
GET  /api/v1/audit/risk                # 리스크 리포트
GET  /api/v1/audit/attribution         # 성과 귀인
```

---

## 🔧 CLI 명령어

### 추가할 명령어

```bash
# Fetcher
go run ./cmd/quant fetcher collect all     # 전체 수집
go run ./cmd/quant fetcher collect naver   # Naver만
go run ./cmd/quant fetcher collect dart    # DART만
go run ./cmd/quant fetcher marketcap       # 시가총액만

# Universe
go run ./cmd/quant universe build          # 유니버스 생성
go run ./cmd/quant universe list           # 현재 유니버스

# Signals
go run ./cmd/quant signals generate        # 신호 생성
go run ./cmd/quant signals show            # 신호 조회

# Ranking
go run ./cmd/quant ranking generate        # 랭킹 생성
go run ./cmd/quant ranking show            # 랭킹 조회

# Audit
go run ./cmd/quant audit performance       # 성과 분석
go run ./cmd/quant audit risk              # 리스크 분석
```

---

## ✅ 마이그레이션 체크리스트

### Phase 1: Fetcher (P1)

```
⬜ external 클라이언트 복사
⬜ domain/fetcher 모델 생성
⬜ service/fetcher 구현
⬜ infrastructure/postgres/fetcher 구현
⬜ CLI 명령어 추가
⬜ 테스트 통과 확인
```

### Phase 2: Universe (P1)

```
⬜ domain/universe 모델 확인/수정
⬜ service/universe 필터 로직 이전
⬜ SPAC 패턴 이전
⬜ 테스트 통과 확인
```

### Phase 3: Signals (P1)

```
⬜ 6개 팩터 계산기 이전
⬜ 이벤트 매핑 로직 이전
⬜ 오케스트레이션 구현
⬜ 테스트 통과 확인
```

### Phase 4: Ranking (P1)

```
⬜ v14 설계 기반 구현
⬜ Signals 연동
⬜ 테스트 통과 확인
```

### Phase 5: Audit (P2)

```
⬜ 성과 계산 로직 이전
⬜ 리스크 계산 로직 이전
⬜ API 엔드포인트 추가
⬜ 테스트 통과 확인
```

---

## ⚠️ 주의사항

### 1. v14 기존 코드 수정 금지

v14에 이미 구현된 모듈들은 수정하지 않음:
- `infra/database`
- `domain/stock`
- `service/price`
- 기존 API 핸들러들

### 2. v14 패턴 준수

새로 추가하는 코드는 v14 패턴을 따름:
- Domain → Service → Infrastructure 레이어
- Repository 인터페이스 패턴
- 에러 처리 패턴

### 3. 점진적 이전

한 번에 모든 것을 이전하지 않고, 모듈 단위로 순차적으로:
1. Fetcher (데이터 없으면 다른 모듈 동작 불가)
2. Universe (Signals 전제조건)
3. Signals (Ranking 전제조건)
4. Ranking
5. Audit

---

## 📚 참조

### v13 소스 코드

```
/Users/wonny/Dev/aegis/v13/backend/internal/
├── s0_data/           # 데이터 수집
├── s1_universe/       # 유니버스
├── s2_signals/        # 신호
├── audit/             # 성과 분석
└── external/          # 외부 API
```

### v14 설계 문서

```
/Users/wonny/Dev/aegis/v14/docs/modules/
├── universe.md        # 유니버스 설계
├── signals.md         # 신호 설계
├── ranking.md         # 랭킹 설계
└── portfolio.md       # 포트폴리오 설계
```

---

**Version**: 1.0.0
**Last Updated**: 2026-01-17
