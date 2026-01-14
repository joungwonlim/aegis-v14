# Ranking API

> 종합 점수 및 랭킹 조회 API

**Last Updated**: 2026-01-14

---

## 개요

리스크 조정 종합 점수 및 랭킹 조회 API. strategy.rankings 테이블의 데이터를 제공합니다.

**Base URL**: `/api/v1/ranking`

**인증**: (추후 추가)

**특징**:
- Signals 결과 기반 리스크 조정 점수 산출
- Alpha Score (70%) + Risk Score (30%) 조합
- 다양성 제약 (섹터/시장 집중도) 적용
- Top 20 종목 선정
- 1시간마다 자동 생성

---

## 엔드포인트

### 1. 최신 랭킹 스냅샷 조회

**GET** `/api/v1/ranking/latest`

최신 생성된 랭킹 스냅샷을 조회합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| selected_only | boolean | N | false | 선정된 종목만 반환 (true/false) |
| limit | integer | N | 20 | 반환할 종목 수 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/ranking/latest?selected_only=true
```

#### Response (200 OK)

```json
{
  "data": {
    "snapshot_id": "20260114-1030",
    "signal_snapshot_id": "20260114-1030",
    "generated_at": "2026-01-14T10:30:00Z",
    "total_count": 80,
    "selected_count": 20,
    "stocks": [
      {
        "symbol": "005930",
        "name": "삼성전자",
        "rank": 1,
        "total_score": 82.3,
        "alpha_score": 85.0,
        "risk_score": 25.0,
        "adjustment": -2.7,
        "selected": true,
        "breakdown": {
          "base_alpha": 85.0,
          "base_risk": 25.0,
          "sector_penalty": 0.0,
          "market_penalty": 0.0
        },
        "risk_factors": {
          "volatility": 18.5,
          "liquidity": 95.0,
          "volatility_score": 72.0,
          "liquidity_score": 95.0
        },
        "sector": "반도체",
        "market": "KOSPI",
        "signal_type": "BUY",
        "signal_strength": 85
      },
      {
        "symbol": "000660",
        "name": "SK하이닉스",
        "rank": 2,
        "total_score": 79.8,
        "alpha_score": 82.0,
        "risk_score": 28.0,
        "adjustment": -2.2,
        "selected": true,
        "breakdown": {
          "base_alpha": 82.0,
          "base_risk": 28.0,
          "sector_penalty": 0.0,
          "market_penalty": 0.0
        },
        "risk_factors": {
          "volatility": 22.3,
          "liquidity": 88.0,
          "volatility_score": 65.0,
          "liquidity_score": 88.0
        },
        "sector": "반도체",
        "market": "KOSPI",
        "signal_type": "BUY",
        "signal_strength": 82
      }
    ],
    "stats": {
      "avg_total_score": 75.6,
      "avg_alpha_score": 77.2,
      "avg_risk_score": 28.4,
      "sector_distribution": {
        "반도체": 3,
        "IT": 4,
        "금융": 3,
        "바이오": 2,
        "자동차": 2,
        "화학": 2,
        "건설": 2,
        "유통": 2
      },
      "market_distribution": {
        "KOSPI": 15,
        "KOSDAQ": 5
      }
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**404 Not Found** - 스냅샷이 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No ranking snapshot found",
    "details": "No snapshots available. Ranking generation may not have run yet.",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

**500 Internal Server Error** - 서버 오류

```json
{
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to query ranking",
    "details": "Database connection error",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 2. 랭킹 스냅샷 목록 조회

**GET** `/api/v1/ranking/snapshots`

생성된 랭킹 스냅샷 목록을 조회합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| from | string | N | - | 시작 시간 (RFC3339 형식) |
| to | string | N | - | 종료 시간 (RFC3339 형식) |
| page | integer | N | 1 | 페이지 번호 (1부터 시작) |
| limit | integer | N | 20 | 페이지 크기 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/ranking/snapshots?from=2026-01-14T00:00:00Z&to=2026-01-14T23:59:59Z
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "snapshot_id": "20260114-1030",
      "signal_snapshot_id": "20260114-1030",
      "generated_at": "2026-01-14T10:30:00Z",
      "total_count": 80,
      "selected_count": 20,
      "stats": {
        "avg_total_score": 75.6,
        "avg_alpha_score": 77.2,
        "avg_risk_score": 28.4
      }
    },
    {
      "snapshot_id": "20260114-0930",
      "signal_snapshot_id": "20260114-0930",
      "generated_at": "2026-01-14T09:30:00Z",
      "total_count": 78,
      "selected_count": 20,
      "stats": {
        "avg_total_score": 74.8,
        "avg_alpha_score": 76.5,
        "avg_risk_score": 29.1
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total_pages": 2,
    "total_count": 25,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 3. 특정 스냅샷 조회

**GET** `/api/v1/ranking/snapshots/:snapshotId`

특정 시점의 랭킹 스냅샷을 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| snapshotId | string | Y | 스냅샷 ID (예: 20260114-1030) |

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| selected_only | boolean | N | false | 선정된 종목만 반환 |
| limit | integer | N | 20 | 반환할 종목 수 |

#### Request Example

```http
GET /api/v1/ranking/snapshots/20260114-1030?selected_only=true
```

#### Response (200 OK)

최신 랭킹 스냅샷 조회와 동일한 구조

#### Error Responses

**404 Not Found** - 스냅샷이 존재하지 않음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Ranking snapshot not found",
    "details": "No snapshot found with ID: 20260114-9999",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 4. 특정 종목 랭킹 조회

**GET** `/api/v1/ranking/symbols/:symbol`

특정 종목의 최신 랭킹 정보를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자) |

#### Request Example

```http
GET /api/v1/ranking/symbols/005930
```

#### Response (200 OK)

```json
{
  "data": {
    "snapshot_id": "20260114-1030",
    "symbol": "005930",
    "name": "삼성전자",
    "rank": 1,
    "total_score": 82.3,
    "alpha_score": 85.0,
    "risk_score": 25.0,
    "adjustment": -2.7,
    "selected": true,
    "breakdown": {
      "base_alpha": 85.0,
      "base_risk": 25.0,
      "sector_penalty": 0.0,
      "market_penalty": 0.0,
      "formula": "TotalScore = AlphaScore × 0.7 + (100 - RiskScore) × 0.3"
    },
    "risk_factors": {
      "volatility": 18.5,
      "liquidity": 95.0,
      "volatility_score": 72.0,
      "liquidity_score": 95.0,
      "risk_formula": "RiskScore = VolatilityScore × 0.6 + LiquidityScore × 0.4"
    },
    "sector": "반도체",
    "market": "KOSPI",
    "signal_type": "BUY",
    "signal_strength": 85,
    "generated_at": "2026-01-14T10:30:00Z"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**404 Not Found** - 랭킹 정보가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Ranking not found for symbol",
    "details": "No ranking found for symbol: 005930. Symbol may not have BUY signal.",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 5. 종목별 랭킹 히스토리 조회

**GET** `/api/v1/ranking/symbols/:symbol/history`

특정 종목의 랭킹 변화 히스토리를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자) |

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| from | string | N | 7일 전 | 시작 시간 (RFC3339) |
| to | string | N | 현재 | 종료 시간 (RFC3339) |
| limit | integer | N | 50 | 최대 반환 개수 |

#### Request Example

```http
GET /api/v1/ranking/symbols/005930/history?from=2026-01-07T00:00:00Z
```

#### Response (200 OK)

```json
{
  "data": {
    "symbol": "005930",
    "name": "삼성전자",
    "history": [
      {
        "snapshot_id": "20260114-1030",
        "generated_at": "2026-01-14T10:30:00Z",
        "rank": 1,
        "total_score": 82.3,
        "alpha_score": 85.0,
        "risk_score": 25.0,
        "selected": true
      },
      {
        "snapshot_id": "20260114-0930",
        "generated_at": "2026-01-14T09:30:00Z",
        "rank": 2,
        "total_score": 81.5,
        "alpha_score": 84.0,
        "risk_score": 26.0,
        "selected": true
      },
      {
        "snapshot_id": "20260113-1530",
        "generated_at": "2026-01-13T15:30:00Z",
        "rank": 3,
        "total_score": 79.2,
        "alpha_score": 82.0,
        "risk_score": 28.0,
        "selected": true
      }
    ],
    "summary": {
      "total_snapshots": 3,
      "avg_rank": 2.0,
      "avg_total_score": 81.0,
      "avg_alpha_score": 83.7,
      "avg_risk_score": 26.3,
      "rank_trend": "IMPROVING",
      "selection_rate": 1.0
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 6. 섹터별 랭킹 조회

**GET** `/api/v1/ranking/sectors/:sector`

특정 섹터 내 종목들의 랭킹을 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| sector | string | Y | 섹터명 (예: 반도체, IT, 금융) |

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| selected_only | boolean | N | false | 선정된 종목만 반환 |
| limit | integer | N | 20 | 반환할 종목 수 |

#### Request Example

```http
GET /api/v1/ranking/sectors/반도체?selected_only=true
```

#### Response (200 OK)

```json
{
  "data": {
    "sector": "반도체",
    "snapshot_id": "20260114-1030",
    "generated_at": "2026-01-14T10:30:00Z",
    "stocks": [
      {
        "symbol": "005930",
        "name": "삼성전자",
        "rank": 1,
        "sector_rank": 1,
        "total_score": 82.3,
        "selected": true
      },
      {
        "symbol": "000660",
        "name": "SK하이닉스",
        "rank": 2,
        "sector_rank": 2,
        "total_score": 79.8,
        "selected": true
      }
    ],
    "stats": {
      "total_in_sector": 5,
      "selected_in_sector": 3,
      "avg_total_score": 78.5,
      "sector_weight": 0.15
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

## 데이터 모델

### RankedStock

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| symbol | string | Y | 종목 코드 |
| name | string | Y | 종목명 |
| rank | integer | Y | 전체 순위 (1부터 시작) |
| total_score | float | Y | 종합 점수 (0-100) |
| alpha_score | float | Y | Alpha 점수 (신호 강도) |
| risk_score | float | Y | Risk 점수 (0-100, 낮을수록 좋음) |
| adjustment | float | Y | 다양성 제약 조정값 |
| selected | boolean | Y | 선정 여부 (Top 20) |
| breakdown | ScoreBreakdown | Y | 점수 상세 |
| risk_factors | RiskFactors | Y | 리스크 요인 |
| sector | string | Y | 섹터 |
| market | string | Y | 시장 (KOSPI, KOSDAQ) |
| signal_type | string | Y | 신호 타입 (BUY, SELL, HOLD) |
| signal_strength | integer | Y | 신호 강도 |

### ScoreBreakdown

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| base_alpha | float | Y | 기본 Alpha 점수 |
| base_risk | float | Y | 기본 Risk 점수 |
| sector_penalty | float | Y | 섹터 집중도 페널티 |
| market_penalty | float | Y | 시장 집중도 페널티 |
| formula | string | N | 계산 공식 설명 |

### RiskFactors

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| volatility | float | Y | 변동성 (%) |
| liquidity | float | Y | 유동성 점수 (0-100) |
| volatility_score | float | Y | 변동성 점수 (낮을수록 좋음) |
| liquidity_score | float | Y | 유동성 점수 (높을수록 좋음) |
| risk_formula | string | N | Risk 계산 공식 |

### RankingSnapshot

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| snapshot_id | string | Y | 스냅샷 ID (YYYYMMDD-HHmm) |
| signal_snapshot_id | string | Y | Signals 스냅샷 ID |
| generated_at | string (RFC3339) | Y | 생성 시간 |
| total_count | integer | Y | 총 종목 수 |
| selected_count | integer | Y | 선정된 종목 수 |
| stocks | array[RankedStock] | Y | 랭킹 목록 |
| stats | RankingStats | Y | 통계 |

### RankingStats

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| avg_total_score | float | Y | 평균 종합 점수 |
| avg_alpha_score | float | Y | 평균 Alpha 점수 |
| avg_risk_score | float | Y | 평균 Risk 점수 |
| sector_distribution | object | Y | 섹터별 분포 |
| market_distribution | object | Y | 시장별 분포 |

---

## 비즈니스 룰

### 1. 종합 점수 (Total Score) 계산
```
TotalScore = AlphaScore × 0.7 + (100 - RiskScore) × 0.3

예시:
- AlphaScore: 85 (신호 강도)
- RiskScore: 25 (변동성 18.5%, 유동성 95.0)
- TotalScore = 85 × 0.7 + 75 × 0.3 = 59.5 + 22.5 = 82.0
```

### 2. Risk Score 계산
```
RiskScore = VolatilityScore × 0.6 + (100 - LiquidityScore) × 0.4

VolatilityScore:
- < 15%: 10점
- 15-20%: 20점
- 20-25%: 40점
- 25-30%: 60점
- > 30%: 80점

LiquidityScore:
- 거래대금 기반 (0-100)
- 높을수록 좋음
```

### 3. 다양성 제약
```
섹터 한도:
- 최대 5개 종목/섹터
- 초과 시 페널티 (2점/추가 종목)

시장 한도:
- 최대 15개 종목/시장
- 초과 시 페널티 (1점/추가 종목)
```

### 4. 선정 기준
```
Top 20 선정:
1. TotalScore ≥ 60 (최소 점수)
2. TotalScore 내림차순 정렬
3. 동점 시 AlphaScore 높은 순
4. 상위 20개 선정
```

---

## 사용 예시

### 예시 1: 투자 후보 조회

```bash
# 선정된 Top 20 종목 조회
curl -X GET "http://localhost:8080/api/v1/ranking/latest?selected_only=true"
```

### 예시 2: 특정 종목 분석

```bash
# 삼성전자 랭킹 정보
curl -X GET "http://localhost:8080/api/v1/ranking/symbols/005930"

# 삼성전자 랭킹 변화 (1주일)
curl -X GET "http://localhost:8080/api/v1/ranking/symbols/005930/history?from=2026-01-07T00:00:00Z"
```

### 예시 3: 섹터 분석

```bash
# 반도체 섹터 내 선정 종목
curl -X GET "http://localhost:8080/api/v1/ranking/sectors/반도체?selected_only=true"
```

---

## 성능 고려사항

### 1. 응답 시간
- 최신 스냅샷 조회: < 100ms
- 스냅샷 목록: < 200ms
- 종목별 랭킹: < 50ms
- 히스토리 조회: < 300ms
- 섹터별 조회: < 150ms

### 2. 캐싱 전략
```
Redis TTL:
- /latest: 5분
- /snapshots/:id: 1시간 (불변)
- /symbols/:symbol: 5분
- /sectors/:sector: 5분
```

### 3. DB 인덱스
```sql
-- 최신 스냅샷
CREATE INDEX idx_ranking_snapshots_time ON strategy.ranking_snapshots(generated_at DESC);

-- 종목별 조회
CREATE INDEX idx_rankings_symbol ON strategy.rankings(symbol, snapshot_id);

-- 섹터별 조회
CREATE INDEX idx_rankings_sector ON strategy.rankings(sector, snapshot_id, rank);
```

---

## 에러 코드

| 코드 | HTTP Status | 설명 |
|------|-------------|------|
| INVALID_PARAMETER | 400 | 잘못된 파라미터 |
| NOT_FOUND | 404 | 리소스를 찾을 수 없음 |
| DATABASE_ERROR | 500 | 데이터베이스 오류 |
| INTERNAL_ERROR | 500 | 내부 서버 오류 |

---

## 참고 문서

- [Ranking 모듈 설계](../modules/ranking.md)
- [API 공통 스펙](./common.md)
- [Signals API](./signals.md)
- [Portfolio API](./portfolio.md)

---

**Version**: 1.0.0
**Author**: Aegis Team
**Status**: ✅ 설계 완료
