# Aegis v14

> 퀀트 트레이딩 시스템 설계 프로젝트

[![GitHub](https://img.shields.io/badge/GitHub-aegis--v14-blue?logo=github)](https://github.com/joungwonlim/aegis-v14)

---

## 📌 현재 단계: 설계 (Design Phase)

v14는 현재 **설계 단계**입니다. 코드 작성이 아닌 **문서 작성과 아키텍처 설계**에 집중하고 있습니다.

```
✍️ 설계 단계 (현재)
   → 구현 단계 (추후)
```

---

## 🏗️ Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 14+ (App Router) + shadcn/ui |
| Backend | Go 1.21+ (BFF) |
| Database | PostgreSQL 15+ |

---

## 📋 설계 문서 구조

```
docs/
├── _index.md              # 문서 등록부
├── architecture/          # 시스템 아키텍처 설계
│   ├── system-overview.md
│   ├── data-flow.md
│   ├── layer-design.md
│   └── tech-stack.md
├── modules/               # 모듈별 설계 (S0-S7)
│   ├── s0-data-quality.md
│   ├── s1-universe.md
│   ├── s2-signals.md
│   ├── s3-screener.md
│   ├── s4-ranking.md
│   ├── s5-portfolio.md
│   ├── s6-execution.md
│   └── s7-audit.md
├── database/              # 데이터베이스 설계
│   ├── erd.md
│   ├── schema.md
│   └── migration-plan.md
├── api/                   # API 설계
│   ├── stocks.md
│   ├── signals.md
│   └── common.md
└── ui/                    # UI/UX 설계
    ├── pages.md
    ├── components.md
    └── state-management.md
```

---

## 🎯 설계 원칙

### 1. 문서 우선 (Documentation First)
코드보다 설계 문서가 먼저. 문서 없이 구현 금지.

### 2. 모듈 독립성 (Module Independence)
각 모듈은 독립적으로 설계. 인터페이스로만 연결.

### 3. SSOT 준수 (Single Source of Truth)
정해진 위치에서만 해당 책임의 설계 정의.

### 4. 엄격한 검증 (Strict Validation)
모든 설계 문서는 체크리스트 통과 필수.

---

## 📊 7단계 파이프라인

```
S0: Data Quality  → 데이터 수집/검증
S1: Universe      → 투자 가능 종목
S2: Signals       → 팩터/이벤트 시그널
S3: Screener      → 1차 필터링 (Hard Cut)
S4: Ranking       → 종합 점수
S5: Portfolio     → 포트폴리오 구성
S6: Execution     → 주문 실행
S7: Audit         → 성과 분석
```

---

## 📝 설계 진행 현황

```
총 문서 수: 0/28
진행률: 0%

✅ 완료: 0
🚧 진행 중: 0
⬜ TODO: 28
```

상세 현황은 [docs/_index.md](docs/_index.md)를 참고하세요.

---

## 🚀 시작하기

### 문서 작성 규칙 확인

```bash
# v14 규칙 확인
cat CLAUDE.md

# 문서 등록부 확인
cat docs/_index.md
```

### 각 영역별 가이드

```bash
# 시스템 아키텍처
cat docs/architecture/README.md

# 모듈 설계
cat docs/modules/README.md

# 데이터베이스 설계
cat docs/database/README.md

# API 설계
cat docs/api/README.md

# UI 설계
cat docs/ui/README.md
```

---

## 📐 설계 템플릿

각 문서 종류별로 상세한 템플릿을 제공합니다:

- ✅ **모듈 설계 템플릿** - 인터페이스, 데이터 모델, 처리 흐름
- ✅ **DB 스키마 템플릿** - ERD, 테이블 정의, 인덱스 전략
- ✅ **API 설계 템플릿** - Request/Response, 에러 코드
- ✅ **UI 설계 가이드** - 컴포넌트 계층, 상태 관리

자세한 내용은 [CLAUDE.md](CLAUDE.md)를 참고하세요.

---

## 🔍 참고 프로젝트

- [Aegis v10](https://github.com/joungwonlim/aegis-v10) - 이전 버전 (참고용)
- [Aegis v13](https://github.com/joungwonlim/aegis-v13) - 이전 버전 (참고용)

---

## 📄 License

MIT License

---

## 📧 Contact

- GitHub: [@joungwonlim](https://github.com/joungwonlim)

---

**Version**: v14.0.0-design
**Phase**: 설계 (Design)
**Last Updated**: 2026-01-13
