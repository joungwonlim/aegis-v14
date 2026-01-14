# API 공통 스펙 (Common API Specification)

> **목적**: 모든 API 엔드포인트가 따라야 하는 공통 규칙과 구조 정의

**Last Updated**: 2026-01-14

---

## 🎯 기본 원칙

### 1. RESTful 설계
- HTTP 메서드 의미에 맞게 사용
- 리소스 중심의 URL 설계
- 적절한 HTTP 상태 코드 사용

### 2. 일관된 응답 형식
- 모든 응답은 JSON 형식
- 성공/실패 모두 일관된 구조
- 타임스탬프는 RFC3339 형식

### 3. 명확한 에러 처리
- 에러 코드는 대문자 스네이크 케이스
- 에러 메시지는 사용자 친화적
- 디버깅을 위한 request_id 포함

---

## 🌐 Base URL

```
Development: http://localhost:8099/api
Production:  https://api.aegis.com/api
```

---

## 📦 공통 응답 구조

### 성공 응답 (2xx)

#### 단일 리소스
```json
{
  "data": {
    "id": "123",
    "name": "Example",
    "created_at": "2026-01-14T12:00:00Z"
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### 리스트 (Pagination 없음)
```json
{
  "data": [
    {"id": "1", "name": "Item 1"},
    {"id": "2", "name": "Item 2"}
  ],
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "count": 2
  }
}
```

#### 리스트 (Pagination 있음)
```json
{
  "data": [
    {"id": "1", "name": "Item 1"},
    {"id": "2", "name": "Item 2"}
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total_pages": 5,
    "total_count": 100,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### 생성 성공 (201 Created)
```json
{
  "data": {
    "id": "new-123",
    "created_at": "2026-01-14T12:00:00Z"
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "message": "Resource created successfully"
  }
}
```

#### 삭제 성공 (204 No Content)
- Body 없음

#### 업데이트 성공 (200 OK)
```json
{
  "data": {
    "id": "123",
    "updated_at": "2026-01-14T12:00:00Z"
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "message": "Resource updated successfully"
  }
}
```

---

## ❌ 에러 응답 구조

### 기본 에러 응답
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "User-friendly error message",
    "details": "Additional technical details (optional)",
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

### Validation 에러 (400 Bad Request)
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "fields": [
      {
        "field": "email",
        "message": "Invalid email format"
      },
      {
        "field": "age",
        "message": "Must be greater than 0"
      }
    ]
  }
}
```

---

## 🔢 HTTP 상태 코드

### 2xx Success
| 코드 | 의미 | 사용 예시 |
|------|------|----------|
| 200 | OK | GET, PUT 성공 |
| 201 | Created | POST 성공 (리소스 생성) |
| 204 | No Content | DELETE 성공 |

### 4xx Client Error
| 코드 | 의미 | 에러 코드 | 사용 예시 |
|------|------|-----------|----------|
| 400 | Bad Request | VALIDATION_ERROR | 잘못된 파라미터 |
| 400 | Bad Request | INVALID_PARAMETER | 파라미터 형식 오류 |
| 401 | Unauthorized | UNAUTHORIZED | 인증 실패 |
| 403 | Forbidden | FORBIDDEN | 권한 없음 |
| 404 | Not Found | NOT_FOUND | 리소스 없음 |
| 409 | Conflict | CONFLICT | 중복 리소스 |
| 422 | Unprocessable Entity | BUSINESS_RULE_VIOLATION | 비즈니스 규칙 위반 |
| 429 | Too Many Requests | RATE_LIMIT_EXCEEDED | Rate limit 초과 |

### 5xx Server Error
| 코드 | 의미 | 에러 코드 | 사용 예시 |
|------|------|-----------|----------|
| 500 | Internal Server Error | INTERNAL_SERVER_ERROR | 서버 오류 |
| 502 | Bad Gateway | BAD_GATEWAY | 외부 API 오류 |
| 503 | Service Unavailable | SERVICE_UNAVAILABLE | 서비스 점검 |
| 504 | Gateway Timeout | GATEWAY_TIMEOUT | 타임아웃 |

---

## 📋 에러 코드 정의

### 일반 에러
| 에러 코드 | HTTP 상태 | 설명 |
|-----------|----------|------|
| INTERNAL_SERVER_ERROR | 500 | 예상치 못한 서버 오류 |
| INVALID_PARAMETER | 400 | 잘못된 파라미터 |
| VALIDATION_ERROR | 400 | 검증 실패 |
| NOT_FOUND | 404 | 리소스를 찾을 수 없음 |
| UNAUTHORIZED | 401 | 인증 실패 |
| FORBIDDEN | 403 | 권한 없음 |
| CONFLICT | 409 | 리소스 충돌 |
| RATE_LIMIT_EXCEEDED | 429 | Rate limit 초과 |

### 데이터베이스 관련
| 에러 코드 | HTTP 상태 | 설명 |
|-----------|----------|------|
| DATABASE_ERROR | 500 | DB 연결/쿼리 오류 |
| DUPLICATE_ENTRY | 409 | 중복 데이터 |
| CONSTRAINT_VIOLATION | 422 | 제약 조건 위반 |

### 외부 API 관련
| 에러 코드 | HTTP 상태 | 설명 |
|-----------|----------|------|
| EXTERNAL_API_ERROR | 502 | 외부 API 오류 |
| EXTERNAL_API_TIMEOUT | 504 | 외부 API 타임아웃 |

---

## 🔄 Pagination

### Query Parameters
```
page:  페이지 번호 (1부터 시작, 기본값: 1)
limit: 페이지당 항목 수 (기본값: 20, 최대: 100)
```

### 예시 요청
```
GET /api/stocks?page=2&limit=50
```

### 응답
```json
{
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 50,
    "total_pages": 10,
    "total_count": 487,
    "has_next": true,
    "has_prev": true
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

## 🔍 Filtering & Sorting

### Query Parameters

#### Filtering
```
# 단일 필터
GET /api/stocks?market=KOSPI

# 다중 필터 (AND)
GET /api/stocks?market=KOSPI&sector=IT

# 범위 필터
GET /api/prices?start_date=2026-01-01&end_date=2026-01-14

# 검색
GET /api/stocks?search=삼성
```

#### Sorting
```
# 오름차순 (기본)
GET /api/stocks?sort=name

# 내림차순
GET /api/stocks?sort=-created_at

# 다중 정렬
GET /api/stocks?sort=market,-name
```

---

## 📨 Request Headers

### 필수 헤더
```
Content-Type: application/json
```

### 선택 헤더
```
X-Request-ID: 클라이언트가 생성한 요청 ID (없으면 서버가 생성)
Authorization: Bearer <token> (인증이 필요한 경우)
```

---

## 📤 Response Headers

### 공통 헤더
```
Content-Type: application/json; charset=utf-8
X-Request-ID: req-abc123
```

### Rate Limiting (향후 추가)
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642176000
```

---

## 📝 필드 네이밍 규칙

### JSON 필드
- **스네이크 케이스** 사용: `created_at`, `stock_code`, `user_id`
- 불린은 `is_`, `has_` 접두사: `is_active`, `has_permission`
- 날짜/시간은 `_at` 접미사: `created_at`, `updated_at`, `traded_at`

### URL Path
- **케밥 케이스** 사용: `/api/stock-prices`, `/api/market-data`
- 리소스는 복수형: `/api/stocks`, `/api/users`

---

## 🔒 보안

### CORS
- Development: 모든 origin 허용
- Production: 허용된 도메인만

### Rate Limiting
- IP 기반 제한
- 기본: 1000 requests / hour
- 초과 시 429 응답

---

## 🎯 구현 위치

### Response Helpers
- **위치**: `internal/api/response/response.go`
- **책임**: 공통 응답 구조 생성

```go
// 성공 응답
response.Success(c, data)
response.SuccessWithPagination(c, data, pagination)
response.Created(c, data, "Resource created")

// 에러 응답
response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input")
response.ValidationError(c, validationErrors)
response.NotFound(c, "Stock not found")
response.InternalError(c, err)
```

### Error Codes
- **위치**: `internal/api/response/error.go`
- **책임**: 에러 코드 상수 정의

```go
const (
    ErrCodeInternalServer = "INTERNAL_SERVER_ERROR"
    ErrCodeValidation     = "VALIDATION_ERROR"
    ErrCodeNotFound       = "NOT_FOUND"
    // ...
)
```

### Middleware
- **위치**: `internal/api/middleware/`
- **적용 순서**:
  1. Recovery (패닉 복구)
  2. RequestID (요청 ID 생성)
  3. Logging (요청/응답 로깅)
  4. CORS (CORS 설정)
  5. (향후) RateLimit

---

## 📖 예시

### GET - 단일 리소스 조회
```http
GET /api/stocks/005930
```

**성공 (200)**:
```json
{
  "data": {
    "code": "005930",
    "name": "삼성전자",
    "market": "KOSPI",
    "sector": "IT",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

**실패 (404)**:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Stock not found",
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

### GET - 리스트 조회 (Pagination)
```http
GET /api/stocks?page=1&limit=20&market=KOSPI
```

**성공 (200)**:
```json
{
  "data": [
    {
      "code": "005930",
      "name": "삼성전자",
      "market": "KOSPI"
    },
    {
      "code": "000660",
      "name": "SK하이닉스",
      "market": "KOSPI"
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
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

### POST - 리소스 생성
```http
POST /api/stocks
Content-Type: application/json

{
  "code": "005930",
  "name": "삼성전자",
  "market": "KOSPI"
}
```

**성공 (201)**:
```json
{
  "data": {
    "code": "005930",
    "name": "삼성전자",
    "market": "KOSPI",
    "created_at": "2026-01-14T12:00:00Z"
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "message": "Stock created successfully"
  }
}
```

**실패 (400 Validation)**:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z",
    "fields": [
      {
        "field": "code",
        "message": "Stock code is required"
      },
      {
        "field": "market",
        "message": "Market must be KOSPI or KOSDAQ"
      }
    ]
  }
}
```

---

## ✅ 체크리스트

새 API 엔드포인트 추가 시:
- [ ] RESTful URL 설계 준수
- [ ] 적절한 HTTP 메서드 사용
- [ ] 공통 응답 구조 사용
- [ ] 에러 처리 구현
- [ ] Request ID 전파
- [ ] 로깅 추가
- [ ] Validation 구현
- [ ] 문서 업데이트

---

## 참고 문서

- [Health Check API](./health.md)
- [Stocks API](./stocks.md)
- [로깅 전략](../operations/logging-strategy.md)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
