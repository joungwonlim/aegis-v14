# Architecture (시스템 아키텍처 설계)

이 폴더는 v14 시스템의 전체 아키텍처 설계 문서를 포함합니다.

---

## 📋 문서 목록

### 1. system-overview.md
- **목적**: 전체 시스템 개요
- **내용**:
  - 시스템 목표
  - 핵심 기능
  - 전체 구조도
  - 기술적 제약사항

### 2. data-flow.md
- **목적**: 데이터 흐름 다이어그램
- **내용**:
  - 데이터 소스 (KRX, DART, Naver, KIS)
  - 데이터 파이프라인 (S0 → S7)
  - 데이터 저장소
  - 데이터 변환 과정

### 3. layer-design.md
- **목적**: 레이어 구조 설계
- **내용**:
  - Frontend 레이어
  - Backend 레이어
  - Database 레이어
  - 레이어 간 통신 방식

### 4. tech-stack.md
- **목적**: 기술 스택 선정 및 근거
- **내용**:
  - Frontend: Next.js + shadcn/ui
  - Backend: Go (BFF)
  - Database: PostgreSQL
  - 선정 이유 및 장단점

---

## 🎯 작성 순서 (권장)

1. **system-overview.md** (가장 먼저)
   - 전체 그림을 먼저 그림

2. **data-flow.md**
   - 데이터가 어떻게 흐르는지 정의

3. **layer-design.md**
   - 레이어별 책임 정의

4. **tech-stack.md**
   - 기술 선택 근거 정리

---

## 📐 다이어그램 작성 가이드

### Mermaid 사용 (권장)

```markdown
\`\`\`mermaid
graph TD
    A[Data Source] --> B[S0: Data Quality]
    B --> C[S1: Universe]
    C --> D[S2: Signals]
\`\`\`
```

### ASCII Art (간단한 경우)

```
┌─────────────┐
│ Data Source │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ S0: Data    │
└──────┬──────┘
```

---

## ✅ 완료 기준

각 문서가 다음을 충족해야 함:

- [ ] 목적과 범위 명확
- [ ] 다이어그램 포함
- [ ] 구체적이고 모호하지 않음
- [ ] 다른 아키텍처 문서와 일관성
- [ ] 구현 가능한 수준

---

## 🔗 참고

- [CLAUDE.md](../../CLAUDE.md) - 설계 템플릿
- [_index.md](../_index.md) - 문서 등록부
- v10/v13 아키텍처 문서 (참고용)
