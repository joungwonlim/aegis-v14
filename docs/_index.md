# v14 설계 문서 등록부

> 모든 설계 문서는 이곳에 등록되어야 합니다.

**Last Updated**: 2026-01-14

---

## 📋 문서 구조

```
docs/
├── _index.md                    # 이 파일 (문서 등록부)
├── architecture/                # 시스템 아키텍처 설계
│   ├── system-overview.md       # 전체 시스템 개요
│   ├── pick-to-execution-pipeline.md
│   ├── architecture-improvements.md  # 성능/안정성 개선안
│   └── module-dependencies.md   # 모듈 의존성 맵 ✨ NEW
├── modules/                     # 모듈별 설계
│   └── module-catalog.md        # 모듈 카탈로그 (독립 작업 체계) ✨ NEW
├── database/                    # 데이터베이스 설계
│   └── setup-guide.md           # DB 초기화 및 권한 설정 가이드 ✨ NEW
├── api/                         # API 설계
├── ui/                          # UI 설계
├── operations/                  # 운영 가이드
│   └── exit-engine-playbook.md  # Exit Engine 운영 플레이북
└── reviews/                     # 설계 검토 기록 (아카이브)
    └── 2026-01-13-ssot-review.md

scripts/                         # 실행 스크립트 ✨ NEW
└── db/                          # DB 초기화 스크립트
    ├── 01_create_database.sql
    ├── 02_create_schemas.sql
    ├── 03_check_permissions.sql
    ├── 04_fix_permissions.sql
    └── 99_reset_all.sql
```

---

## 🏗️ Architecture (시스템 아키텍처)

| 문서 | 상태 | 설명 |
|------|------|------|
| `architecture/system-overview.md` | ✅ 완료 | 전체 시스템 개요 (SSOT, 모듈 독립성, 멱등성) |
| `architecture/pick-to-execution-pipeline.md` | ✅ 완료 | 다중 선정 모듈 → 단일 실행 시스템 파이프라인 |
| `architecture/architecture-improvements.md` | ✅ 완료 | 성능 및 안정성 개선안 (P0~P2 우선순위, Redis 읽기 가속 - SSOT 원칙 준수) |
| `architecture/module-dependencies.md` | ✅ 완료 | 모듈 의존성 맵 (레이어 구조, 의존성 방향, 순환 참조 방지) |
| `architecture/data-flow.md` | ✅ 완료 | 데이터 흐름 다이어그램 (SSOT, Cache-Aside, 이벤트 기반) |
| `architecture/layer-design.md` | ✅ 완료 | 레이어 구조 설계 (Go 프로젝트 구조, 5-Layer Architecture) |
| `architecture/tech-stack.md` | ✅ 완료 | 기술 스택 선정 및 근거 (Go, PostgreSQL, Next.js) |

---

## 🧩 Modules (모듈 설계)

| 문서 | 상태 | 설명 |
|------|------|------|
| `modules/module-catalog.md` | ✅ 완료 | 모듈 카탈로그 (독립 작업 체계, 14개 모듈 등록, 개발 준비도 추적) |
| `modules/development-guide.md` | ⬜ TODO | 모듈별 개발 가이드 (독립 개발 환경, Mock/Stub 전략) |

### 핵심 모듈 (Quant Runtime)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| PriceSync | `modules/price-sync.md` | ✅ 완료 | 현재가 동기화 (KIS WS/REST, Naver) |
| Exit Engine | `modules/exit-engine.md` | ✅ 완료 | 자동 청산 (Hybrid % + ATR 표준, Control Gate, Profile System, **v10 사고 사례 추가**) |
| Reentry Engine | `modules/reentry-engine.md` | ✅ 완료 | 재진입 전략 (ExitEvent 기반, Control Gate) |
| Execution | `modules/execution-service.md` | ✅ 완료 | 주문 제출/체결 관리 (ExitEvent 생성 SSOT) |

### 전략 모듈 (향후 확장)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| Universe | `modules/universe.md` | ✅ 완료 | 투자 가능 종목 선정 (Tier 구조, 필터링 기준, Snapshot) |
| Signals | `modules/signals.md` | ✅ 완료 | 팩터 기반 매매 신호 (Momentum, Quality, Value, Technical) |
| Ranking | `modules/ranking.md` | ⬜ TODO | 종합 점수 산출 |
| Portfolio | `modules/portfolio.md` | ⬜ TODO | 포트폴리오 구성 |
| Risk | `modules/risk-management.md` | ⬜ TODO | 리스크 관리 |

### 인프라 모듈

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| External APIs | `modules/external-apis.md` | ✅ 완료 | 외부 API 연동 (KIS WS/REST, Naver) |
| Monitoring | `modules/monitoring.md` | ⬜ TODO | 모니터링/알람 |

---

## 🗄️ Database (데이터베이스 설계)

| 문서 | 상태 | 설명 |
|------|------|------|
| `database/schema.md` | ✅ 완료 | 전체 테이블 스키마 정의 (market, trade, system schema, **stocks 추가**) |
| `database/access-control.md` | ✅ 완료 | PostgreSQL RBAC 설계 (role 기반 접근 제어) |
| `database/migration-stocks.md` | ✅ 완료 | market.stocks 테이블 마이그레이션 계획 (Phase 1~5, FK 제약조건) |
| `database/setup-guide.md` | ✅ 완료 | DB 초기화 및 권한 설정 가이드 (권한 문제 방지, 트러블슈팅) |
| `database/erd.md` | ⬜ TODO | ERD 상세 (schema.md에 포함되어 있음) |
| `database/indexes.md` | ⬜ TODO | 인덱스 최적화 전략 (schema.md에 일부 포함) |

---

## 🌐 API (API 설계)

| 엔드포인트 | 문서 | 상태 | 설명 |
|------------|------|------|------|
| Common | `api/common.md` | ✅ 완료 | 공통 스펙 (응답 구조, 에러 코드, Pagination, CORS) |
| Health Check | `api/health.md` | ✅ 완료 | Health Check API (liveness, readiness, detailed) |
| Stocks | `api/stocks.md` | ✅ 완료 | 종목 조회/관리 (목록, 상세, 필터링, 검색) |
| Prices | `api/prices.md` | ✅ 완료 | 가격 조회 (Best Price, Batch, Freshness) |
| Signals | `api/signals.md` | ⬜ TODO | 시그널 조회 |
| Portfolio | `api/portfolio.md` | ⬜ TODO | 포트폴리오 조회/관리 |
| Orders | `api/orders.md` | ⬜ TODO | 주문 조회/실행 |
| Performance | `api/performance.md` | ⬜ TODO | 성과 분석 조회 |

---

## 🎨 UI (UI 설계)

| 문서 | 상태 | 설명 |
|------|------|------|
| `ui/pages.md` | ⬜ TODO | 페이지 구조 |
| `ui/components.md` | ⬜ TODO | 컴포넌트 계층 |
| `ui/state-management.md` | ⬜ TODO | 상태 관리 전략 |
| `ui/api-integration.md` | ⬜ TODO | API 연동 방안 |

---

## 📝 설계 검토 (Reviews)

| 문서 | 상태 | 설명 |
|------|------|------|
| `reviews/2026-01-13-ssot-review.md` | ✅ 완료 | SSOT 불일치 검증 및 수정 완료 (아카이브) |

---

## 🎮 운영 가이드 (Operations)

| 문서 | 상태 | 설명 |
|------|------|------|
| `operations/exit-engine-playbook.md` | ✅ 완료 | Exit Engine 운영 플레이북 (If-Then 시나리오, 긴급 대응, 모니터링) |
| `operations/database-setup.md` | ✅ 완료 | 데이터베이스 설정 가이드 (SSOT, 권한 문제 Zero 보장, 자동 초기화) |
| `operations/logging-strategy.md` | ✅ 완료 | 로깅 전략 (구조화된 로깅, Request ID, 파일 rotation, 디버깅 가이드) |

---

## 📊 설계 진행 현황

```
총 문서 수: 27/39 (Signals 설계 추가)
진행률: 69%

✅ 완료: 27
  - architecture/system-overview.md (Router SSOT 추가)
  - architecture/pick-to-execution-pipeline.md
  - architecture/architecture-improvements.md (성능/안정성 개선안 P0~P2)
  - architecture/module-dependencies.md (모듈 의존성 맵, 레이어 구조)
  - architecture/data-flow.md (데이터 흐름, SSOT, Cache-Aside)
  - architecture/layer-design.md (Go 프로젝트 구조, 5-Layer)
  - architecture/tech-stack.md (Go/PostgreSQL/Next.js 선정 근거)
  - modules/module-catalog.md (모듈 카탈로그, 14개 모듈 등록)
  - modules/price-sync.md
  - modules/exit-engine.md (Control Gate + Profile System, SSOT 강화, v10 사고 사례)
  - modules/reentry-engine.md (ExitEvent 기반 디커플링)
  - modules/execution-service.md (ExitEvent 생성 SSOT)
  - modules/external-apis.md (KIS WS TR별 소유권 분리)
  - modules/universe.md (투자 가능 종목 선정, Tier 구조, 필터링 기준, Snapshot)
  - modules/signals.md (팩터 기반 매매 신호, Momentum/Quality/Value/Technical) ⭐ NEW
  - database/schema.md (22 tables, market.stocks 추가, 컬럼별 SSOT 명시)
  - database/access-control.md (컬럼별 권한, DELETE 제거)
  - database/migration-stocks.md (stocks 테이블 마이그레이션 Phase 1~5, FK 제약조건)
  - database/setup-guide.md (DB 초기화 및 권한 설정, 권한 문제 방지)
  - api/common.md (API 공통 스펙, 응답 구조, 에러 코드, Pagination)
  - api/health.md (Health Check API, liveness/readiness/detailed)
  - api/stocks.md (Stocks API, 목록/상세/필터링/검색) ⭐ NEW
  - operations/exit-engine-playbook.md (If-Then 시나리오, 긴급 대응, 모니터링)
  - operations/database-setup.md (SSOT 기반 DB 설정, 권한 문제 Zero 보장)
  - operations/logging-strategy.md (구조화된 로깅, Request ID, 디버깅)
  - reviews/2026-01-13-ssot-review.md (SSOT 검증 아카이브)
  - scripts/db/ (DB 초기화 스크립트 6개)

🚧 진행 중: 0
⬜ TODO: 11

핵심 Quant Runtime 완료 (PriceSync, Exit, Reentry, Execution) ✅
외부 API 연동 설계 완료 (KIS, Naver) ✅
데이터베이스 접근 제어 설계 완료 (PostgreSQL RBAC, 컬럼별 권한) ✅
종목 마스터 SSOT 설계 완료 (market.stocks, FK 제약조건, 마이그레이션 계획) ✅
v10 운영 이슈 해결 설계 완료 (중복 실행, 평단가 변경, Price Sync 장애) ✅
Pick-to-Execution Pipeline 설계 완료 (다중 선정 → 단일 실행) ✅
Exit/Reentry 제어 시스템 완료 (Kill Switch, Profile, Symbol Override) ✅
Exit/Reentry 디커플링 완료 (ExitEvent SSOT 기반 아키텍처) ✅
Exit 표준 룰 완료 (Hybrid % + ATR, 프로파일 3종, HardStop) ✅
SSOT 검증 및 수정 완료 (문서 간 불일치 5건 해결) ✅
아키텍처 개선안 작성 완료 (P0~P2 우선순위별 6건) ✅
Exit Engine 운영 플레이북 작성 완료 (If-Then, 긴급 대응, 조정 우선순위) ✅
모듈 독립 작업 체계 완료 (모듈 카탈로그, 의존성 맵, DB 권한 문제 해결) ✅
Architecture 설계 완성 (데이터 흐름, 레이어 구조, 기술 스택) ✅
```

---

## 📝 문서 추가 방법

1. 해당 카테고리 폴더에 문서 생성
2. 이 파일 (`_index.md`)에 등록
3. 상태를 ⬜ TODO → 🚧 진행 중 → ✅ 완료로 업데이트
4. Git 커밋: `docs(scope): 문서명 추가`

---

## 🔍 문서 검색 팁

```bash
# 특정 키워드로 문서 검색
grep -r "keyword" docs/

# 문서 목록 확인
find docs/ -name "*.md" | sort

# 미완성 문서 찾기
grep "⬜ TODO" docs/_index.md
```

---

## ⚠️ 주의사항

1. **모든 새 문서는 반드시 이 파일에 등록**
2. **문서 작성 전 중복 확인**
3. **템플릿 사용 (CLAUDE.md 참고)**
4. **다른 문서와의 일관성 유지**
