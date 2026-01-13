# v14 설계 문서 등록부

> 모든 설계 문서는 이곳에 등록되어야 합니다.

**Last Updated**: 2026-01-13

---

## 📋 문서 구조

```
docs/
├── _index.md              # 이 파일 (문서 등록부)
├── architecture/          # 시스템 아키텍처 설계
├── modules/               # 모듈별 설계
├── database/              # 데이터베이스 설계
├── api/                   # API 설계
└── ui/                    # UI 설계
```

---

## 🏗️ Architecture (시스템 아키텍처)

| 문서 | 상태 | 설명 |
|------|------|------|
| `architecture/system-overview.md` | ✅ 완료 | 전체 시스템 개요 (SSOT, 모듈 독립성, 멱등성) |
| `architecture/data-flow.md` | ⬜ TODO | 데이터 흐름 다이어그램 |
| `architecture/layer-design.md` | ⬜ TODO | 레이어 구조 설계 |
| `architecture/tech-stack.md` | ⬜ TODO | 기술 스택 선정 및 근거 |

---

## 🧩 Modules (모듈 설계)

### 핵심 모듈 (Quant Runtime)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| PriceSync | `modules/price-sync.md` | ✅ 완료 | 현재가 동기화 (KIS WS/REST, Naver) |
| Exit Engine | `modules/exit-engine.md` | ✅ 완료 | 자동 청산 (손절/익절/트레일링) |
| Reentry Engine | `modules/reentry-engine.md` | ✅ 완료 | 재진입 전략 (쿨다운/게이트/트리거) |
| Execution | `modules/execution-service.md` | ✅ 완료 | 주문 제출/체결 관리 (KIS API 연동) |

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
| `database/schema.md` | ✅ 완료 | 전체 테이블 스키마 정의 (market, trade schema) |
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

## 📊 설계 진행 현황

```
총 문서 수: 7/30 (계획 변경: Quant Runtime 중심)
진행률: 23%

✅ 완료: 7
  - architecture/system-overview.md
  - modules/price-sync.md
  - modules/exit-engine.md
  - modules/reentry-engine.md
  - modules/execution-service.md
  - modules/external-apis.md
  - database/schema.md

🚧 진행 중: 0
⬜ TODO: 23

핵심 Quant Runtime 완료 (PriceSync, Exit, Reentry, Execution) ✅
외부 API 연동 설계 완료 (KIS, Naver) ✅
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
