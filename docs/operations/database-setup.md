# 데이터베이스 설정 가이드

> **권한 문제 Zero 보장**: 이 가이드를 따르면 DB 권한 문제가 발생하지 않습니다.

**Last Updated**: 2026-01-14

---

## 🎯 핵심 원칙

### SSOT (Single Source of Truth)
```
.env 파일 = 모든 DB 접근의 유일한 진실 소스
```

**모든 DB 연결은 `.env`의 `DATABASE_URL`에서만 가져옵니다.**
- ✅ 코드: `config.Load()` → `.env` 읽기
- ✅ 스크립트: `DATABASE_URL` 환경 변수 사용
- ✅ Makefile: `.env`의 값 사용
- ❌ 하드코딩 금지

---

## 🚀 빠른 시작 (자동화)

### Step 1: 자동 초기화 스크립트 실행

```bash
cd backend
make db-init
```

**이 명령어가 자동으로 수행하는 작업**:
1. ✅ PostgreSQL 실행 확인
2. ✅ Database 및 Role 생성 (aegis_v14)
3. ✅ Schema 생성 (market, trade, system)
4. ✅ 모든 권한 자동 설정
5. ✅ 권한 검증
6. ✅ .env 파일 생성/확인
7. ✅ 최종 연결 테스트

**문제 발생 시 자동 수정**:
- 권한 문제 → 자동으로 `04_fix_permissions.sql` 실행
- .env 없음 → 자동으로 `.env.example` 복사

---

## 📋 수동 설정 (문제 해결용)

### Step 1: PostgreSQL 실행 확인

```bash
# PostgreSQL 실행 여부 확인
pg_isready -h localhost -p 5432

# 실행되지 않았다면
brew services start postgresql

# 또는
pg_ctl -D /usr/local/var/postgres start
```

---

### Step 2: Database 생성

```bash
# 방법 1: Makefile 사용 (권장)
make db-init-manual

# 방법 2: 직접 실행
psql -U postgres -f ../scripts/db/01_create_database.sql
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/02_create_schemas.sql
```

---

### Step 3: 권한 확인

```bash
# 권한 상태 확인
make db-check

# 또는
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/03_check_permissions.sql
```

**예상 출력**:
```
schema_name | owner      | can_create
------------|------------|------------
market      | aegis_v14  | t
trade       | aegis_v14  | t
system      | aegis_v14  | t
```

---

### Step 4: 권한 문제 수정 (필요시)

```bash
# 권한 문제 자동 수정
make db-fix

# 또는
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/04_fix_permissions.sql
```

---

## 🔒 SSOT 설정: .env 파일

### .env 파일 생성

```bash
# .env.example 복사
cp .env.example .env

# 필요시 값 수정
vi .env
```

### .env 파일 내용 (SSOT)

```bash
# Database (SSOT)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=aegis_v14
DB_USER=aegis_v14
DB_PASSWORD=aegis_v14_won

# 🔥 이것이 SSOT: 모든 연결은 이 URL 사용
DATABASE_URL=postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable
```

**중요**: `DATABASE_URL`만 수정하면 모든 연결이 자동으로 업데이트됩니다.

---

## 🧪 연결 테스트

### 1. psql로 직접 테스트

```bash
# .env의 DATABASE_URL 사용
source .env
psql $DATABASE_URL -c "SELECT 'Connection OK' as status;"
```

**예상 출력**:
```
   status
-------------
 Connection OK
```

---

### 2. Go 코드로 테스트

```bash
# 애플리케이션 실행
make run
```

**예상 로그**:
```json
{"level":"info","message":"Connecting to PostgreSQL...","host":"localhost","port":"5432"}
{"level":"info","message":"✅ PostgreSQL connected successfully"}
{"level":"info","message":"✅ Database connection OK"}
```

---

## 🚨 문제 해결

### 문제 1: "permission denied for schema"

**증상**:
```
ERROR: permission denied for schema market
```

**해결**:
```bash
make db-fix
```

또는

```bash
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/04_fix_permissions.sql
```

---

### 문제 2: "database does not exist"

**증상**:
```
FATAL: database "aegis_v14" does not exist
```

**해결**:
```bash
psql -U postgres -f ../scripts/db/01_create_database.sql
```

---

### 문제 3: "role does not exist"

**증상**:
```
FATAL: role "aegis_v14" does not exist
```

**해결**:
```bash
psql -U postgres -f ../scripts/db/01_create_database.sql
```

---

### 문제 4: "connection refused"

**증상**:
```
could not connect to server: Connection refused
```

**해결**:
```bash
# PostgreSQL 실행
brew services start postgresql

# 또는
pg_ctl -D /usr/local/var/postgres start
```

---

### 문제 5: ".env file not found"

**증상**:
```
Warning: .env file not found
```

**해결**:
```bash
cp .env.example .env
```

---

## 🔍 권한 확인 체크리스트

애플리케이션 시작 전 확인:

- [ ] PostgreSQL 실행 중 (`pg_isready`)
- [ ] aegis_v14 database 존재
- [ ] aegis_v14 role 존재
- [ ] market, trade, system schema 존재
- [ ] aegis_v14 role이 모든 schema에 권한 보유
- [ ] .env 파일 존재
- [ ] DATABASE_URL이 올바르게 설정됨

**모두 확인**:
```bash
./scripts/init-dev.sh
```

---

## 📝 개발 워크플로우

### 매일 개발 시작 시

```bash
# 1. PostgreSQL 실행 확인
pg_isready || brew services start postgresql

# 2. 애플리케이션 실행
make run
```

**DB가 초기화되지 않았다면**:
```bash
make db-init
```

### CI/CD 환경

```bash
# 1. .env 설정
export DATABASE_URL="postgresql://..."

# 2. DB 초기화
./scripts/init-dev.sh

# 3. 마이그레이션
make migrate-up

# 4. 테스트
make test
```

---

## 🔐 프로덕션 환경 주의사항

### 절대 하지 말 것

- ❌ .env 파일을 Git에 커밋
- ❌ 프로덕션 DB URL을 로컬 .env에 저장
- ❌ aegis_v14 role에 SUPERUSER 권한 부여
- ❌ 프로덕션에서 db-init 스크립트 실행

### 권장 사항

- ✅ 환경 변수로 DATABASE_URL 주입
- ✅ Secrets Management 사용 (AWS Secrets Manager, etc.)
- ✅ 읽기 전용 replica 사용
- ✅ Connection Pool 크기 조정 (max_conns: 50+)

---

## 🔍 참고 문서

- [데이터베이스 스키마](../database/schema.md)
- [데이터베이스 접근 제어](../database/access-control.md)
- [DB 초기화 스크립트](../../scripts/db/)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
