# Portfolio API

> 포트폴리오 구성 및 조회 API

**Last Updated**: 2026-01-14

---

## 개요

포트폴리오 구성 및 조회 API. portfolio.portfolios 테이블의 데이터를 제공합니다.

**Base URL**: `/api/v1/portfolio`

**인증**: (추후 추가)

**특징**:
- Ranking 결과 기반 포트폴리오 구성
- Equal-Weight / Score-Weighted 할당 지원
- 비중 제약 (단일 종목 15%, 섹터 40%)
- 주간 리밸런싱 (매주 월요일 09:10 KST)
- 히스토리 추적 (DRAFT, ACTIVE, ARCHIVED)

---

## 엔드포인트

### 1. 포트폴리오 생성

**POST** `/api/v1/portfolio/generate`

Ranking 결과로부터 새 포트폴리오를 생성합니다.

#### Request Body

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| snapshot_id | string (UUID) | Y | Ranking 스냅샷 ID |
| notes | string | N | 메모 (최대 1000자) |

#### Request Example

```http
POST /api/v1/portfolio/generate
Content-Type: application/json

{
  "snapshot_id": "550e8400-e29b-41d4-a716-446655440001",
  "notes": "2026년 1월 3주차 리밸런싱"
}
```

#### Response (201 Created)

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
        "target_weight": 8.33,
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
        "target_weight": 8.33,
        "total_score": 79.8,
        "alpha_score": 82.0,
        "risk_score": 28.0,
        "capped": false,
        "capped_reason": "",
        "sector": "반도체",
        "market": "KOSPI"
      },
      {
        "symbol": "035420",
        "name": "NAVER",
        "target_weight": 8.33,
        "total_score": 78.5,
        "alpha_score": 80.0,
        "risk_score": 30.0,
        "capped": false,
        "capped_reason": "",
        "sector": "IT",
        "market": "KOSPI"
      }
    ],
    "total_weight": 100.0,
    "stats": {
      "total_holdings": 12,
      "avg_weight": 8.33,
      "max_weight": 8.33,
      "min_weight": 8.33,
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
    },
    "status": "DRAFT",
    "notes": "2026년 1월 3주차 리밸런싱"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:05Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 파라미터

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid snapshot_id",
    "details": "snapshot_id must be valid UUID",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

**404 Not Found** - Ranking 스냅샷이 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Ranking snapshot not found",
    "details": "No ranking snapshot found with ID: 550e8400-e29b-41d4-a716-446655440001",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

**422 Unprocessable Entity** - 포트폴리오 생성 실패

```json
{
  "error": {
    "code": "INSUFFICIENT_HOLDINGS",
    "message": "Cannot create portfolio",
    "details": "Insufficient holdings: got 5, need at least 10",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

---

### 2. 최신 활성 포트폴리오 조회

**GET** `/api/v1/portfolio/latest`

현재 활성화된 포트폴리오를 조회합니다.

#### Request Example

```http
GET /api/v1/portfolio/latest
```

#### Response (200 OK)

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
        "target_weight": 8.33,
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
    "status": "ACTIVE",
    "notes": "2026년 1월 3주차 리밸런싱"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

#### Error Responses

**404 Not Found** - 활성 포트폴리오가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No active portfolio found",
    "details": "No active portfolio available. Please generate a new portfolio.",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

---

### 3. 특정 포트폴리오 조회

**GET** `/api/v1/portfolio/:id`

특정 ID의 포트폴리오를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| id | string (UUID) | Y | 포트폴리오 ID |

#### Request Example

```http
GET /api/v1/portfolio/660e8400-e29b-41d4-a716-446655440002
```

#### Response (200 OK)

최신 활성 포트폴리오 조회와 동일한 구조

#### Error Responses

**400 Bad Request** - 잘못된 UUID 형식

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid portfolio ID",
    "details": "id must be valid UUID",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

**404 Not Found** - 포트폴리오가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Portfolio not found",
    "details": "No portfolio found with ID: 660e8400-e29b-41d4-a716-446655440002",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

---

### 4. 포트폴리오 목록 조회

**GET** `/api/v1/portfolio`

포트폴리오 목록을 조회합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| status | string | N | - | 상태 필터 (DRAFT, ACTIVE, ARCHIVED) |
| from | string | N | - | 시작일 (RFC3339) |
| to | string | N | - | 종료일 (RFC3339) |
| page | integer | N | 1 | 페이지 번호 (1부터 시작) |
| limit | integer | N | 20 | 페이지 크기 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/portfolio?status=ACTIVE&from=2026-01-01T00:00:00Z&page=1&limit=10
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "snapshot_id": "550e8400-e29b-41d4-a716-446655440001",
      "created_at": "2026-01-14T15:00:00Z",
      "total_weight": 100.0,
      "stats": {
        "total_holdings": 12,
        "avg_weight": 8.33,
        "avg_total_score": 75.6
      },
      "status": "ACTIVE",
      "notes": "2026년 1월 3주차 리밸런싱"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "snapshot_id": "550e8400-e29b-41d4-a716-446655440002",
      "created_at": "2026-01-07T15:00:00Z",
      "total_weight": 100.0,
      "stats": {
        "total_holdings": 13,
        "avg_weight": 7.69,
        "avg_total_score": 74.2
      },
      "status": "ARCHIVED",
      "notes": "2026년 1월 2주차 리밸런싱"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total_pages": 5,
    "total_count": 45,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 상태 값

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid status",
    "details": "status must be one of: DRAFT, ACTIVE, ARCHIVED",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

---

### 5. 포트폴리오 활성화

**POST** `/api/v1/portfolio/:id/activate`

특정 포트폴리오를 활성화합니다. 기존 ACTIVE 포트폴리오는 자동으로 ARCHIVED 상태로 변경됩니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| id | string (UUID) | Y | 포트폴리오 ID |

#### Request Example

```http
POST /api/v1/portfolio/660e8400-e29b-41d4-a716-446655440002/activate
```

#### Response (200 OK)

```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "status": "ACTIVE",
    "activated_at": "2026-01-14T15:10:00Z",
    "previous_active_id": "770e8400-e29b-41d4-a716-446655440003"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:10:00Z"
  }
}
```

#### Error Responses

**404 Not Found** - 포트폴리오가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Portfolio not found",
    "details": "No portfolio found with ID: 660e8400-e29b-41d4-a716-446655440002",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:10:00Z"
  }
}
```

**409 Conflict** - 이미 활성화됨

```json
{
  "error": {
    "code": "ALREADY_ACTIVE",
    "message": "Portfolio already active",
    "details": "Portfolio 660e8400-e29b-41d4-a716-446655440002 is already in ACTIVE status",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:10:00Z"
  }
}
```

---

### 6. 포트폴리오 비교

**GET** `/api/v1/portfolio/compare`

두 포트폴리오 간의 차이를 비교합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| from_id | string (UUID) | Y | 비교 기준 포트폴리오 ID |
| to_id | string (UUID) | Y | 비교 대상 포트폴리오 ID |

#### Request Example

```http
GET /api/v1/portfolio/compare?from_id=660e8400-e29b-41d4-a716-446655440002&to_id=770e8400-e29b-41d4-a716-446655440003
```

#### Response (200 OK)

```json
{
  "data": {
    "from": {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "created_at": "2026-01-14T15:00:00Z",
      "total_holdings": 12
    },
    "to": {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "created_at": "2026-01-07T15:00:00Z",
      "total_holdings": 13
    },
    "changes": {
      "added": [
        {
          "symbol": "035720",
          "name": "카카오",
          "target_weight": 7.69,
          "sector": "IT"
        }
      ],
      "removed": [
        {
          "symbol": "051910",
          "name": "LG화학",
          "target_weight": 8.33,
          "sector": "화학"
        }
      ],
      "weight_changed": [
        {
          "symbol": "005930",
          "name": "삼성전자",
          "from_weight": 8.33,
          "to_weight": 7.69,
          "diff": -0.64
        }
      ],
      "unchanged": [
        {
          "symbol": "000660",
          "name": "SK하이닉스",
          "target_weight": 8.33
        }
      ]
    },
    "summary": {
      "total_changes": 3,
      "added_count": 1,
      "removed_count": 1,
      "weight_changed_count": 1,
      "unchanged_count": 10,
      "turnover": 0.15
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 필수 파라미터 누락

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Missing required parameters",
    "details": "Both from_id and to_id are required",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

**404 Not Found** - 포트폴리오가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Portfolio not found",
    "details": "One or both portfolios not found",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T15:00:00Z"
  }
}
```

---

## 데이터 모델

### Portfolio

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| id | string (UUID) | Y | 포트폴리오 ID |
| snapshot_id | string (UUID) | Y | Ranking 스냅샷 ID |
| created_at | string (RFC3339) | Y | 생성 시간 |
| holdings | array[Holding] | Y | 보유 종목 목록 |
| total_weight | float | Y | 총 비중 (100.0) |
| stats | PortfolioStats | Y | 포트폴리오 통계 |
| status | string | Y | 상태 (DRAFT, ACTIVE, ARCHIVED) |
| notes | string | N | 메모 |

### Holding

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| symbol | string | Y | 종목 코드 |
| name | string | Y | 종목명 |
| target_weight | float | Y | 목표 비중 (%) |
| total_score | float | Y | Ranking 종합 점수 |
| alpha_score | float | Y | Alpha 점수 |
| risk_score | float | Y | Risk 점수 |
| capped | boolean | Y | 한도 적용 여부 |
| capped_reason | string | N | 한도 적용 이유 |
| sector | string | Y | 섹터 |
| market | string | Y | 시장 (KOSPI, KOSDAQ) |

### PortfolioStats

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| total_holdings | integer | Y | 총 보유 종목 수 |
| avg_weight | float | Y | 평균 비중 (%) |
| max_weight | float | Y | 최대 비중 (%) |
| min_weight | float | Y | 최소 비중 (%) |
| sector_count | object | Y | 섹터별 종목 수 |
| market_count | object | Y | 시장별 종목 수 |
| avg_total_score | float | Y | 평균 종합 점수 |
| avg_risk_score | float | Y | 평균 리스크 점수 |

---

## 비즈니스 룰

### 1. 포트폴리오 크기
```
목표: 10-15 종목

최소: 10종목 (분산 확보)
최대: 15종목 (관리 가능성)
```

### 2. 비중 할당 (Phase 1: Equal-Weight)
```
Equal-Weight:
- 모든 종목 균등 배분
- 12종목 → 각 8.33%
- 단순하고 효과적
```

### 3. 비중 제약
```
단일 종목 한도: 15%
- 초과 시 15%로 제한
- 여유 비중은 다른 종목에 재분배

섹터 한도: 40%
- 섹터 내 종목 비중 합계
- 초과 시 비례 축소
```

### 4. 리밸런싱 주기
```
주기: 1주 (매주 월요일 09:10 KST)

프로세스:
1. 최신 Ranking 조회
2. 새 포트폴리오 생성 (DRAFT)
3. 검증 및 승인
4. 활성화 (ACTIVE)
5. 기존 포트폴리오 아카이브 (ARCHIVED)
```

### 5. 상태 전환
```
DRAFT → ACTIVE: 활성화 (POST /activate)
ACTIVE → ARCHIVED: 새 포트폴리오 활성화 시 자동

상태별 의미:
- DRAFT: 생성 완료, 검토 대기
- ACTIVE: 현재 운용 중
- ARCHIVED: 과거 포트폴리오 (히스토리)
```

---

## 사용 예시

### 예시 1: 포트폴리오 생성 및 활성화

```bash
# 1. 최신 Ranking으로 포트폴리오 생성
PORTFOLIO_ID=$(curl -X POST "http://localhost:8080/api/v1/portfolio/generate" \
  -H "Content-Type: application/json" \
  -d '{"snapshot_id":"550e8400-e29b-41d4-a716-446655440001"}' \
  | jq -r '.data.id')

# 2. 생성된 포트폴리오 검토
curl -X GET "http://localhost:8080/api/v1/portfolio/${PORTFOLIO_ID}"

# 3. 활성화
curl -X POST "http://localhost:8080/api/v1/portfolio/${PORTFOLIO_ID}/activate"
```

### 예시 2: 현재 포트폴리오 조회

```bash
# 활성 포트폴리오 조회
curl -X GET "http://localhost:8080/api/v1/portfolio/latest"
```

### 예시 3: 리밸런싱 비교

```bash
# 현재 vs 이전 포트폴리오 비교
curl -X GET "http://localhost:8080/api/v1/portfolio/compare?from_id=${OLD_ID}&to_id=${NEW_ID}"
```

---

## 성능 고려사항

### 1. 응답 시간
- 생성 (POST /generate): < 200ms
- 최신 조회 (GET /latest): < 50ms
- 목록 조회 (GET /): < 150ms
- 활성화 (POST /activate): < 100ms
- 비교 (GET /compare): < 200ms

### 2. 캐싱 전략
```
Redis TTL:
- /latest: 5분 (리밸런싱 주기 고려)
- /:id: 1시간 (불변 데이터)
- /compare: 10분 (계산 비용)
```

### 3. DB 인덱스
```sql
-- 최신 활성 포트폴리오
CREATE INDEX idx_portfolios_active ON portfolio.portfolios(status, created_at DESC)
    WHERE status = 'ACTIVE';

-- 목록 조회
CREATE INDEX idx_portfolios_created ON portfolio.portfolios(created_at DESC);

-- 상태별 조회
CREATE INDEX idx_portfolios_status ON portfolio.portfolios(status, created_at DESC);
```

---

## 에러 코드

| 코드 | HTTP Status | 설명 |
|------|-------------|------|
| INVALID_PARAMETER | 400 | 잘못된 파라미터 |
| NOT_FOUND | 404 | 리소스를 찾을 수 없음 |
| ALREADY_ACTIVE | 409 | 포트폴리오가 이미 활성화됨 |
| INSUFFICIENT_HOLDINGS | 422 | 보유 종목 수 부족 |
| DATABASE_ERROR | 500 | 데이터베이스 오류 |
| INTERNAL_ERROR | 500 | 내부 서버 오류 |

---

## 참고 문서

- [Portfolio 모듈 설계](../modules/portfolio.md)
- [API 공통 스펙](./common.md)
- [Ranking API](./ranking.md)

---

**Version**: 1.0.0
**Author**: Aegis Team
**Status**: ✅ 설계 완료
