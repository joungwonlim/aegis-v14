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

### 6. shadcn/ui 컴포넌트 설치 필수 (CRITICAL)
**UI 컴포넌트 사용 전 반드시 설치 확인!**

```bash
# ❌ 금지: 설치 없이 import
import { Input } from '@/components/ui/input'  // 빌드 실패!

# ✅ 필수: 먼저 설치, 그 다음 import
npx shadcn@latest add input
npx shadcn@latest add label
npx shadcn@latest add radio-group
```

**설치 체크리스트 (컴포넌트 사용 전 필수):**
```
□ 사용할 shadcn/ui 컴포넌트 목록 작성
□ 각 컴포넌트가 이미 설치되어 있는지 확인 (frontend/components/ui/ 폴더 확인)
□ 미설치 컴포넌트는 `npx shadcn@latest add {component}` 실행
□ 설치 완료 후 import 문 작성
```

**주요 shadcn/ui 컴포넌트:**
- `button` - Button 컴포넌트
- `input` - Input 입력 필드
- `label` - Label 레이블
- `radio-group` - RadioGroup 라디오 버튼
- `select` - Select 드롭다운
- `checkbox` - Checkbox 체크박스
- `dialog` - Dialog 모달
- `sheet` - Sheet 사이드 패널
- `table` - Table 테이블
- `tabs` - Tabs 탭
- `card` - Card 카드
- `badge` - Badge 뱃지
- `switch` - Switch 스위치

**위반 시:**
- 즉시 설치 필요
- 빌드 실패 원인이 됨
- 커밋 전 반드시 확인

---

## 🔄 작업 수행 시 필수 체크리스트 (CRITICAL)

**모든 구현 작업 완료 후 반드시 다음 순서대로 수행:**

### Step 1: SSOT 확인 ✅
```
□ 변경한 코드가 올바른 모듈의 책임인가?
□ 다른 모듈의 책임을 침범하지 않았는가?
□ 인터페이스를 통해서만 다른 모듈과 통신하는가?
□ 순환 참조가 없는가?
```

**위반 시 즉시 수정 필요!**

### Step 2: 문서 동기화 📝
```
□ 코드 변경 시 관련 설계 문서 업데이트 완료?
□ 새로운 함수/메서드 추가 시 docs/ 해당 모듈 문서에 반영?
□ API 변경 시 docs/api/ 문서 업데이트?
□ DB 스키마 변경 시 docs/database/ 문서 업데이트?
```

**업데이트할 문서:**
- 코드 변경: `docs/modules/{모듈명}.md`
- API 변경: `docs/api/*.md`
- DB 변경: `docs/database/schema.md`
- 아키텍처 변경: `docs/architecture/*.md`

### Step 3: Git 커밋 🔖
```
□ 변경사항을 논리적 단위로 그룹화했는가?
□ 커밋 메시지가 명확한가?
□ Co-Authored-By 태그 추가했는가?
```

**커밋 메시지 형식:**
```bash
{type}({scope}): {subject}

{body}

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Type:**
- `feat`: 새로운 기능
- `fix`: 버그 수정
- `refactor`: 리팩토링
- `docs`: 문서 변경
- `test`: 테스트 추가/수정
- `chore`: 빌드/설정 변경

**예시:**
```bash
git add {변경된 파일들}
git commit -m "$(cat <<'EOF'
feat(exit): Intent Reconciliation 기능 추가

- 중복 Intent 자동 탐지 및 취소 (30초 주기)
- position_id + reason_code 기반 중복 검사
- reconciliationLoop 백그라운드 실행

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## 🚨 체크리스트 위반 시 대응

**SSOT 위반:**
- 즉시 코드 수정
- 올바른 모듈로 이동
- 인터페이스 재설계

**문서 누락:**
- 작업 중단
- 문서 업데이트 우선 완료
- 코드 재검토

**Git 미처리:**
- 변경사항 커밋
- 논리적 단위로 분리
- 의미 있는 메시지 작성

---

## 📋 작업 완료 기준 (Definition of Done)

모든 작업은 다음 3가지를 모두 완료해야 "완료"로 간주:

1. ✅ **SSOT 준수** - 모듈 책임 경계 확인
2. 📝 **문서 동기화** - 관련 문서 업데이트 완료
3. 🔖 **Git 커밋** - 변경사항 커밋 완료

**하나라도 누락 시 작업 미완료!**

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

## Claude Code 활용 베스트 프랙티스

### 📝 1. 실수 학습 패턴 (Continuous Learning)

**원칙**: Claude가 실수할 때마다 CLAUDE.md에 지침을 추가하여 같은 실수를 반복하지 않게 합니다.

#### 실수 → 학습 프로세스

```
1. Claude가 실수 발생
   ↓
2. 실수 원인 분석
   ↓
3. CLAUDE.md에 명확한 지침 추가
   ↓
4. 다음 대화부터 자동 적용
```

#### 예시: 실수 사례와 개선

**실수 사례 1**: Exit Engine 구현 시 HardStop을 TODO 주석으로만 남김

**개선 지침 추가**:
```markdown
## ❌ 금지: TODO 주석으로 중요 기능 미루기

### 규칙
- 안전장치(HardStop, Circuit Breaker 등)는 TODO가 아닌 즉시 구현
- "// TODO: Implement later" 금지 → 구현하거나 별도 이슈 생성

### 예외
- 성능 최적화 (미래 개선)
- 선택적 기능 (현재 불필요)
```

**실수 사례 2**: Profile Resolver를 껍데기만 만들고 항상 default 반환

**개선 지침 추가**:
```markdown
## ❌ 금지: 인터페이스만 구현하고 내부는 빈 껍데기

### 규칙
- Repository를 주입받았으면 실제로 사용해야 함
- "For now, return default" 패턴 금지
- 미구현 시 명시적 에러 반환 또는 panic

### 체크리스트
- [ ] 주입된 dependency가 실제로 호출되는가?
- [ ] 오버라이드 로직이 실제로 작동하는가?
```

#### 지침 추가 위치

| 실수 유형 | CLAUDE.md 섹션 |
|----------|----------------|
| 설계 원칙 위반 | `## 절대 규칙` 또는 `## 모듈 독립성 설계 원칙` |
| 구현 패턴 오류 | `## 금지 패턴` (새 섹션 생성) |
| 문서 작성 오류 | `## 설계 문서 SSOT 규칙` |
| 테스트 누락 | `## 테스트 전략` (새 섹션 생성) |

---

### 🔍 2. PR 검증 활용 (Pull Request Review)

**원칙**: PR 과정에서 Claude를 태그해 리뷰 내용을 CLAUDE.md에 자동 반영합니다.

#### PR 워크플로우

```
1. 개발자가 PR 생성
   ↓
2. PR 설명에 @claude 태그
   ↓
3. Claude가 자동 코드 리뷰
   ↓
4. 발견된 문제를 CLAUDE.md에 지침으로 추가
   ↓
5. PR 승인 + CLAUDE.md 업데이트 커밋
```

#### PR 템플릿 예시

```markdown
## PR 설명
Exit Engine HardStop 구현

## 변경 사항
- [ ] HardStop 트리거 평가 메서드 추가
- [ ] PAUSE_ALL에서도 HardStop 예외 처리
- [ ] Runtime에 signalRepo 주입

## Claude 검증 요청
@claude 다음을 확인해주세요:
- [ ] HardStop이 PAUSE_ALL에서도 작동하는가?
- [ ] 테스트 커버리지가 충분한가?
- [ ] CLAUDE.md에 누락된 지침이 있는가?
```

#### Claude의 PR 리뷰 → CLAUDE.md 업데이트

Claude가 PR에서 발견한 패턴을 CLAUDE.md에 자동 추가:

```markdown
## 🔍 PR #123에서 발견된 패턴

### ✅ 좋은 패턴
- Intent 생성 후 Position.status를 CLOSING으로 전이
- 버전 체크 후 UpdateStatus 호출

### ⚠️ 개선 필요
- ExitSignalRepository 주입 시 nil 체크 누락
- GetAllOpenPositions 쿼리에 인덱스 힌트 부재

### 📋 CLAUDE.md 업데이트 필요
- [ ] "Repository 주입 시 nil 체크 필수" 규칙 추가
- [ ] "DB 쿼리 시 인덱스 활용 검증" 체크리스트 추가
```

---

### 🔗 3. GitHub 연동 (Issue-Driven Learning)

**원칙**: GitHub 이슈와 연동하여 문제 해결 과정을 자동으로 학습합니다.

#### 이슈 생성 → 해결 → 학습 사이클

```
1. 버그/기능 요청 이슈 생성
   ↓
2. 이슈에 @claude 태그
   ↓
3. Claude가 이슈 분석 + 해결 방안 제시
   ↓
4. 구현 후 PR 생성
   ↓
5. 해결 패턴을 CLAUDE.md에 기록
```

#### 이슈 템플릿

```markdown
## 버그 리포트
**제목**: Exit Engine이 PAUSE_ALL 상태에서 HardStop을 평가하지 않음

**재현 방법**:
1. Exit Control을 PAUSE_ALL로 설정
2. 급락장 발생 (-15%)
3. HardStop이 트리거되지 않음

**기대 동작**:
PAUSE_ALL에서도 HardStop은 예외로 작동해야 함

**실제 동작**:
모든 트리거가 차단됨 (HardStop 포함)

@claude 이 문제를 분석하고 CLAUDE.md에 방지 지침을 추가해주세요
```

#### Claude의 이슈 해결 → CLAUDE.md 반영

```markdown
## 🐛 Issue #456: PAUSE_ALL에서 HardStop 미작동

### 근본 원인
- `evaluateTriggers()`에서 PAUSE_ALL 체크가 HardStop 평가보다 먼저 실행됨

### 해결 방안
- HardStop을 우선순위 0번으로 이동 (PAUSE_ALL 체크보다 먼저)

### CLAUDE.md 업데이트
```markdown
## 안전장치 평가 우선순위 규칙

### 필수 원칙
- 최후 안전장치(HardStop, Emergency Flatten 등)는 **모든 제어 모드를 우회**
- 평가 순서: HardStop → Control Mode Check → 일반 트리거

### 코드 패턴
\`\`\`go
// ✅ CORRECT
func evaluateTriggers() {
    // Priority 0: HardStop (bypasses control mode)
    if profile.Config.HardStop.Enabled {
        if trigger := s.evaluateHardStop(...); trigger != nil {
            return trigger
        }
    }

    // Control Mode filtering
    if controlMode == PAUSE_ALL {
        return nil
    }

    // Regular triggers...
}
\`\`\`
```
```

---

### ⚙️ 4. 백그라운드 에이전트 활용 (Background Agents)

**원칙**: 오래 걸리는 작업은 백그라운드 에이전트에게 위임하고, 주 작업에 집중합니다.

#### 백그라운드 작업 적합 케이스

| 작업 유형 | 예시 | 백그라운드 권장 이유 |
|----------|------|---------------------|
| 대규모 테스트 실행 | 전체 유닛 테스트 (1000+ 케이스) | 5분+ 소요 |
| 코드 품질 검증 | Linter + Security Scan + Coverage | 반복적, 병렬 실행 가능 |
| 문서 생성 | API 문서 자동 생성 | 비동기 가능 |
| 데이터 마이그레이션 검증 | 1000만+ 레코드 검증 | 시간 소모적 |

#### 사용 방법

**예시 1: 전체 테스트 실행**

```bash
# 주 Claude: 기능 구현에 집중
# 백그라운드 Agent: 테스트 실행 + 리포트 생성

$ claude --background "go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html"
```

**예시 2: Exit Engine 검증**

```bash
# 주 Claude: 다음 기능 구현
# 백그라운드 Agent: Exit Engine 시뮬레이션 (1000 틱)

$ claude --background "./scripts/validate-exit-engine.sh --ticks=1000"
```

#### 백그라운드 결과 활용

```
1. 백그라운드 작업 완료 알림 수신
   ↓
2. 결과 분석 (통과/실패)
   ↓
3. 실패 시 → 이슈 생성 + CLAUDE.md 업데이트
   ↓
4. 통과 시 → PR 승인 진행
```

---

### 🔌 5. 플러그인 활용 (Plugins for Automation)

**원칙**: 반복 작업, 모니터링, 무한 루프 작업은 플러그인(Wilgan 등)으로 자동화합니다.

#### Wilgan Plugin 활용 예시

**사용 케이스 1: 지속적 모니터링**

```yaml
# .wilgan/monitor-exit-engine.yml
name: Exit Engine Health Monitor
trigger: every 5 minutes
actions:
  - check_service_status: runtime
  - query_db: "SELECT COUNT(*) FROM trade.positions WHERE status = 'CLOSING' AND updated_ts < NOW() - INTERVAL '10 minutes'"
  - alert_if: result > 0
    message: "⚠️ 10분 이상 CLOSING 상태인 포지션 발견"
```

**사용 케이스 2: 자동 문서 동기화**

```yaml
# .wilgan/sync-docs.yml
name: CLAUDE.md ↔ Implementation Sync Check
trigger: on every commit
actions:
  - grep_todos: "backend/**/*.go"
  - check_claude_md: TODO 주석이 CLAUDE.md에 금지 패턴으로 등록되었는가?
  - create_pr: CLAUDE.md 업데이트 필요 시 자동 PR 생성
```

**사용 케이스 3: 무한 반복 백테스팅**

```yaml
# .wilgan/backtest-loop.yml
name: Continuous Backtesting
trigger: on demand
actions:
  - loop:
      - run: go run ./cmd/backtest --date=$(date +%Y-%m-%d)
      - collect_metrics: PnL, Sharpe Ratio, Max Drawdown
      - update_dashboard: Grafana
      - sleep: 1 hour
    until: stopped
```

#### Wilgan Plugin 설정

```bash
# Wilgan 초기화
$ wilgan init

# Plugin 등록
$ wilgan add monitor-exit-engine.yml

# Plugin 실행
$ wilgan start monitor-exit-engine

# 상태 확인
$ wilgan status
```

---

### 🎯 통합 워크플로우 예시

#### 시나리오: Exit Engine 신규 트리거 추가

```
[Step 1] 이슈 생성
GitHub Issue: "EXIT_MOMENTUM 트리거 추가 요청"
@claude 요구사항 분석 + 설계 제안

[Step 2] 설계 단계
Claude: docs/modules/exit-engine.md 업데이트
백그라운드 Agent: 기존 Exit 로직 테스트 실행

[Step 3] 구현 단계
Claude: 코드 작성
백그라운드 Agent: Linter + Security Scan

[Step 4] PR 생성
PR 설명: @claude 코드 리뷰 + CLAUDE.md 업데이트 요청
Claude: 리뷰 후 CLAUDE.md에 신규 패턴 추가

[Step 5] 자동화
Wilgan Plugin: 매 1시간마다 EXIT_MOMENTUM 백테스팅
결과를 Slack으로 자동 알림
```

---

### 📋 체크리스트: Claude Code 효과적 활용

#### 실수 학습
- [ ] Claude 실수 발생 시 CLAUDE.md에 금지 패턴 추가
- [ ] 매 PR마다 CLAUDE.md 업데이트 검토
- [ ] 주 1회 CLAUDE.md 정리 (중복 제거, 우선순위 조정)

#### PR 검증
- [ ] PR 템플릿에 @claude 태그 포함
- [ ] PR 승인 전 CLAUDE.md 동기화 확인
- [ ] 리뷰 내용을 다음 작업에 반영

#### GitHub 연동
- [ ] 모든 버그 이슈에 @claude 태그
- [ ] 해결된 이슈의 패턴을 CLAUDE.md에 기록
- [ ] 이슈 템플릿 표준화

#### 백그라운드 작업
- [ ] 5분+ 소요 작업은 백그라운드로 위임
- [ ] 테스트는 항상 백그라운드에서 실행
- [ ] 백그라운드 결과를 PR에 자동 첨부

#### 플러그인 활용
- [ ] 반복 작업 식별 후 Wilgan으로 자동화
- [ ] 모니터링 플러그인 등록 (Exit Engine, API 헬스체크)
- [ ] 백테스팅 루프 자동화

---

### 🚀 효과

이 베스트 프랙티스를 따르면:

1. **학습 가속화**: 실수를 반복하지 않음
2. **품질 향상**: PR 단계에서 자동 검증
3. **생산성 증대**: 백그라운드 작업으로 병렬화
4. **자동화 극대화**: 플러그인으로 무인 운영

---

**Version**: v14.0.0-design
**Phase**: 설계 (Design)
**Last Updated**: 2026-01-15
