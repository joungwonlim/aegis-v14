# Signals API

> 매매 신호 조회 API

**Last Updated**: 2026-01-14

---

## 개요

팩터 기반 매매 신호 조회 API. strategy.signals 테이블의 데이터를 제공합니다.

**Base URL**: `/api/v1/signals`

**인증**: (추후 추가)

**특징**:
- 1시간마다 자동 생성되는 스냅샷 기반
- BUY/SELL/HOLD 신호 제공
- 4가지 팩터 (Momentum, Quality, Value, Technical) 분석 결과 포함
- 시계열 조회 지원

---

## 엔드포인트

### 1. 최신 신호 스냅샷 조회

**GET** `/api/v1/signals/latest`

최신 생성된 신호 스냅샷을 조회합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| signal_type | string | N | - | 신호 타입 필터 (BUY, SELL, HOLD) |
| min_strength | integer | N | - | 최소 신호 강도 (0-100) |
| min_conviction | integer | N | - | 최소 확신도 (0-100) |
| limit | integer | N | 50 | 반환할 신호 수 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/signals/latest?signal_type=BUY&min_strength=70&limit=20
```

#### Response (200 OK)

```json
{
  "data": {
    "snapshot_id": "20260114-1030",
    "universe_id": "20260114-1000",
    "generated_at": "2026-01-14T10:30:00Z",
    "total_count": 30,
    "signals": [
      {
        "signal_id": "550e8400-e29b-41d4-a716-446655440001",
        "symbol": "005930",
        "name": "삼성전자",
        "signal_type": "BUY",
        "strength": 78,
        "conviction": 75,
        "rank": 1,
        "factors": {
          "momentum": {
            "score": 85,
            "weight": 0.35,
            "weighted_score": 29.75,
            "triggered": true,
            "indicators": ["5D_RETURN_HIGH", "20D_RETURN_HIGH", "RS_STRONG"]
          },
          "quality": {
            "score": 70,
            "weight": 0.25,
            "weighted_score": 17.5,
            "triggered": true,
            "indicators": ["ROE_HIGH", "DEBT_LOW"]
          },
          "value": {
            "score": 60,
            "weight": 0.20,
            "weighted_score": 12.0,
            "triggered": false,
            "indicators": ["PER_FAIR"]
          },
          "technical": {
            "score": 80,
            "weight": 0.20,
            "weighted_score": 16.0,
            "triggered": true,
            "indicators": ["MACD_BULLISH", "BB_LOWER", "VOLUME_HIGH"]
          }
        },
        "reasons": [
          "모멘텀 강함 (85점)",
          "품질 우수 (70점)",
          "기술적 매수 신호 (80점)"
        ],
        "sector": "반도체",
        "market": "KOSPI",
        "generated_at": "2026-01-14T10:30:00Z"
      },
      {
        "signal_id": "550e8400-e29b-41d4-a716-446655440002",
        "symbol": "000660",
        "name": "SK하이닉스",
        "signal_type": "BUY",
        "strength": 75,
        "conviction": 75,
        "rank": 2,
        "factors": {
          "momentum": {
            "score": 80,
            "weight": 0.35,
            "weighted_score": 28.0,
            "triggered": true,
            "indicators": ["5D_RETURN_HIGH", "RS_STRONG"]
          },
          "quality": {
            "score": 75,
            "weight": 0.25,
            "weighted_score": 18.75,
            "triggered": true,
            "indicators": ["ROE_HIGH", "PROFIT_MARGIN_HIGH"]
          },
          "value": {
            "score": 55,
            "weight": 0.20,
            "weighted_score": 11.0,
            "triggered": false,
            "indicators": []
          },
          "technical": {
            "score": 85,
            "weight": 0.20,
            "weighted_score": 17.0,
            "triggered": true,
            "indicators": ["MACD_BULLISH", "RSI_OVERSOLD"]
          }
        },
        "reasons": [
          "모멘텀 강함 (80점)",
          "품질 우수 (75점)",
          "기술적 매수 신호 (85점)"
        ],
        "sector": "반도체",
        "market": "KOSPI",
        "generated_at": "2026-01-14T10:30:00Z"
      }
    ],
    "stats": {
      "buy_count": 25,
      "sell_count": 5,
      "hold_count": 120,
      "avg_strength": 72.5,
      "avg_conviction": 68.3
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 파라미터

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid signal_type",
    "details": "signal_type must be one of: BUY, SELL, HOLD",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

**404 Not Found** - 스냅샷이 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No signal snapshot found",
    "details": "No snapshots available. Signal generation may not have run yet.",
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
    "message": "Failed to query signals",
    "details": "Database connection error",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 2. 신호 스냅샷 목록 조회

**GET** `/api/v1/signals/snapshots`

생성된 신호 스냅샷 목록을 조회합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| from | string | N | - | 시작 시간 (RFC3339 형식, 예: 2026-01-14T00:00:00Z) |
| to | string | N | - | 종료 시간 (RFC3339 형식) |
| page | integer | N | 1 | 페이지 번호 (1부터 시작) |
| limit | integer | N | 20 | 페이지 크기 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/signals/snapshots?from=2026-01-14T00:00:00Z&to=2026-01-14T23:59:59Z&page=1&limit=10
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "snapshot_id": "20260114-1030",
      "universe_id": "20260114-1000",
      "generated_at": "2026-01-14T10:30:00Z",
      "total_count": 30,
      "stats": {
        "buy_count": 25,
        "sell_count": 5,
        "hold_count": 120,
        "avg_strength": 72.5,
        "avg_conviction": 68.3
      }
    },
    {
      "snapshot_id": "20260114-0930",
      "universe_id": "20260114-0900",
      "generated_at": "2026-01-14T09:30:00Z",
      "total_count": 28,
      "stats": {
        "buy_count": 22,
        "sell_count": 6,
        "hold_count": 122,
        "avg_strength": 70.2,
        "avg_conviction": 65.8
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total_pages": 2,
    "total_count": 15,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 날짜 형식

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid date format",
    "details": "from must be in RFC3339 format (e.g., 2026-01-14T00:00:00Z)",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 3. 특정 스냅샷 조회

**GET** `/api/v1/signals/snapshots/:snapshotId`

특정 시점의 신호 스냅샷을 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| snapshotId | string | Y | 스냅샷 ID (예: 20260114-1030) |

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| signal_type | string | N | - | 신호 타입 필터 (BUY, SELL, HOLD) |
| min_strength | integer | N | - | 최소 신호 강도 (0-100) |
| limit | integer | N | 50 | 반환할 신호 수 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/signals/snapshots/20260114-1030?signal_type=BUY&min_strength=70
```

#### Response (200 OK)

최신 신호 스냅샷 조회와 동일한 구조

#### Error Responses

**404 Not Found** - 스냅샷이 존재하지 않음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Signal snapshot not found",
    "details": "No snapshot found with ID: 20260114-9999",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 4. 특정 종목 신호 조회

**GET** `/api/v1/signals/symbols/:symbol`

특정 종목의 최신 신호를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자, 예: 005930) |

#### Request Example

```http
GET /api/v1/signals/symbols/005930
```

#### Response (200 OK)

```json
{
  "data": {
    "signal_id": "550e8400-e29b-41d4-a716-446655440001",
    "snapshot_id": "20260114-1030",
    "symbol": "005930",
    "name": "삼성전자",
    "signal_type": "BUY",
    "strength": 78,
    "conviction": 75,
    "rank": 1,
    "factors": {
      "momentum": {
        "score": 85,
        "weight": 0.35,
        "weighted_score": 29.75,
        "triggered": true,
        "indicators": ["5D_RETURN_HIGH", "20D_RETURN_HIGH", "RS_STRONG"]
      },
      "quality": {
        "score": 70,
        "weight": 0.25,
        "weighted_score": 17.5,
        "triggered": true,
        "indicators": ["ROE_HIGH", "DEBT_LOW"]
      },
      "value": {
        "score": 60,
        "weight": 0.20,
        "weighted_score": 12.0,
        "triggered": false,
        "indicators": ["PER_FAIR"]
      },
      "technical": {
        "score": 80,
        "weight": 0.20,
        "weighted_score": 16.0,
        "triggered": true,
        "indicators": ["MACD_BULLISH", "BB_LOWER", "VOLUME_HIGH"]
      }
    },
    "reasons": [
      "모멘텀 강함 (85점)",
      "품질 우수 (70점)",
      "기술적 매수 신호 (80점)"
    ],
    "sector": "반도체",
    "market": "KOSPI",
    "generated_at": "2026-01-14T10:30:00Z"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 종목 코드 형식

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid symbol format",
    "details": "Symbol must be 6-digit number",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

**404 Not Found** - 신호가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Signal not found for symbol",
    "details": "No signal found for symbol: 005930. Symbol may not be in universe.",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 5. 종목별 신호 히스토리 조회

**GET** `/api/v1/signals/symbols/:symbol/history`

특정 종목의 신호 변화 히스토리를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자) |

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| from | string | N | 7일 전 | 시작 시간 (RFC3339) |
| to | string | N | 현재 | 종료 시간 (RFC3339) |
| limit | integer | N | 50 | 최대 반환 개수 (최소: 1, 최대: 100) |

#### Request Example

```http
GET /api/v1/signals/symbols/005930/history?from=2026-01-07T00:00:00Z&to=2026-01-14T23:59:59Z
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
        "signal_type": "BUY",
        "strength": 78,
        "conviction": 75,
        "rank": 1
      },
      {
        "snapshot_id": "20260114-0930",
        "generated_at": "2026-01-14T09:30:00Z",
        "signal_type": "BUY",
        "strength": 76,
        "conviction": 75,
        "rank": 2
      },
      {
        "snapshot_id": "20260113-1530",
        "generated_at": "2026-01-13T15:30:00Z",
        "signal_type": "BUY",
        "strength": 72,
        "conviction": 50,
        "rank": 3
      }
    ],
    "summary": {
      "total_snapshots": 3,
      "signal_changes": [
        {
          "from": "HOLD",
          "to": "BUY",
          "changed_at": "2026-01-13T15:30:00Z"
        }
      ],
      "avg_strength": 75.3,
      "avg_conviction": 66.7
    }
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### Error Responses

**404 Not Found** - 히스토리가 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No signal history found",
    "details": "No signals found for symbol: 005930 in the specified time range",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

## 데이터 모델

### Signal

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| signal_id | string (UUID) | Y | 신호 고유 ID |
| snapshot_id | string | Y | 스냅샷 ID |
| symbol | string | Y | 종목 코드 |
| name | string | Y | 종목명 |
| signal_type | string | Y | 신호 타입 (BUY, SELL, HOLD) |
| strength | integer | Y | 신호 강도 (0-100) |
| conviction | integer | Y | 확신도 (0-100) |
| rank | integer | Y | 순위 (1부터 시작) |
| factors | object | Y | 팩터별 점수 |
| factors.momentum | FactorScore | Y | 모멘텀 팩터 |
| factors.quality | FactorScore | Y | 품질 팩터 |
| factors.value | FactorScore | Y | 가치 팩터 |
| factors.technical | FactorScore | Y | 기술적 팩터 |
| reasons | array[string] | Y | 신호 이유 목록 |
| sector | string | Y | 섹터 |
| market | string | Y | 시장 (KOSPI, KOSDAQ) |
| generated_at | string (RFC3339) | Y | 생성 시간 |

### FactorScore

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| score | integer | Y | 팩터 점수 (0-100) |
| weight | float | Y | 가중치 (0.0-1.0) |
| weighted_score | float | Y | 가중 점수 (score × weight) |
| triggered | boolean | Y | 트리거 여부 |
| indicators | array[string] | Y | 트리거된 지표 목록 |

### Snapshot

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| snapshot_id | string | Y | 스냅샷 ID (YYYYMMDD-HHmm) |
| universe_id | string | Y | Universe 스냅샷 ID |
| generated_at | string (RFC3339) | Y | 생성 시간 |
| total_count | integer | Y | 총 신호 수 |
| signals | array[Signal] | Y | 신호 목록 |
| stats | SnapshotStats | Y | 통계 |

### SnapshotStats

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| buy_count | integer | Y | BUY 신호 수 |
| sell_count | integer | Y | SELL 신호 수 |
| hold_count | integer | Y | HOLD 신호 수 |
| avg_strength | float | Y | 평균 신호 강도 |
| avg_conviction | float | Y | 평균 확신도 |

---

## 비즈니스 룰

### 1. 신호 강도 (Strength)
- 범위: 0-100
- 계산: 4가지 팩터의 가중 평균
- 해석:
  - 80-100: 강한 신호
  - 60-79: 보통 신호
  - 0-59: 약한 신호

### 2. 확신도 (Conviction)
- 범위: 0-100
- 계산: (트리거된 팩터 수 / 전체 팩터 수) × 100
- 해석:
  - 100: 4개 팩터 모두 트리거
  - 75: 3개 팩터 트리거
  - 50: 2개 팩터 트리거
  - 25: 1개 팩터 트리거

### 3. 신호 타입
- **BUY**: strength ≥ 60 AND conviction ≥ 50
- **SELL**: strength < 40 OR (strength < 60 AND conviction < 25)
- **HOLD**: 그 외

### 4. 순위 (Rank)
- BUY 신호 내에서 strength 기준 내림차순
- 동점 시 conviction 기준 내림차순
- 동점 시 symbol 기준 오름차순

---

## 사용 예시

### 예시 1: 매수 후보 조회

```bash
# 강한 매수 신호 (strength ≥ 70, conviction ≥ 75) 조회
curl -X GET "http://localhost:8080/api/v1/signals/latest?signal_type=BUY&min_strength=70&min_conviction=75&limit=10"
```

### 예시 2: 특정 종목 모니터링

```bash
# 삼성전자 신호 조회
curl -X GET "http://localhost:8080/api/v1/signals/symbols/005930"

# 삼성전자 신호 히스토리 (1주일)
curl -X GET "http://localhost:8080/api/v1/signals/symbols/005930/history?from=2026-01-07T00:00:00Z"
```

### 예시 3: 시계열 분석

```bash
# 오늘 생성된 모든 스냅샷 조회
curl -X GET "http://localhost:8080/api/v1/signals/snapshots?from=2026-01-14T00:00:00Z&to=2026-01-14T23:59:59Z"

# 특정 시점 스냅샷 조회
curl -X GET "http://localhost:8080/api/v1/signals/snapshots/20260114-1030"
```

---

## 성능 고려사항

### 1. 응답 시간
- 최신 스냅샷 조회: < 100ms (인덱스 활용)
- 스냅샷 목록: < 200ms (페이징)
- 종목별 신호: < 50ms (symbol 인덱스)
- 히스토리 조회: < 300ms (시계열 인덱스)

### 2. 캐싱 전략
```
Redis TTL:
- /latest: 5분 (신호 생성 주기보다 짧음)
- /snapshots/:id: 1시간 (불변 데이터)
- /symbols/:symbol: 5분
```

### 3. DB 인덱스
```sql
-- 최신 스냅샷 조회
CREATE INDEX idx_snapshots_generated_at ON strategy.signal_snapshots(generated_at DESC);

-- 종목별 신호 조회
CREATE INDEX idx_signals_symbol ON strategy.signals(symbol, snapshot_id);

-- 시계열 조회
CREATE INDEX idx_signals_time ON strategy.signals(generated_at DESC, symbol);
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

- [Signals 모듈 설계](../modules/signals.md)
- [API 공통 스펙](./common.md)
- [Universe API](../api/universe.md)
- [Ranking API](./ranking.md)

---

**Version**: 1.0.0
**Author**: Aegis Team
**Status**: ✅ 설계 완료
