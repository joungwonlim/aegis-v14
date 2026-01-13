# v14 설계 문서 등록부

> 모든 설계 문서는 이곳에 등록되어야 합니다.

**Last Updated**: 2026-01-13

---

## 📋 문서 구조

```
docs/
├── _index.md                    # 이 파일 (문서 등록부)
├── architecture/                # 시스템 아키텍처 설계
│   ├── system-overview.md       # 전체 시스템 개요
│   ├── pick-to-execution-pipeline.md
│   └── architecture-improvements.md  # 성능/안정성 개선안
├── modules/                     # 모듈별 설계
├── database/                    # 데이터베이스 설계
├── api/                         # API 설계
├── ui/                          # UI 설계
└── reviews/                     # 설계 검토 기록 (아카이브)
    └── 2026-01-13-ssot-review.md
```

---

## 🏗️ Architecture (시스템 아키텍처)

| 문서 | 상태 | 설명 |
|------|------|------|
| `architecture/system-overview.md` | ✅ 완료 | 전체 시스템 개요 (SSOT, 모듈 독립성, 멱등성) |
| `architecture/pick-to-execution-pipeline.md` | ✅ 완료 | 다중 선정 모듈 → 단일 실행 시스템 파이프라인 |
| `architecture/architecture-improvements.md` | ✅ 완료 | 성능 및 안정성 개선안 (P0~P2 우선순위, Redis 읽기 가속 - SSOT 원칙 준수) |
| `architecture/data-flow.md` | ⬜ TODO | 데이터 흐름 다이어그램 |
| `architecture/layer-design.md` | ⬜ TODO | 레이어 구조 설계 |
| `architecture/tech-stack.md` | ⬜ TODO | 기술 스택 선정 및 근거 |

---

## 🧩 Modules (모듈 설계)

### 핵심 모듈 (Quant Runtime)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| PriceSync | `modules/price-sync.md` | ✅ 완료 | 현재가 동기화 (KIS WS/REST, Naver) |
| Exit Engine | `modules/exit-engine.md` | ✅ 완료 | 자동 청산 (Hybrid % + ATR 표준, Control Gate, Profile System) |
| Reentry Engine | `modules/reentry-engine.md` | ✅ 완료 | 재진입 전략 (ExitEvent 기반, Control Gate) |
| Execution | `modules/execution-service.md` | ✅ 완료 | 주문 제출/체결 관리 (ExitEvent 생성 SSOT) |

### 전략 모듈 (향후 확장)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| Universe | `modules/universe.md` | ⬜ TODO | 투자 가능 종목 선정 |
| Signals | `modules/signals.md` | ⬜ TODO | 팩터/이벤트 시그널 |
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
| `database/schema.md` | ✅ 완료 | 전체 테이블 스키마 정의 (market, trade, system schema) |
| `database/access-control.md` | ✅ 완료 | PostgreSQL RBAC 설계 (role 기반 접근 제어) |
| `database/erd.md` | ⬜ TODO | ERD 상세 (schema.md에 포함되어 있음) |
| `database/indexes.md` | ⬜ TODO | 인덱스 최적화 전략 (schema.md에 일부 포함) |
| `database/migration-plan.md` | ⬜ TODO | 마이그레이션 계획 |

---

## 🌐 API (API 설계)

| 엔드포인트 | 문서 | 상태 | 설명 |
|------------|------|------|------|
| Stocks | `api/stocks.md` | ⬜ TODO | 종목 조회/관리 |
| Signals | `api/signals.md` | ⬜ TODO | 시그널 조회 |
| Portfolio | `api/portfolio.md` | ⬜ TODO | 포트폴리오 조회/관리 |
| Orders | `api/orders.md` | ⬜ TODO | 주문 조회/실행 |
| Performance | `api/performance.md` | ⬜ TODO | 성과 분석 조회 |
| Common | `api/common.md` | ⬜ TODO | 공통 스펙 (인증, 에러 코드 등) |

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

## 📊 설계 진행 현황

```
총 문서 수: 11/30 (계획 변경: Quant Runtime + Pick Pipeline)
진행률: 37%

✅ 완료: 11
  - architecture/system-overview.md (Router SSOT 추가)
  - architecture/pick-to-execution-pipeline.md
  - architecture/architecture-improvements.md (성능/안정성 개선안 P0~P2)
  - modules/price-sync.md
  - modules/exit-engine.md (Control Gate + Profile System, SSOT 강화)
  - modules/reentry-engine.md (ExitEvent 기반 디커플링)
  - modules/execution-service.md (ExitEvent 생성 SSOT)
  - modules/external-apis.md (KIS WS TR별 소유권 분리)
  - database/schema.md (21 tables, positions 컬럼별 SSOT 명시)
  - database/access-control.md (컬럼별 권한, DELETE 제거)
  - reviews/2026-01-13-ssot-review.md (SSOT 검증 아카이브)

🚧 진행 중: 0
⬜ TODO: 20

핵심 Quant Runtime 완료 (PriceSync, Exit, Reentry, Execution) ✅
외부 API 연동 설계 완료 (KIS, Naver) ✅
데이터베이스 접근 제어 설계 완료 (PostgreSQL RBAC, 컬럼별 권한) ✅
v10 운영 이슈 해결 설계 완료 (중복 실행, 평단가 변경, Price Sync 장애) ✅
Pick-to-Execution Pipeline 설계 완료 (다중 선정 → 단일 실행) ✅
Exit/Reentry 제어 시스템 완료 (Kill Switch, Profile, Symbol Override) ✅
Exit/Reentry 디커플링 완료 (ExitEvent SSOT 기반 아키텍처) ✅
Exit 표준 룰 완료 (Hybrid % + ATR, 프로파일 3종, HardStop) ✅
SSOT 검증 및 수정 완료 (문서 간 불일치 5건 해결) ✅
아키텍처 개선안 작성 완료 (P0~P2 우선순위별 6건) ✅
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
