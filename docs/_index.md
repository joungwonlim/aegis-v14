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
| `architecture/system-overview.md` | ⬜ TODO | 전체 시스템 개요 |
| `architecture/data-flow.md` | ⬜ TODO | 데이터 흐름 다이어그램 |
| `architecture/layer-design.md` | ⬜ TODO | 레이어 구조 설계 |
| `architecture/tech-stack.md` | ⬜ TODO | 기술 스택 선정 및 근거 |

---

## 🧩 Modules (모듈 설계)

| 모듈 | 문서 | 상태 | 설명 |
|------|------|------|------|
| S0 | `modules/s0-data-quality.md` | ⬜ TODO | 데이터 수집/검증 |
| S1 | `modules/s1-universe.md` | ⬜ TODO | 투자 가능 종목 선정 |
| S2 | `modules/s2-signals.md` | ⬜ TODO | 팩터/이벤트 시그널 |
| S3 | `modules/s3-screener.md` | ⬜ TODO | 1차 필터링 |
| S4 | `modules/s4-ranking.md` | ⬜ TODO | 종합 점수 산출 |
| S5 | `modules/s5-portfolio.md` | ⬜ TODO | 포트폴리오 구성 |
| S6 | `modules/s6-execution.md` | ⬜ TODO | 주문 실행 |
| S7 | `modules/s7-audit.md` | ⬜ TODO | 성과 분석 |
| External | `modules/external-apis.md` | ⬜ TODO | 외부 API 연동 (KIS, DART, Naver) |
| Brain | `modules/brain-orchestrator.md` | ⬜ TODO | 오케스트레이터 |

---

## 🗄️ Database (데이터베이스 설계)

| 문서 | 상태 | 설명 |
|------|------|------|
| `database/erd.md` | ⬜ TODO | ERD (Entity Relationship Diagram) |
| `database/schema.md` | ⬜ TODO | 전체 테이블 스키마 정의 |
| `database/indexes.md` | ⬜ TODO | 인덱스 전략 |
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
총 문서 수: 0/28
진행률: 0%

✅ 완료: 0
🚧 진행 중: 0
⬜ TODO: 28
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
