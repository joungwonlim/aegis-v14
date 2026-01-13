# API (API 설계)

이 폴더는 v14 시스템의 API 엔드포인트 설계 문서를 포함합니다.

---

## 📋 문서 목록

### 엔드포인트별 문서

| 문서 | 엔드포인트 | 설명 |
|------|-----------|------|
| `stocks.md` | `/api/stocks/*` | 종목 조회/관리 |
| `signals.md` | `/api/signals/*` | 시그널 조회 |
| `portfolio.md` | `/api/portfolio/*` | 포트폴리오 조회/관리 |
| `orders.md` | `/api/orders/*` | 주문 조회/실행 |
| `performance.md` | `/api/performance/*` | 성과 분석 조회 |
| `common.md` | - | 공통 스펙 (인증, 에러, 페이지네이션) |

---

## 🎯 API 설계 원칙

### 1. RESTful 설계
```
GET    /api/stocks       # 목록 조회
GET    /api/stocks/:id   # 단일 조회
POST   /api/stocks       # 생성
PUT    /api/stocks/:id   # 전체 수정
PATCH  /api/stocks/:id   # 부분 수정
DELETE /api/stocks/:id   # 삭제
```

### 2. 일관된 응답 구조
```json
{
  "data": { ... },        // 성공 시 데이터
  "error": { ... },       // 실패 시 에러
  "meta": { ... }         // 메타 정보 (페이지네이션 등)
}
```

### 3. 명확한 에러 코드
```json
{
  "error": {
    "code": "STOCK_NOT_FOUND",
    "message": "종목을 찾을 수 없습니다",
    "details": {
      "stock_code": "005930"
    }
  }
}
```

### 4. API 버저닝
```
/api/v1/stocks    # 버전 1
/api/v2/stocks    # 버전 2 (호환성 깨질 때)
```

---

## 📝 API 문서 템플릿

각 엔드포인트는 다음 구조를 따라야 합니다:

```markdown
## GET /api/stocks

**목적**: 종목 목록 조회

### Request

**Query Parameters**:
| 파라미터 | 타입 | 필수 | 기본값 | 설명 |
|----------|------|------|--------|------|
| market | string | N | - | 시장 필터 |
| page | int | N | 1 | 페이지 번호 |
| limit | int | N | 20 | 페이지 크기 |

**Headers**:
```
Authorization: Bearer {token}
```

### Response

**200 OK**:
\`\`\`json
{
  "data": [...],
  "meta": {
    "pagination": { ... }
  }
}
\`\`\`

**400 Bad Request**:
\`\`\`json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "..."
  }
}
\`\`\`

### 구현 위치
- Handler: `backend/internal/api/handlers/stocks.go`
- Service: `backend/internal/stocks/service.go`

### 테스트 시나리오
1. 정상 케이스
2. 에러 케이스
```

---

## 🔐 인증/인가

### 인증 방식 (선택 필요)

#### Option 1: JWT
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Option 2: API Key
```
X-API-Key: your-api-key-here
```

#### Option 3: Session
```
Cookie: session_id=abc123...
```

### 인가 레벨

| 레벨 | 권한 | 예시 |
|------|------|------|
| Public | 인증 불필요 | 시장 정보 조회 |
| User | 일반 사용자 | 포트폴리오 조회 |
| Admin | 관리자 | 시스템 설정 변경 |
| System | 내부 시스템 | 자동 트레이딩 실행 |

---

## 🚨 에러 코드 설계

### HTTP Status Codes

| 코드 | 의미 | 사용 시점 |
|------|------|----------|
| 200 | OK | 성공 |
| 201 | Created | 리소스 생성 성공 |
| 400 | Bad Request | 잘못된 요청 |
| 401 | Unauthorized | 인증 실패 |
| 403 | Forbidden | 권한 없음 |
| 404 | Not Found | 리소스 없음 |
| 409 | Conflict | 충돌 (중복 생성 등) |
| 422 | Unprocessable Entity | 검증 실패 |
| 500 | Internal Server Error | 서버 오류 |
| 503 | Service Unavailable | 서비스 일시 중단 |

### 비즈니스 에러 코드

```
STOCK_NOT_FOUND
INVALID_STOCK_CODE
MARKET_CLOSED
INSUFFICIENT_BALANCE
ORDER_LIMIT_EXCEEDED
SIGNAL_GENERATION_FAILED
PORTFOLIO_REBALANCE_FAILED
```

---

## 📄 페이지네이션

### Offset-based (권장)

```json
// Request
GET /api/stocks?page=2&limit=20

// Response
{
  "data": [...],
  "meta": {
    "pagination": {
      "page": 2,
      "limit": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

### Cursor-based (대용량 데이터)

```json
// Request
GET /api/stocks?cursor=eyJpZCI6MTIzfQ==&limit=20

// Response
{
  "data": [...],
  "meta": {
    "pagination": {
      "next_cursor": "eyJpZCI6MTQzfQ==",
      "has_more": true
    }
  }
}
```

---

## 🔍 필터링 및 정렬

### 필터링

```
GET /api/stocks?market=KOSPI&price_min=10000&price_max=50000
```

### 정렬

```
GET /api/stocks?sort=price:desc,volume:asc
```

### 검색

```
GET /api/stocks?q=삼성
```

---

## 📊 Rate Limiting

### 제한 정책

| 레벨 | 제한 | 예시 |
|------|------|------|
| Public | 100 req/min | 시장 정보 조회 |
| User | 1000 req/min | 포트폴리오 조회 |
| System | Unlimited | 내부 시스템 |

### 응답 헤더

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1610000000
```

### 초과 시

```http
HTTP/1.1 429 Too Many Requests

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "요청 한도를 초과했습니다",
    "retry_after": 60
  }
}
```

---

## 🧪 테스트 가이드

### cURL 예시

```bash
# 종목 목록 조회
curl -X GET "http://localhost:8080/api/stocks?market=KOSPI" \
  -H "Authorization: Bearer {token}"

# 종목 생성
curl -X POST "http://localhost:8080/api/stocks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {token}" \
  -d '{
    "code": "005930",
    "name": "삼성전자",
    "market": "KOSPI"
  }'
```

---

## ✅ 설계 검증 체크리스트

API 설계 완료 시:

- [ ] 모든 엔드포인트 정의
- [ ] Request/Response 스키마 정의
- [ ] 에러 코드 정의
- [ ] 인증/인가 전략 정의
- [ ] 페이지네이션 방식 정의
- [ ] Rate Limiting 정책 정의
- [ ] 테스트 시나리오 작성
- [ ] 구현 위치 명시

---

## 🔗 참고

- [CLAUDE.md](../../CLAUDE.md) - API 설계 템플릿
- [modules/](../modules/) - 각 모듈의 기능
- [database/](../database/) - 데이터 모델
- REST API Best Practices
- OpenAPI Specification (선택)
