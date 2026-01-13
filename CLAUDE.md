# CLAUDE.md

> Aegis v14 - 퀀트 트레이딩 시스템 설계 문서

---

## 🚨 현재 단계: 설계 (Design Phase)

**중요**: v14는 현재 **설계 단계**입니다. 코드 작성이 아닌 **문서 작성과 아키텍처 설계**에 집중합니다.

```
현재 작업: 설계 문서 작성 ✍️
다음 단계: 구현 (추후)
```

---

## 절대 규칙 (Non-negotiable)

### 1. 한국어 응답 필수
모든 응답은 한국어로 작성 (코드/커밋 메시지 제외)

### 2. 설계 문서 우선
코드보다 설계 문서가 먼저. 문서 없이 구현 금지.

### 3. SSOT 준수
정해진 위치에서만 해당 책임의 설계 정의.

### 4. 모듈 독립성
각 모듈은 독립적으로 설계되어야 함. 인터페이스로만 연결.

### 5. 문서 생성 시 즉시 등록
새 문서 생성 시 `docs/_index.md`에 즉시 등록

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 14+ (App Router) + shadcn/ui |
| Backend | Go 1.21+ (BFF) |
| Database | PostgreSQL 15+ |

---

## 프로젝트 구조 (설계 단계)

```
v14/
├── docs/                      # 📋 설계 문서 (최우선)
│   ├── _index.md             # 문서 등록부
│   ├── architecture/         # 아키텍처 설계
│   │   ├── system-overview.md
│   │   ├── data-flow.md
│   │   └── layer-design.md
│   ├── modules/              # 모듈별 설계
│   │   ├── s0-data.md
│   │   ├── s1-universe.md
│   │   ├── s2-signals.md
│   │   └── ...
│   ├── database/             # DB 설계
│   │   ├── schema.md
│   │   ├── erd.md
│   │   └── migration-plan.md
│   ├── api/                  # API 설계
│   │   ├── endpoints.md
│   │   └── contracts.md
│   └── ui/                   # UI 설계
│       ├── pages.md
│       └── components.md
├── backend/                   # (구현 단계에서 생성)
└── frontend/                  # (구현 단계에서 생성)
```

---

## 설계 단계 작업 순서 (강제)

### 1. 시스템 아키텍처 설계
- [ ] 전체 시스템 개요
- [ ] 데이터 흐름 정의
- [ ] 레이어 구조 설계

### 2. 모듈 설계
- [ ] 각 모듈의 책임 정의
- [ ] 모듈 간 인터페이스 설계
- [ ] 의존성 방향 정의

### 3. 데이터베이스 설계
- [ ] ERD 작성
- [ ] 테이블 스키마 정의
- [ ] 인덱스 전략
- [ ] 마이그레이션 계획

### 4. API 설계
- [ ] 엔드포인트 정의
- [ ] Request/Response 스키마
- [ ] 에러 코드 정의

### 5. UI 설계
- [ ] 페이지 구조
- [ ] 컴포넌트 계층
- [ ] 상태 관리 전략

---

## 설계 문서 SSOT 규칙

| 문서 종류 | 필수 위치 | 금지 위치 |
|-----------|----------|----------|
| 시스템 아키텍처 | `docs/architecture/` | 프로젝트 루트 |
| 모듈 설계 | `docs/modules/` | backend/, frontend/ |
| DB 스키마 | `docs/database/` | backend/ |
| API 스펙 | `docs/api/` | backend/internal/ |
| UI 설계 | `docs/ui/` | frontend/src/ |

**중요**: 모든 설계 문서는 `docs/` 아래에만 생성

---

## 설계 문서 템플릿

### 모듈 설계 템플릿 (`docs/modules/`)

```markdown
# [모듈명] 설계

## 개요
- **책임**: 이 모듈이 담당하는 핵심 기능
- **위치**: `backend/internal/[module_name]/`
- **의존성**: 다른 모듈 의존 관계

## 인터페이스 설계

### 외부 제공 인터페이스
\`\`\`go
type [ModuleName]Service interface {
    Method1(ctx context.Context, param Type) (Result, error)
    Method2(ctx context.Context, param Type) (Result, error)
}
\`\`\`

### 내부 의존 인터페이스
\`\`\`go
type [Dependency]Client interface {
    Fetch(ctx context.Context) (Data, error)
}
\`\`\`

## 데이터 모델

### 입력
\`\`\`go
type Input struct {
    Field1 string
    Field2 int
}
\`\`\`

### 출력
\`\`\`go
type Output struct {
    Result string
    Status string
}
\`\`\`

## 처리 흐름

1. 단계 1: ...
2. 단계 2: ...
3. 단계 3: ...

## 에러 처리

| 에러 | 조건 | 처리 방법 |
|------|------|----------|
| ErrInvalidInput | 입력 검증 실패 | 400 반환 |
| ErrNotFound | 데이터 없음 | 404 반환 |

## 성능 고려사항

- 예상 처리량: ...
- 캐시 전략: ...
- 최적화 포인트: ...

## 보안 고려사항

- 인증/인가: ...
- 데이터 보호: ...

## 테스트 전략

- 단위 테스트: ...
- 통합 테스트: ...
- E2E 테스트: ...
```

---

### DB 스키마 설계 템플릿 (`docs/database/`)

```markdown
# 데이터베이스 스키마 설계

## ERD

\`\`\`mermaid
erDiagram
    STOCKS ||--o{ PRICES : has
    STOCKS {
        string code PK
        string name
        string market
        timestamp created_at
    }
    PRICES {
        uuid id PK
        string stock_code FK
        decimal price
        bigint volume
        timestamp traded_at
    }
\`\`\`

## 테이블 설계

### stocks

**목적**: 종목 기본 정보

| 컬럼명 | 타입 | 제약 | 설명 |
|--------|------|------|------|
| code | VARCHAR(10) | PK | 종목 코드 |
| name | VARCHAR(100) | NOT NULL | 종목명 |
| market | VARCHAR(20) | NOT NULL | 시장 구분 |
| created_at | TIMESTAMP | NOT NULL | 생성일시 |

**인덱스**:
- PK: `code`
- IDX: `market`

### prices

**목적**: 가격 데이터

| 컬럼명 | 타입 | 제약 | 설명 |
|--------|------|------|------|
| id | UUID | PK | 고유 ID |
| stock_code | VARCHAR(10) | FK | 종목 코드 |
| price | DECIMAL(10,2) | NOT NULL | 가격 |
| volume | BIGINT | NOT NULL | 거래량 |
| traded_at | TIMESTAMP | NOT NULL | 거래 시각 |

**인덱스**:
- PK: `id`
- IDX: `stock_code, traded_at DESC`
- FK: `stock_code` → `stocks(code)`

## 마이그레이션 계획

1. `000001_create_stocks_table.sql`
2. `000002_create_prices_table.sql`
3. `000003_add_indexes.sql`
```

---

### API 설계 템플릿 (`docs/api/`)

```markdown
# API 엔드포인트 설계

## GET /api/stocks

**목적**: 종목 목록 조회

### Request

**Query Parameters**:
| 파라미터 | 타입 | 필수 | 설명 |
|----------|------|------|------|
| market | string | N | 시장 필터 (KOSPI, KOSDAQ) |
| page | int | N | 페이지 번호 (기본: 1) |
| limit | int | N | 페이지 크기 (기본: 20, 최대: 100) |

### Response

**200 OK**:
\`\`\`json
{
  "data": [
    {
      "code": "005930",
      "name": "삼성전자",
      "market": "KOSPI",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
\`\`\`

**400 Bad Request**:
\`\`\`json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid market value"
  }
}
\`\`\`

### 구현 위치
- Handler: `backend/internal/api/handlers/stocks.go`
- Service: `backend/internal/stocks/service.go`
- Repository: `backend/internal/stocks/repository.go`
```

---

## 설계 검증 체크리스트 (BLOCKER)

새 설계 문서 작성 시 **반드시** 확인:

### ✅ Step 1: SSOT 위치 확인
- [ ] 문서가 올바른 `docs/` 하위 폴더에 생성되었는가?
- [ ] 중복된 설계 문서가 없는가?

### ✅ Step 2: 문서 등록
- [ ] `docs/_index.md`에 새 문서가 등록되었는가?

### ✅ Step 3: 템플릿 준수
- [ ] 해당 문서 종류의 템플릿을 따랐는가?
- [ ] 필수 섹션이 모두 포함되었는가?

### ✅ Step 4: 설계 일관성
- [ ] 다른 모듈과의 인터페이스가 명확한가?
- [ ] 순환 참조가 없는가?
- [ ] 의존성 방향이 올바른가?

### ✅ Step 5: 완성도
- [ ] 모호한 부분이 없는가?
- [ ] 구현 가능한 수준으로 구체적인가?

---

## 모듈 독립성 설계 원칙

### 1. Interface-First Design

```go
// ✅ CORRECT - 인터페이스 먼저 정의
type FetcherService interface {
    FetchStocks(ctx context.Context) ([]Stock, error)
}

type BrainService interface {
    Analyze(ctx context.Context, stocks []Stock) ([]Signal, error)
}

// Brain은 FetcherService 인터페이스에만 의존
func NewBrainService(fetcher FetcherService) *BrainService
```

### 2. 의존성 방향

```
하위 레이어 → 상위 레이어 (금지)
상위 레이어 → 하위 레이어 (허용, 인터페이스 통해서만)
```

```
pkg/           # 최하위 (의존성 없음)
  ↑
internal/contracts/  # 타입/인터페이스 정의
  ↑
internal/modules/    # 비즈니스 로직 (인터페이스로만 의존)
  ↑
internal/api/        # HTTP 핸들러 (최상위)
```

### 3. 순환 참조 금지

```
❌ 금지:
ModuleA → ModuleB → ModuleA

✅ 허용:
ModuleA → Interface
ModuleB → Interface
```

---

## 설계 단계 커밋 규칙

```
docs(scope): 설계 문서 작성/수정 내용

예시:
docs(architecture): 시스템 아키텍처 초안 작성
docs(database): ERD 및 스키마 설계 추가
docs(api): 종목 조회 API 설계 추가
docs(modules): S2 시그널 모듈 설계 작성
```

---

## 설계 완료 기준 (Definition of Done)

### 시스템 아키텍처
- [ ] 전체 시스템 구조도 작성
- [ ] 데이터 흐름 다이어그램 작성
- [ ] 레이어별 책임 정의
- [ ] 기술 스택 선정 및 근거

### 모듈 설계
- [ ] 모든 모듈의 인터페이스 정의
- [ ] 모듈 간 의존성 다이어그램
- [ ] 각 모듈의 데이터 모델
- [ ] 처리 흐름 정의

### 데이터베이스 설계
- [ ] 완전한 ERD
- [ ] 모든 테이블 스키마
- [ ] 인덱스 전략
- [ ] 마이그레이션 순서

### API 설계
- [ ] 모든 엔드포인트 정의
- [ ] Request/Response 스키마
- [ ] 에러 코드 정의
- [ ] 인증/인가 전략

### UI 설계
- [ ] 페이지 구조
- [ ] 컴포넌트 계층
- [ ] 상태 관리 전략
- [ ] API 연동 방안

---

## 금지 패턴 (설계 단계)

### ❌ 문서 없이 코드 작성
```
현재는 설계 단계입니다. 코드는 작성하지 않습니다.
```

### ❌ 모호한 설계
```markdown
❌ 나쁜 예:
"데이터를 처리한다"

✅ 좋은 예:
"KIS API에서 종목 데이터를 fetch하여 PostgreSQL stocks 테이블에 저장한다.
실패 시 3회 재시도하며, 최종 실패 시 에러 로그를 남긴다."
```

### ❌ 중복된 설계 문서
```
❌ 나쁜 예:
docs/modules/stocks.md
docs/api/stocks-api.md
docs/database/stocks-schema.md
(같은 내용이 여러 곳에 흩어짐)

✅ 좋은 예:
docs/modules/stocks.md (모듈 설계)
- API 섹션: "API 상세는 docs/api/stocks.md 참고"
- DB 섹션: "스키마는 docs/database/schema.md의 stocks 테이블 참고"
```

### ❌ 순환 참조 설계
```
❌ 금지:
Brain → Fetcher → Brain

✅ 허용:
Brain → FetcherInterface
Fetcher → (구현)
```

---

## 설계 검토 프로세스

### 1. 자가 검토
작성자가 체크리스트로 자가 검토

### 2. 일관성 검토
다른 문서와의 일관성 확인

### 3. 구현 가능성 검토
실제 구현 시 문제가 없는지 검토

### 4. 승인
설계 문서 승인 후 다음 단계 진행

---

## v10/v13 참고 시 주의사항

### v10/v13의 좋은 점 (참고)
- 검증된 아키텍처 패턴
- 모듈 구조
- API 설계 방식
- DB 스키마 구조

### v10/v13에서 개선할 점 (v14에서 반영)
- 더 명확한 모듈 경계
- 더 단순한 설계
- 더 나은 문서화
- 더 엄격한 SSOT

### 참고 방법
```
1. v10/v13 설계 문서 읽기
2. 핵심 아이디어 추출
3. v14에 맞게 단순화/개선
4. v14 규칙에 맞게 재작성
```

**절대 복사/붙여넣기 금지!**

---

## 다음 단계 (구현 단계 전환 시)

설계가 완료되면:

1. **설계 검토 완료 체크**
   - [ ] 모든 설계 문서 작성
   - [ ] 상호 일관성 확인
   - [ ] 구현 가능성 검증

2. **CLAUDE.md 업데이트**
   - 설계 단계 → 구현 단계 전환
   - 코드 작성 규칙 활성화
   - Quality Gates 추가

3. **구현 착수**
   - 설계 문서 기반으로 코드 작성
   - 설계와 코드 동기화 유지

---

## 빠른 체크리스트

설계 문서 작성 시:

- [ ] 올바른 `docs/` 위치에 작성했나?
- [ ] `docs/_index.md`에 등록했나?
- [ ] 템플릿을 따랐나?
- [ ] 다른 설계와 일관성이 있나?
- [ ] 구체적이고 모호하지 않은가?
- [ ] 순환 참조가 없는가?
- [ ] 구현 가능한 수준인가?

---

## 참고 문서

- v10: `/Users/wonny/Dev/aegis/v10/CLAUDE.md`
- v13: `/Users/wonny/Dev/aegis/v13/CLAUDE.md`

**Remember**: v14는 지금 **설계 단계**입니다. 코드가 아닌 **문서 작성**에 집중하세요.

---

**Version**: v14.0.0-design
**Phase**: 설계 (Design)
**Last Updated**: 2026-01-13
