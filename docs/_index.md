# Aegis v14 문서 인덱스

> v14 설계 및 구현 문서 중앙 인덱스

---

## 📁 문서 구조

```
docs/
├── _index.md                 (이 파일)
├── modules/                  모듈별 설계 문서
│   └── price-sync.md        ✅ PriceSync (v14.1.0 구현 완료)
├── architecture/             시스템 아키텍처
├── database/                 DB 스키마/ERD
├── api/                      API 계약
├── ui/                       UI 설계
└── templates/                문서 템플릿
```

---

## 📝 모듈 문서

### ✅ 구현 완료

| 문서 | 상태 | 버전 | 최종 업데이트 |
|------|------|------|--------------|
| [price-sync.md](./modules/price-sync.md) | ✅ Production Ready | v14.1.0 | 2026-01-18 |

**PriceSync 주요 구현 내용**:
- Portfolio 우선순위 가격 동기화 (WS 40개 제한 전용 할당)
- 3-Tier REST 시스템 (Tier0=3초, Tier1=10초, Tier2=30초)
- PriorityManager 자동 우선순위 계산
- Exit Engine 연동 완료

---

## 🚧 설계 중

(아직 없음)

---

## 📋 작성 규칙

### 문서 추가 시

1. **적절한 디렉토리에 문서 생성**
   - 모듈 설계: `docs/modules/<module-name>.md`
   - 아키텍처: `docs/architecture/<topic>.md`
   - DB 설계: `docs/database/<topic>.md`

2. **이 인덱스 파일 업데이트**
   - 해당 섹션에 문서 링크 추가
   - 상태/버전/날짜 기록

3. **Git 커밋**
   ```bash
   git add docs/
   git commit -m "docs(module): <설명>"
   ```

### 문서 상태 표시

| 상태 | 의미 |
|------|------|
| 🚧 설계 중 | 초안 작성 중 |
| ✅ 설계 완료 | 설계 문서 완성 (구현 전) |
| ✅ 구현 완료 | 설계 + 구현 완료 |
| ✅ Production Ready | 운영 검증 완료 |

---

## 🔗 관련 자료

- [CLAUDE.md](../CLAUDE.md) - Claude Code 작업 규칙
- v10: `/Users/wonny/Dev/aegis/v10/`
- v13: `/Users/wonny/Dev/aegis/v13/`

---

**Last Updated**: 2026-01-18
**Maintained by**: Aegis v14 Team
