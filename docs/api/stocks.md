# Stocks API

> 종목 조회 및 관리 API

**Last Updated**: 2026-01-14

---

## 개요

종목 기본 정보 조회 API. market.stocks 테이블의 데이터를 제공합니다.

**Base URL**: `/api/stocks`

**인증**: (추후 추가)

---

## 엔드포인트

### 1. 종목 목록 조회

**GET** `/api/stocks`

종목 목록을 조회합니다. 필터링, 정렬, 페이징을 지원합니다.

#### Query Parameters

| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| page | integer | N | 1 | 페이지 번호 (1부터 시작) |
| limit | integer | N | 20 | 페이지 크기 (최소: 1, 최대: 100) |
| market | string | N | - | 시장 필터 (KOSPI, KOSDAQ, KONEX, ETF) |
| status | string | N | ACTIVE | 상태 필터 (ACTIVE, SUSPENDED, DELISTED, ALL) |
| is_tradable | boolean | N | - | 거래 가능 여부 필터 (true, false) |
| search | string | N | - | 종목명 또는 코드 검색 (부분 일치) |
| sort | string | N | symbol | 정렬 기준 (symbol, name, market_cap) |
| order | string | N | asc | 정렬 방향 (asc, desc) |

#### Request Example

```http
GET /api/stocks?market=KOSPI&status=ACTIVE&is_tradable=true&page=1&limit=20
```

#### Response (200 OK)

```json
{
  "data": [
    {
      "symbol": "005930",
      "name": "삼성전자",
      "market": "KOSPI",
      "status": "ACTIVE",
      "listing_date": "1975-06-11",
      "delisting_date": null,
      "sector": "전기전자",
      "industry": "반도체",
      "market_cap": 500000000000000,
      "is_tradable": true,
      "trade_halt_reason": null,
      "created_ts": "2026-01-13T00:00:00Z",
      "updated_ts": "2026-01-14T00:00:00Z"
    },
    {
      "symbol": "000660",
      "name": "SK하이닉스",
      "market": "KOSPI",
      "status": "ACTIVE",
      "listing_date": "1996-12-26",
      "delisting_date": null,
      "sector": "전기전자",
      "industry": "반도체",
      "market_cap": 80000000000000,
      "is_tradable": true,
      "trade_halt_reason": null,
      "created_ts": "2026-01-13T00:00:00Z",
      "updated_ts": "2026-01-14T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total_pages": 50,
    "total_count": 1000,
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

**400 Bad Request** - 잘못된 파라미터

```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid market value",
    "details": "market must be one of: KOSPI, KOSDAQ, KONEX, ETF",
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
    "message": "Failed to query stocks",
    "details": "Database connection error",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

### 2. 종목 상세 조회

**GET** `/api/stocks/:symbol`

특정 종목의 상세 정보를 조회합니다.

#### Path Parameters

| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| symbol | string | Y | 종목 코드 (6자리 숫자, 예: 005930) |

#### Request Example

```http
GET /api/stocks/005930
```

#### Response (200 OK)

```json
{
  "data": {
    "symbol": "005930",
    "name": "삼성전자",
    "market": "KOSPI",
    "status": "ACTIVE",
    "listing_date": "1975-06-11",
    "delisting_date": null,
    "sector": "전기전자",
    "industry": "반도체",
    "market_cap": 500000000000000,
    "is_tradable": true,
    "trade_halt_reason": null,
    "created_ts": "2026-01-13T00:00:00Z",
    "updated_ts": "2026-01-14T00:00:00Z"
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

**404 Not Found** - 종목이 존재하지 않음

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Stock not found",
    "details": "No stock found with symbol: 999999",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

## 데이터 모델

### Stock

| 필드 | 타입 | 설명 |
|------|------|------|
| symbol | string | 종목 코드 (6자리 숫자) |
| name | string | 종목명 |
| market | string | 시장 구분 (KOSPI, KOSDAQ, KONEX) |
| status | string | 상태 (ACTIVE, SUSPENDED, DELISTED) |
| listing_date | string \| null | 상장일 (YYYY-MM-DD) |
| delisting_date | string \| null | 상장폐지일 (YYYY-MM-DD) |
| sector | string \| null | 섹터 |
| industry | string \| null | 업종 |
| market_cap | integer \| null | 시가총액 (원) |
| is_tradable | boolean | 거래 가능 여부 |
| trade_halt_reason | string \| null | 거래정지 사유 |
| created_ts | string | 생성 일시 (ISO 8601) |
| updated_ts | string | 수정 일시 (ISO 8601) |

---

## 비즈니스 규칙

### 종목 코드 형식
- 6자리 숫자 (`005930`, `000660`)
- KIS API와 동일한 형식
- 선행 0 포함

### 기본 필터링
- 별도 지정이 없으면 `status=ACTIVE`인 종목만 반환
- `status=ALL`로 모든 상태 조회 가능

### 정렬
- 기본 정렬: symbol 오름차순
- 지원 정렬 필드: symbol, name, market_cap
- 시가총액 정렬 시 NULL 값은 마지막에 배치

### 검색
- `search` 파라미터: 종목명 또는 종목코드 부분 일치
- 대소문자 구분 없음
- 공백 제거 후 검색

### 페이징
- 기본 페이지 크기: 20
- 최대 페이지 크기: 100
- 최소 페이지 크기: 1

---

## 구현 위치

### Backend (Go)

```
backend/internal/
├── domain/
│   └── stock/
│       └── model.go           # Stock 도메인 모델
├── infra/
│   └── database/
│       └── postgres/
│           └── stock_repository.go  # PostgreSQL CRUD
└── api/
    └── handlers/
        └── stock.go           # HTTP 핸들러
```

### 의존성

- **Database**: market.stocks 테이블 (READ ONLY)
- **Repository**: StockRepository 인터페이스
- **Response**: `internal/api/response` 패키지

---

## 권한

**현재 단계**: 인증/인가 없음 (추후 추가 예정)

**DB 권한**: aegis_trade role (SELECT only on market.stocks)

---

## 테스트 시나리오

### 정상 시나리오
1. ✅ 기본 목록 조회 (페이징)
2. ✅ 시장별 필터링 (KOSPI, KOSDAQ)
3. ✅ 거래 가능 종목만 조회
4. ✅ 종목명 검색
5. ✅ 종목 코드 검색
6. ✅ 시가총액 내림차순 정렬
7. ✅ 특정 종목 상세 조회

### 에러 시나리오
1. ❌ 잘못된 페이지 번호 (0, -1)
2. ❌ 잘못된 limit (0, 101)
3. ❌ 잘못된 market 값 (NASDAQ)
4. ❌ 잘못된 종목 코드 형식 (5자리, 7자리, 문자 포함)
5. ❌ 존재하지 않는 종목 조회

---

## 성능 고려사항

### 인덱스 활용
- market 필터: `idx_stocks_market`
- status 필터: `idx_stocks_status`
- is_tradable 필터: `idx_stocks_tradable` (partial index)
- 종목명 검색: `idx_stocks_name`

### 쿼리 최적화
- COUNT(*) 최적화: 필터 조건 동일하게 유지
- LIMIT/OFFSET 대신 Keyset Pagination 고려 (향후)

### 캐싱
- 종목 마스터는 변경이 드물어 캐싱에 적합
- Redis 캐시 추가 고려 (향후)

---

## 향후 개선

1. **인증/인가**: JWT 기반 인증 추가
2. **Keyset Pagination**: OFFSET 대신 커서 기반 페이징
3. **Redis 캐싱**: 자주 조회되는 종목 캐싱
4. **Bulk 조회**: 여러 종목 한번에 조회 (POST /api/stocks/bulk)
5. **필터 확장**: 섹터별, 업종별, 시가총액 범위 필터
6. **Rate Limiting**: API 호출 제한

---

## 참고 문서

- [API Common Spec](./common.md)
- [Database Schema](../database/schema.md)
- [Module Catalog](../modules/module-catalog.md)
