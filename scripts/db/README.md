# 데이터베이스 초기화 스크립트

> v14 데이터베이스 권한 문제를 방지하기 위한 완전한 초기화 스크립트

---

## 📋 스크립트 목록

| 파일 | 목적 | 실행 User | 실행 순서 |
|------|------|-----------|----------|
| `01_create_database.sql` | Database 및 Role 생성 | `postgres` | 1️⃣ |
| `02_create_schemas.sql` | Schema 생성 및 권한 설정 | `aegis_v14` | 2️⃣ |
| `03_check_permissions.sql` | 권한 확인 | `aegis_v14` | 검증 |
| `04_fix_permissions.sql` | 권한 문제 수정 | `aegis_v14` | 문제 발생 시 |
| `99_reset_all.sql` | 완전 초기화 (모든 데이터 삭제) | `postgres` | 긴급 시 |

---

## 🚀 빠른 시작

### Step 1: Database 및 Role 생성

```bash
psql -U postgres -f scripts/db/01_create_database.sql
```

**결과**:
- ✅ `aegis_v14` 데이터베이스 생성
- ✅ `aegis_v14` Role 생성 (기본 권한)
- ✅ `aegis_v14_readonly` Role 생성 (읽기 전용)

---

### Step 2: Schema 및 권한 설정

```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql
```

**결과**:
- ✅ `market`, `trade`, `system` Schema 생성
- ✅ 모든 권한 설정 (기존/향후 테이블)
- ✅ Default Privileges 설정

---

### Step 3: 권한 확인

```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql
```

**확인 사항**:
- Schema 권한 (CREATE, USAGE)
- 테이블 권한 (SELECT, INSERT, UPDATE, DELETE)
- Default Privileges
- Role 정보

---

## 🔧 문제 해결

### 권한 문제 발생 시

```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/04_fix_permissions.sql
```

**수정 내용**:
- 모든 기존 테이블에 권한 부여
- Sequence 권한 부여
- 테이블 소유권 변경
- Default Privileges 재설정

---

### 완전 초기화 (모든 데이터 삭제)

```bash
psql -U postgres -f scripts/db/99_reset_all.sql
```

⚠️ **경고**: 모든 데이터가 손실됩니다!

---

## 📝 자주 발생하는 권한 문제

### 1. "permission denied for table"

**원인**: 테이블에 대한 권한이 없음

**해결**:
```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/04_fix_permissions.sql
```

---

### 2. "must be owner of table"

**원인**: 테이블 소유권이 다른 Role에 있음

**해결**:
```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/04_fix_permissions.sql
```

---

### 3. "cannot create objects in schema"

**원인**: Schema에 대한 CREATE 권한이 없음

**확인**:
```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql
```

**해결**:
```sql
GRANT CREATE ON SCHEMA market TO aegis_v14;
```

---

## 🎯 개발 워크플로우

### 첫 설정 (최초 1회)

```bash
# 1. Database 생성
psql -U postgres -f scripts/db/01_create_database.sql

# 2. Schema 설정
psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql

# 3. 권한 확인
psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql
```

---

### 일일 개발 시작 전

```bash
# PostgreSQL 실행 확인
pg_isready -h localhost -p 5432 -U aegis_v14

# 연결 테스트
psql -U aegis_v14 -d aegis_v14 -c "SELECT current_database(), current_user;"
```

---

### 권한 문제 발생 시

```bash
# 1. 권한 확인
psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql

# 2. 권한 수정
psql -U aegis_v14 -d aegis_v14 -f scripts/db/04_fix_permissions.sql

# 3. 재확인
psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql
```

---

## 📚 참고 문서

- [데이터베이스 설정 가이드](../../docs/database/setup-guide.md)
- [데이터베이스 스키마](../../docs/database/schema.md)
- [접근 제어 설계](../../docs/database/access-control.md)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
