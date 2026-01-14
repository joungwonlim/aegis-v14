# Portfolio 모듈 설계

> **목적**: Ranking 결과를 바탕으로 실제 투자 포트폴리오를 구성합니다.

**Last Updated**: 2026-01-14

---

## 📋 개요

### 책임 (Responsibility)
- Ranking 상위 종목 선택
- 종목별 투자 비중 계산
- 리스크 제약 조건 적용
- 포트폴리오 스냅샷 생성 및 저장

### 위치 (Location)
```
backend/internal/strategy/portfolio/
├── service.go        # 포트폴리오 구성 로직
├── types.go          # 도메인 모델
├── repository.go     # DB 접근
└── handler.go        # HTTP 핸들러 (API Layer에서 호출)
```

### 의존성 (Dependencies)
- `strategy.ranking` (RankingService) - Ranking 결과 조회
- `infra.database` (Repository) - 포트폴리오 저장/조회

### v10과의 차이점
| 항목 | v10 | v14 |
|------|-----|-----|
| **포지션 사이징** | Vol Targeting + Forecast | Score-weighted (단순화) |
| **시장 국면 대응** | Regime Multiplier (4단계) | Market Regime Gate (2단계, Reentry에서 구현) |
| **AI 관여** | AI Thesis 반영 | 없음 (100% 규칙 기반) |
| **복잡도** | 높음 (321 lines) | 낮음 (핵심 로직만) |

---

## 🎯 핵심 설계 결정

### 1. 포지션 할당 방식
```
Equal-Weight vs Score-Weighted

선택: Equal-Weight (Phase 1)
이유:
- 단순하고 이해하기 쉬움
- 백테스트 결과 유의미한 차이 없음
- Score-Weighted는 Phase 2에서 선택적 추가
```

### 2. 포트폴리오 크기
```
Target Portfolio Size: 10-15 종목

근거:
- 10종목 미만: 분산 부족, 개별 종목 리스크 높음
- 15종목 초과: 관리 복잡도 증가, 성과 희석
- Ranking Top 20 중 리스크 제약 통과한 10-15개 선택
```

### 3. 리밸런싱 주기
```
주기: 1주 (매주 월요일 09:10 KST)

근거:
- 너무 빈번: 거래 비용 증가, Whipsaw
- 너무 느림: 시장 변화 대응 느림
- 1주는 균형점 (v10 검증 완료)
```

---

## 📐 도메인 모델

### Portfolio
```go
// Portfolio 포트폴리오
type Portfolio struct {
    ID              uuid.UUID           `json:"id"`
    SnapshotID      uuid.UUID           `json:"snapshot_id"`       // Ranking Snapshot ID
    CreatedAt       time.Time           `json:"created_at"`

    // 구성
    Holdings        []Holding           `json:"holdings"`          // 보유 종목
    TotalWeight     float64             `json:"total_weight"`      // 총 비중 (100% 목표)

    // 통계
    Stats           PortfolioStats      `json:"stats"`

    // 메타데이터
    Status          PortfolioStatus     `json:"status"`            // DRAFT, ACTIVE, ARCHIVED
    Notes           string              `json:"notes,omitempty"`
}

// Holding 보유 종목
type Holding struct {
    Symbol          string              `json:"symbol"`
    Name            string              `json:"name"`

    // 비중
    TargetWeight    float64             `json:"target_weight"`     // 목표 비중 (%)

    // Ranking 정보
    TotalScore      float64             `json:"total_score"`       // Ranking Total Score
    AlphaScore      float64             `json:"alpha_score"`       // Signal Strength
    RiskScore       float64             `json:"risk_score"`        // Risk Score

    // 제약 적용 여부
    Capped          bool                `json:"capped"`            // 한도 적용 여부
    CappedReason    string              `json:"capped_reason,omitempty"`

    // 메타
    Sector          string              `json:"sector"`
    Market          string              `json:"market"`
}

// PortfolioStats 포트폴리오 통계
type PortfolioStats struct {
    TotalHoldings   int                 `json:"total_holdings"`
    AvgWeight       float64             `json:"avg_weight"`
    MaxWeight       float64             `json:"max_weight"`
    MinWeight       float64             `json:"min_weight"`

    // 분산도
    SectorCount     map[string]int      `json:"sector_count"`
    MarketCount     map[string]int      `json:"market_count"`

    // 점수 분포
    AvgTotalScore   float64             `json:"avg_total_score"`
    AvgRiskScore    float64             `json:"avg_risk_score"`
}

// PortfolioStatus 포트폴리오 상태
type PortfolioStatus string

const (
    PortfolioStatusDraft    PortfolioStatus = "DRAFT"    // 생성 중
    PortfolioStatusActive   PortfolioStatus = "ACTIVE"   // 활성
    PortfolioStatusArchived PortfolioStatus = "ARCHIVED" // 아카이브
)
```

---

## 🔧 Service Layer

### PortfolioService Interface
```go
// PortfolioService 포트폴리오 구성 서비스
type PortfolioService interface {
    // GeneratePortfolio Ranking 결과로부터 포트폴리오 생성
    GeneratePortfolio(ctx context.Context, snapshotID uuid.UUID) (*Portfolio, error)

    // GetLatestPortfolio 최신 활성 포트폴리오 조회
    GetLatestPortfolio(ctx context.Context) (*Portfolio, error)

    // GetPortfolio 특정 포트폴리오 조회
    GetPortfolio(ctx context.Context, id uuid.UUID) (*Portfolio, error)

    // ListPortfolios 포트폴리오 목록 조회
    ListPortfolios(ctx context.Context, filters ListFilters) ([]Portfolio, error)

    // ActivatePortfolio 포트폴리오 활성화
    ActivatePortfolio(ctx context.Context, id uuid.UUID) error
}
```

### 포트폴리오 생성 알고리즘

#### Step 1: Ranking 결과 조회
```go
// generatePortfolio 포트폴리오 생성
func (s *Service) GeneratePortfolio(ctx context.Context, snapshotID uuid.UUID) (*Portfolio, error) {
    // 1. Ranking Snapshot 조회
    rankingSnapshot, err := s.rankingService.GetSnapshot(ctx, snapshotID)
    if err != nil {
        return nil, fmt.Errorf("failed to get ranking snapshot: %w", err)
    }

    // 2. 선택 기준 적용
    candidates := s.selectCandidates(rankingSnapshot.Stocks)

    // 3. 비중 할당
    holdings := s.allocateWeights(candidates)

    // 4. 제약 조건 적용
    holdings = s.applyConstraints(holdings)

    // 5. 정규화 (총 비중 100%)
    holdings = s.normalizeWeights(holdings)

    // 6. 통계 계산
    stats := s.calculateStats(holdings)

    // 7. Portfolio 생성
    portfolio := &Portfolio{
        ID:          uuid.New(),
        SnapshotID:  snapshotID,
        CreatedAt:   time.Now(),
        Holdings:    holdings,
        TotalWeight: sumWeights(holdings),
        Stats:       stats,
        Status:      PortfolioStatusDraft,
    }

    // 8. 저장
    if err := s.repo.Save(ctx, portfolio); err != nil {
        return nil, fmt.Errorf("failed to save portfolio: %w", err)
    }

    return portfolio, nil
}
```

#### Step 2: 후보 선택
```go
// selectCandidates 포트폴리오 후보 선택
func (s *Service) selectCandidates(stocks []ranking.RankedStock) []ranking.RankedStock {
    var candidates []ranking.RankedStock

    for _, stock := range stocks {
        // 선택된 종목만
        if !stock.Selected {
            continue
        }

        // 최소 점수 충족 (Ranking에서 이미 필터링되었지만 재확인)
        if stock.TotalScore < s.criteria.MinScore {
            continue
        }

        candidates = append(candidates, stock)
    }

    // 점수 내림차순 정렬
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].TotalScore > candidates[j].TotalScore
    })

    // Top N개 선택
    if len(candidates) > s.criteria.MaxHoldings {
        candidates = candidates[:s.criteria.MaxHoldings]
    }

    return candidates
}
```

#### Step 3: 비중 할당
```go
// allocateWeights 비중 할당
func (s *Service) allocateWeights(candidates []ranking.RankedStock) []Holding {
    holdings := make([]Holding, 0, len(candidates))

    switch s.criteria.AllocationMethod {
    case AllocationMethodEqualWeight:
        // Equal-Weight: 균등 배분
        equalWeight := 100.0 / float64(len(candidates))

        for _, stock := range candidates {
            holdings = append(holdings, Holding{
                Symbol:       stock.Symbol,
                Name:         stock.Name,
                TargetWeight: equalWeight,
                TotalScore:   stock.TotalScore,
                AlphaScore:   stock.AlphaScore,
                RiskScore:    stock.RiskScore,
                Sector:       stock.Sector,
                Market:       stock.Market,
                Capped:       false,
            })
        }

    case AllocationMethodScoreWeighted:
        // Score-Weighted: 점수 기반 가중치 (Phase 2)
        totalScore := 0.0
        for _, stock := range candidates {
            totalScore += stock.TotalScore
        }

        for _, stock := range candidates {
            weight := (stock.TotalScore / totalScore) * 100.0
            holdings = append(holdings, Holding{
                Symbol:       stock.Symbol,
                Name:         stock.Name,
                TargetWeight: weight,
                TotalScore:   stock.TotalScore,
                AlphaScore:   stock.AlphaScore,
                RiskScore:    stock.RiskScore,
                Sector:       stock.Sector,
                Market:       stock.Market,
                Capped:       false,
            })
        }
    }

    return holdings
}
```

#### Step 4: 제약 조건 적용
```go
// applyConstraints 제약 조건 적용
func (s *Service) applyConstraints(holdings []Holding) []Holding {
    // 1. 단일 종목 한도
    for i := range holdings {
        if holdings[i].TargetWeight > s.criteria.MaxSingleWeight {
            holdings[i].TargetWeight = s.criteria.MaxSingleWeight
            holdings[i].Capped = true
            holdings[i].CappedReason = fmt.Sprintf("Single position limit (%.1f%%)", s.criteria.MaxSingleWeight)
        }
    }

    // 2. 섹터 한도
    sectorWeights := make(map[string]float64)
    for _, h := range holdings {
        sectorWeights[h.Sector] += h.TargetWeight
    }

    for sector, totalWeight := range sectorWeights {
        if totalWeight > s.criteria.MaxSectorWeight {
            // 섹터 내 종목들의 비중을 비례적으로 감소
            ratio := s.criteria.MaxSectorWeight / totalWeight
            for i := range holdings {
                if holdings[i].Sector == sector {
                    holdings[i].TargetWeight *= ratio
                    holdings[i].Capped = true
                    holdings[i].CappedReason = fmt.Sprintf("Sector limit (%.1f%%)", s.criteria.MaxSectorWeight)
                }
            }
        }
    }

    return holdings
}
```

#### Step 5: 정규화
```go
// normalizeWeights 비중 정규화 (총합 100%)
func (s *Service) normalizeWeights(holdings []Holding) []Holding {
    totalWeight := 0.0
    for _, h := range holdings {
        totalWeight += h.TargetWeight
    }

    if totalWeight == 0 {
        return holdings
    }

    // 100%로 정규화
    factor := 100.0 / totalWeight
    for i := range holdings {
        holdings[i].TargetWeight *= factor
    }

    return holdings
}
```

---

## ⚙️ 설정 (Configuration)

### PortfolioCriteria
```go
// PortfolioCriteria 포트폴리오 구성 기준
type PortfolioCriteria struct {
    // 종목 수
    MinHoldings      int                 `json:"min_holdings"`      // 최소 보유 종목 (기본: 10)
    MaxHoldings      int                 `json:"max_holdings"`      // 최대 보유 종목 (기본: 15)

    // 비중 제약
    MaxSingleWeight  float64             `json:"max_single_weight"` // 단일 종목 최대 비중 (기본: 15%)
    MaxSectorWeight  float64             `json:"max_sector_weight"` // 섹터 최대 비중 (기본: 40%)

    // 선택 기준
    MinScore         float64             `json:"min_score"`         // 최소 점수 (기본: 60)

    // 할당 방식
    AllocationMethod AllocationMethod    `json:"allocation_method"` // EQUAL_WEIGHT, SCORE_WEIGHTED
}

// AllocationMethod 할당 방식
type AllocationMethod string

const (
    AllocationMethodEqualWeight   AllocationMethod = "EQUAL_WEIGHT"    // 균등 배분
    AllocationMethodScoreWeighted AllocationMethod = "SCORE_WEIGHTED"  // 점수 가중
)

// DefaultPortfolioCriteria 기본 설정
func DefaultPortfolioCriteria() *PortfolioCriteria {
    return &PortfolioCriteria{
        MinHoldings:      10,
        MaxHoldings:      15,
        MaxSingleWeight:  15.0,  // 15%
        MaxSectorWeight:  40.0,  // 40%
        MinScore:         60.0,
        AllocationMethod: AllocationMethodEqualWeight,
    }
}
```

---

## 💾 Database Schema

### portfolio.portfolios
```sql
CREATE TABLE portfolio.portfolios (
    -- 기본 정보
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id     UUID NOT NULL,                          -- ranking.snapshots FK
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 구성
    holdings        JSONB NOT NULL,                         -- Holding[]
    total_weight    NUMERIC(5,2) NOT NULL,                  -- 총 비중 (100.00 목표)

    -- 통계
    stats           JSONB NOT NULL,                         -- PortfolioStats

    -- 메타데이터
    status          VARCHAR(20) NOT NULL DEFAULT 'DRAFT',   -- DRAFT, ACTIVE, ARCHIVED
    notes           TEXT,

    -- 제약 조건
    CONSTRAINT fk_snapshot FOREIGN KEY (snapshot_id)
        REFERENCES ranking.snapshots(id) ON DELETE RESTRICT,
    CONSTRAINT check_total_weight CHECK (total_weight >= 0 AND total_weight <= 100)
);

-- 인덱스
CREATE INDEX idx_portfolios_snapshot ON portfolio.portfolios(snapshot_id);
CREATE INDEX idx_portfolios_created_at ON portfolio.portfolios(created_at DESC);
CREATE INDEX idx_portfolios_status ON portfolio.portfolios(status);
CREATE INDEX idx_portfolios_active ON portfolio.portfolios(status, created_at DESC)
    WHERE status = 'ACTIVE';
```

### JSONB 구조 예시

#### holdings
```json
[
  {
    "symbol": "005930",
    "name": "삼성전자",
    "target_weight": 10.5,
    "total_score": 82.3,
    "alpha_score": 85.0,
    "risk_score": 25.0,
    "capped": false,
    "capped_reason": "",
    "sector": "반도체",
    "market": "KOSPI"
  },
  {
    "symbol": "000660",
    "name": "SK하이닉스",
    "target_weight": 15.0,
    "total_score": 79.8,
    "alpha_score": 82.0,
    "risk_score": 28.0,
    "capped": true,
    "capped_reason": "Single position limit (15.0%)",
    "sector": "반도체",
    "market": "KOSPI"
  }
]
```

#### stats
```json
{
  "total_holdings": 12,
  "avg_weight": 8.33,
  "max_weight": 15.0,
  "min_weight": 5.2,
  "sector_count": {
    "반도체": 3,
    "IT": 2,
    "금융": 2,
    "바이오": 2,
    "자동차": 1,
    "화학": 1,
    "건설": 1
  },
  "market_count": {
    "KOSPI": 10,
    "KOSDAQ": 2
  },
  "avg_total_score": 75.6,
  "avg_risk_score": 28.4
}
```

---

## 🔗 API Layer

### HTTP Endpoints

#### POST /api/v1/portfolio/generate
**목적**: Ranking 결과로부터 포트폴리오 생성

**Request**:
```json
{
  "snapshot_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

**Response 200**:
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "snapshot_id": "550e8400-e29b-41d4-a716-446655440001",
    "created_at": "2026-01-14T15:00:00Z",
    "holdings": [
      {
        "symbol": "005930",
        "name": "삼성전자",
        "target_weight": 10.5,
        "total_score": 82.3,
        "alpha_score": 85.0,
        "risk_score": 25.0,
        "capped": false,
        "sector": "반도체",
        "market": "KOSPI"
      }
    ],
    "total_weight": 100.0,
    "stats": {
      "total_holdings": 12,
      "avg_weight": 8.33,
      "sector_count": {
        "반도체": 3,
        "IT": 2
      }
    },
    "status": "DRAFT"
  }
}
```

**Errors**:
- `400`: Invalid snapshot_id
- `404`: Ranking snapshot not found
- `500`: Internal server error

---

#### GET /api/v1/portfolio/latest
**목적**: 최신 활성 포트폴리오 조회

**Response 200**:
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "snapshot_id": "550e8400-e29b-41d4-a716-446655440001",
    "created_at": "2026-01-14T15:00:00Z",
    "holdings": [...],
    "total_weight": 100.0,
    "stats": {...},
    "status": "ACTIVE"
  }
}
```

**Errors**:
- `404`: No active portfolio found
- `500`: Internal server error

---

#### GET /api/v1/portfolio/:id
**목적**: 특정 포트폴리오 조회

**Response 200**:
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "holdings": [...],
    "stats": {...}
  }
}
```

**Errors**:
- `404`: Portfolio not found
- `500`: Internal server error

---

#### GET /api/v1/portfolio
**목적**: 포트폴리오 목록 조회

**Query Parameters**:
- `status` (optional): 상태 필터 (DRAFT, ACTIVE, ARCHIVED)
- `from` (optional): 시작일 (RFC3339)
- `to` (optional): 종료일 (RFC3339)
- `page` (optional): 페이지 번호 (기본: 1)
- `limit` (optional): 페이지 크기 (기본: 20, 최대: 100)

**Response 200**:
```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "created_at": "2026-01-14T15:00:00Z",
      "total_weight": 100.0,
      "stats": {
        "total_holdings": 12
      },
      "status": "ACTIVE"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45
  }
}
```

---

#### POST /api/v1/portfolio/:id/activate
**목적**: 포트폴리오 활성화

**Response 200**:
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "status": "ACTIVE"
  }
}
```

**Business Logic**:
1. 기존 ACTIVE 포트폴리오를 ARCHIVED로 변경
2. 대상 포트폴리오를 ACTIVE로 변경

**Errors**:
- `404`: Portfolio not found
- `409`: Portfolio already active
- `500`: Internal server error

---

## 📊 예시 시나리오

### 시나리오 1: 균등 배분 (Equal-Weight)

**입력**: Ranking Top 20 종목

**설정**:
- MaxHoldings: 12
- AllocationMethod: EQUAL_WEIGHT
- MaxSingleWeight: 15%
- MaxSectorWeight: 40%

**출력**:
```
총 12종목, 각 8.33% 균등 배분

종목       | 비중    | 점수 | 섹터   | 제약 적용
---------|--------|-----|--------|----------
삼성전자   | 8.33%  | 85  | 반도체  | -
SK하이닉스 | 8.33%  | 82  | 반도체  | -
NAVER     | 8.33%  | 78  | IT     | -
...       | ...    | ... | ...    | ...

총 비중: 100.00%
섹터별 분산: 반도체 3종목 (25%), IT 2종목 (16.7%), ...
```

---

### 시나리오 2: 섹터 한도 적용

**입력**: Ranking Top 20 종목 (반도체 5종목 포함)

**설정**:
- MaxHoldings: 15
- MaxSectorWeight: 40%

**결과**:
```
반도체 섹터 5종목 → 총 비중 41.67% (초과)
→ 40%로 캡핑
→ 각 종목 8.33% → 8.0%로 조정
→ 여유 비중 1.67%를 다른 종목에 재분배

최종:
반도체 5종목: 40% (8% × 5)
IT 3종목: 24% (8% × 3)
기타 7종목: 36% (조정된 비중)
```

---

### 시나리오 3: 단일 종목 한도 적용

**입력**: Score-Weighted, 최고 점수 종목 95점

**설정**:
- AllocationMethod: SCORE_WEIGHTED
- MaxSingleWeight: 15%

**결과**:
```
원래 계산 비중: 18.5% (점수 기반)
→ 15%로 캡핑
→ 여유 3.5%를 다른 종목에 재분배

종목       | 원래 비중 | 최종 비중 | 제약
---------|---------|---------|-----
삼성전자   | 18.5%   | 15.0%   | Capped (Single limit)
SK하이닉스 | 15.2%   | 15.0%   | Capped (Single limit)
NAVER     | 12.8%   | 13.5%   | 재분배
...       | ...     | ...     | ...
```

---

## 🔒 제약 조건 및 검증

### 1. 입력 검증
```go
// validateInput 입력 검증
func (s *Service) validateInput(snapshotID uuid.UUID) error {
    // Snapshot 존재 확인
    snapshot, err := s.rankingService.GetSnapshot(ctx, snapshotID)
    if err != nil {
        return fmt.Errorf("invalid snapshot_id: %w", err)
    }

    // 선택된 종목 수 확인
    selectedCount := 0
    for _, stock := range snapshot.Stocks {
        if stock.Selected {
            selectedCount++
        }
    }

    if selectedCount < s.criteria.MinHoldings {
        return fmt.Errorf("insufficient holdings: got %d, need at least %d",
            selectedCount, s.criteria.MinHoldings)
    }

    return nil
}
```

### 2. 비즈니스 룰
```go
// Business Rules
const (
    // 종목 수 제약
    AbsoluteMinHoldings = 5   // 절대 최소 (너무 적으면 위험)
    AbsoluteMaxHoldings = 20  // 절대 최대 (너무 많으면 희석)

    // 비중 제약
    AbsoluteMaxSingleWeight = 30.0  // 단일 종목 절대 최대 (30%)
    AbsoluteMaxSectorWeight = 60.0  // 섹터 절대 최대 (60%)

    // 점수 제약
    AbsoluteMinScore = 40.0  // 절대 최소 점수 (너무 낮으면 제외)
)
```

### 3. 출력 검증
```go
// validateOutput 출력 검증
func (s *Service) validateOutput(portfolio *Portfolio) error {
    // 총 비중 100% 확인
    totalWeight := 0.0
    for _, h := range portfolio.Holdings {
        totalWeight += h.TargetWeight
    }

    tolerance := 0.01 // 0.01% 허용 오차
    if math.Abs(totalWeight-100.0) > tolerance {
        return fmt.Errorf("total weight must be 100%%, got %.2f%%", totalWeight)
    }

    // 종목 수 확인
    if len(portfolio.Holdings) < s.criteria.MinHoldings {
        return fmt.Errorf("insufficient holdings: %d", len(portfolio.Holdings))
    }

    // 단일 종목 한도 확인
    for _, h := range portfolio.Holdings {
        if h.TargetWeight > AbsoluteMaxSingleWeight {
            return fmt.Errorf("holding %s exceeds max weight: %.2f%%", h.Symbol, h.TargetWeight)
        }
    }

    return nil
}
```

---

## 🚨 에러 처리

### Error Types
```go
var (
    ErrInvalidSnapshotID    = errors.New("invalid snapshot_id")
    ErrSnapshotNotFound     = errors.New("ranking snapshot not found")
    ErrInsufficientHoldings = errors.New("insufficient holdings")
    ErrInvalidWeight        = errors.New("invalid weight")
    ErrPortfolioNotFound    = errors.New("portfolio not found")
    ErrAlreadyActive        = errors.New("portfolio already active")
)
```

### Error Handling
```go
// GeneratePortfolio 에러 처리
func (s *Service) GeneratePortfolio(ctx context.Context, snapshotID uuid.UUID) (*Portfolio, error) {
    // 입력 검증
    if err := s.validateInput(snapshotID); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // 로직 실행 (panic 방지)
    defer func() {
        if r := recover(); r != nil {
            log.Error().
                Interface("panic", r).
                Str("snapshot_id", snapshotID.String()).
                Msg("portfolio generation panicked")
        }
    }()

    // ... 포트폴리오 생성 로직 ...

    // 출력 검증
    if err := s.validateOutput(portfolio); err != nil {
        return nil, fmt.Errorf("output validation failed: %w", err)
    }

    return portfolio, nil
}
```

---

## 📈 성능 고려사항

### 1. 처리 성능
```
예상 처리 시간: < 100ms (Ranking Top 20 → Portfolio 12종목)

병목 지점:
1. Ranking Snapshot 조회: ~20ms (DB 쿼리)
2. 비중 계산: ~10ms (CPU)
3. 제약 적용: ~30ms (CPU)
4. 저장: ~40ms (DB 삽입)

최적화 전략:
- Ranking Snapshot 캐싱 (Redis, TTL 1시간)
- 비중 계산 병렬화 (goroutine)
```

### 2. 데이터베이스
```sql
-- 쿼리 성능 목표
-- GetLatestPortfolio: < 50ms
-- ListPortfolios: < 100ms (20개 페이지)

-- 인덱스 전략
CREATE INDEX idx_portfolios_active ON portfolio.portfolios(status, created_at DESC)
    WHERE status = 'ACTIVE';
-- → GetLatestPortfolio 가속

CREATE INDEX idx_portfolios_created_at ON portfolio.portfolios(created_at DESC);
-- → ListPortfolios 가속
```

---

## 🔄 배치 작업

### 주간 리밸런싱
```go
// RebalancePortfolio 주간 리밸런싱
// 매주 월요일 09:10 KST 실행
func (s *Service) RebalancePortfolio(ctx context.Context) error {
    // 1. 최신 Ranking Snapshot 조회
    snapshot, err := s.rankingService.GetLatestSnapshot(ctx)
    if err != nil {
        return fmt.Errorf("failed to get latest ranking: %w", err)
    }

    // 2. 새 포트폴리오 생성
    newPortfolio, err := s.GeneratePortfolio(ctx, snapshot.ID)
    if err != nil {
        return fmt.Errorf("failed to generate portfolio: %w", err)
    }

    // 3. 활성화
    if err := s.ActivatePortfolio(ctx, newPortfolio.ID); err != nil {
        return fmt.Errorf("failed to activate portfolio: %w", err)
    }

    log.Info().
        Str("portfolio_id", newPortfolio.ID.String()).
        Int("holdings", len(newPortfolio.Holdings)).
        Float64("total_weight", newPortfolio.TotalWeight).
        Msg("weekly rebalance completed")

    return nil
}
```

---

## 🧪 테스트 전략

### 1. 단위 테스트
```go
func TestAllocateWeights_EqualWeight(t *testing.T) {
    // Given
    service := NewService(...)
    candidates := []ranking.RankedStock{
        {Symbol: "005930", TotalScore: 85},
        {Symbol: "000660", TotalScore: 82},
        {Symbol: "035420", TotalScore: 78},
    }

    // When
    holdings := service.allocateWeights(candidates)

    // Then
    assert.Len(t, holdings, 3)
    for _, h := range holdings {
        assert.InDelta(t, 33.33, h.TargetWeight, 0.01)
    }
}

func TestApplyConstraints_SinglePositionLimit(t *testing.T) {
    // Given
    service := NewService(...)
    service.criteria.MaxSingleWeight = 15.0

    holdings := []Holding{
        {Symbol: "005930", TargetWeight: 20.0},
    }

    // When
    result := service.applyConstraints(holdings)

    // Then
    assert.Equal(t, 15.0, result[0].TargetWeight)
    assert.True(t, result[0].Capped)
}
```

### 2. 통합 테스트
```go
func TestGeneratePortfolio_Integration(t *testing.T) {
    // Given
    db := setupTestDB(t)
    rankingService := ranking.NewService(...)
    portfolioService := NewService(rankingService, db)

    // Create test ranking snapshot
    snapshot := createTestRankingSnapshot(t, db)

    // When
    portfolio, err := portfolioService.GeneratePortfolio(context.Background(), snapshot.ID)

    // Then
    require.NoError(t, err)
    assert.NotNil(t, portfolio)
    assert.Equal(t, 100.0, portfolio.TotalWeight)
    assert.GreaterOrEqual(t, len(portfolio.Holdings), 10)
    assert.LessOrEqual(t, len(portfolio.Holdings), 15)
}
```

### 3. E2E 테스트
```bash
# 전체 파이프라인 테스트
Universe → Signals → Ranking → Portfolio

# 예상 결과:
# - Universe: 150종목
# - Signals: 80종목 (BUY)
# - Ranking: Top 20 선정
# - Portfolio: 12종목, 100% 배분
```

---

## 📝 운영 체크리스트

### 배포 전 확인
- [ ] Ranking Service 연동 테스트
- [ ] DB 스키마 마이그레이션
- [ ] API 엔드포인트 테스트
- [ ] 비중 계산 정확도 검증
- [ ] 제약 조건 적용 확인

### 배포 후 모니터링
- [ ] 포트폴리오 생성 성공률 (목표: 99%)
- [ ] 생성 시간 (목표: < 100ms)
- [ ] 총 비중 100% 달성 (목표: 100%)
- [ ] 제약 조건 위반 0건 (목표: 0건)

### 주간 점검
- [ ] 리밸런싱 자동 실행 확인
- [ ] 포트폴리오 활성화 상태 확인
- [ ] 히스토리 아카이브 확인

---

## 🔗 의존성 인터페이스

### RankingService (입력)
```go
// Portfolio가 의존하는 Ranking 인터페이스
type RankingService interface {
    GetLatestSnapshot(ctx context.Context) (*ranking.Snapshot, error)
    GetSnapshot(ctx context.Context, id uuid.UUID) (*ranking.Snapshot, error)
}
```

### Repository (저장)
```go
// Portfolio Repository 인터페이스
type Repository interface {
    Save(ctx context.Context, portfolio *Portfolio) error
    FindByID(ctx context.Context, id uuid.UUID) (*Portfolio, error)
    FindLatestActive(ctx context.Context) (*Portfolio, error)
    List(ctx context.Context, filters ListFilters) ([]Portfolio, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status PortfolioStatus) error
}
```

---

## 🚀 향후 확장 계획

### Phase 2: Score-Weighted 할당
```go
// 점수 기반 가중치 할당
// 높은 점수 = 높은 비중

예:
종목A (점수 90) → 12%
종목B (점수 80) → 10%
종목C (점수 70) → 8%
```

### Phase 3: Dynamic Rebalancing
```go
// 시장 변동성에 따른 동적 리밸런싱
// 변동성 높을 때: 보유 종목 수 증가 (분산 강화)
// 변동성 낮을 때: 보유 종목 수 감소 (집중 강화)
```

### Phase 4: Tax-Loss Harvesting
```go
// 세금 최적화를 위한 손실 실현
// 연말 리밸런싱 시 손실 종목 우선 매도
```

---

## 📚 참고 문서

- [Ranking 모듈 설계](./ranking.md)
- [Database Schema](../database/schema.md)
- [API 공통 스펙](../api/common.md)

---

**Version**: 1.0.0
**Author**: Aegis Team
**Status**: ✅ 설계 완료
