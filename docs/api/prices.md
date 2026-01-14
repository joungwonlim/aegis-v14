# Prices API

> 가격 조회 API (PriceSync 모듈)

**Last Updated**: 2026-01-14

---

## 개요

실시간 최적 가격(Best Price) 조회 API. market.prices_best 테이블의 데이터를 제공합니다.

**Base URL**: `/api/prices`

**인증**: (추후 추가)

---

## 엔드포인트

### 1. Best Price 조회 (단일 종목)

**GET** `/api/prices/:symbol`

특정 종목의 최적 가격을 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자, 예: 005930) |

#### Request Example

```http
GET /api/prices/005930
```

#### Response (200 OK)

```json
{
  "data": {
    "symbol": "005930",
    "best_price": 71000,
    "best_source": "KIS_WS",
    "best_ts": "2026-01-14T13:05:20Z",
    "change_price": 500,
    "change_rate": 0.71,
    "volume": 10000000,
    "bid_price": 70900,
    "ask_price": 71000,
    "is_stale": false,
    "updated_ts": "2026-01-14T13:05:25Z"
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T13:05:25Z"
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
    "timestamp": "2026-01-14T13:05:25Z"
  }
}
```

**404 Not Found** - 가격 데이터 없음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Price not found",
    "details": "No price data for symbol: 999999",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T13:05:25Z"
  }
}
```

---

### 2. Best Price 조회 (다중 종목)

**POST** `/api/prices/batch`

여러 종목의 최적 가격을 한 번에 조회합니다.

#### Request Body

```json
{
  "symbols": ["005930", "000660", "035420"]
}
```

#### Request Example

```http
POST /api/prices/batch
Content-Type: application/json

{
  "symbols": ["005930", "000660", "035420"]
}
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "symbol": "005930",
      "best_price": 71000,
      "best_source": "KIS_WS",
      "best_ts": "2026-01-14T13:05:20Z",
      "change_price": 500,
      "change_rate": 0.71,
      "volume": 10000000,
      "is_stale": false,
      "updated_ts": "2026-01-14T13:05:25Z"
    },
    {
      "symbol": "000660",
      "best_price": 130000,
      "best_source": "KIS_REST",
      "best_ts": "2026-01-14T13:05:15Z",
      "change_price": -2000,
      "change_rate": -1.52,
      "volume": 5000000,
      "is_stale": false,
      "updated_ts": "2026-01-14T13:05:20Z"
    }
  ],
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T13:05:25Z"
  }
}
```

#### Error Responses

**400 Bad Request** - 잘못된 요청

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid request",
    "details": "symbols array is required",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T13:05:25Z"
  }
}
```

---

### 3. Freshness 조회

**GET** `/api/prices/:symbol/freshness`

특정 종목의 소스별 신선도 정보를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자) |

#### Request Example

```http
GET /api/prices/005930/freshness
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "symbol": "005930",
      "source": "KIS_WS",
      "last_ts": "2026-01-14T13:05:20Z",
      "last_price": 71000,
      "is_stale": false,
      "staleness_ms": 5000,
      "quality_score": 95,
      "updated_ts": "2026-01-14T13:05:25Z"
    },
    {
      "symbol": "005930",
      "source": "KIS_REST",
      "last_ts": "2026-01-14T13:05:10Z",
      "last_price": 71000,
      "is_stale": false,
      "staleness_ms": 15000,
      "quality_score": 50,
      "updated_ts": "2026-01-14T13:05:25Z"
    },
    {
      "symbol": "005930",
      "source": "NAVER",
      "last_ts": "2026-01-14T13:04:55Z",
      "last_price": 71100,
      "is_stale": false,
      "staleness_ms": 30000,
      "quality_score": 20,
      "updated_ts": "2026-01-14T13:05:25Z"
    }
  ],
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T13:05:25Z"
  }
}
```

---

## 데이터 모델

### BestPrice

| 필드 | 타입 | 설명 |
|------|------|------|
| symbol | string | 종목 코드 |
| best_price | integer | 최적 가격 (원) |
| best_source | string | 최적 소스 (KIS_WS, KIS_REST, NAVER) |
| best_ts | string | 최적 가격 수신 시각 (ISO 8601) |
| change_price | integer \| null | 전일대비 |
| change_rate | float \| null | 등락률 (%) |
| volume | integer \| null | 거래량 |
| bid_price | integer \| null | 매수호가 |
| ask_price | integer \| null | 매도호가 |
| is_stale | boolean | 모든 소스 stale 여부 |
| updated_ts | string | 업데이트 일시 (ISO 8601) |

### Freshness

| 필드 | 타입 | 설명 |
|------|------|------|
| symbol | string | 종목 코드 |
| source | string | 소스 (KIS_WS, KIS_REST, NAVER) |
| last_ts | string | 마지막 수신 시각 (ISO 8601) |
| last_price | integer | 마지막 가격 (원) |
| is_stale | boolean | Stale 여부 |
| staleness_ms | integer | 현재 - last_ts (밀리초) |
| quality_score | integer | 품질 점수 (0~100) |
| updated_ts | string | 업데이트 일시 (ISO 8601) |

---

## 비즈니스 규칙

### Best Source 선택

1. Quality Score 기반 선택
2. Score = Base Score - Staleness Penalty
3. Base Score = Priority × 30 (WS=90, REST=60, NAVER=30)
4. Stale 시 Score = 0

### Freshness 임계값

**장중**:
- WS: 2,000ms
- REST: 10,000ms
- NAVER: 30,000ms

**장후**:
- WS: 10,000ms
- REST: 30,000ms
- NAVER: 60,000ms

### is_stale 처리

- 모든 소스가 stale 시 `is_stale = true`
- 하나라도 fresh 시 `is_stale = false`

---

## 구현 위치

### Backend (Go)

```
backend/internal/
├── domain/price/
│   └── model.go              # Price 도메인 모델
├── infra/database/postgres/
│   └── price_repository.go   # PostgreSQL CRUD
├── service/pricesync/
│   └── service.go            # PriceSync 서비스
└── api/handlers/
    └── price.go              # HTTP 핸들러
```

### 의존성

- **Database**: market.prices_best, market.freshness (READ ONLY)
- **Service**: PriceSync Service
- **Response**: `internal/api/response` 패키지

---

## 권한

**현재 단계**: 인증/인가 없음 (추후 추가 예정)

**DB 권한**: aegis_v14 role (SELECT only on market.prices_*)

---

## 테스트 시나리오

### 정상 시나리오
1. ✅ 단일 종목 Best Price 조회
2. ✅ 다중 종목 Batch 조회
3. ✅ Freshness 정보 조회
4. ✅ Stale 종목 처리

### 에러 시나리오
1. ❌ 잘못된 종목 코드 형식
2. ❌ 존재하지 않는 종목
3. ❌ 빈 symbols 배열

---

## 성능 고려사항

### 캐싱
- prices_best는 UPSERT로 1행 유지 (빠름)
- 인덱스: PK(symbol)만으로 충분
- Redis 캐싱 고려 (향후)

### Batch 조회
- 최대 100개 제한 권장
- IN 절 사용 (PostgreSQL 최적화)

---

## 향후 개선

1. **WebSocket API**: 실시간 가격 스트리밍
2. **Redis 캐싱**: Best Price 캐싱
3. **Rate Limiting**: API 호출 제한
4. **GraphQL**: 유연한 쿼리

---

## 참고 문서

- [API Common Spec](./common.md)
- [Stocks API](./stocks.md)
- [PriceSync Module](../modules/price-sync.md)
