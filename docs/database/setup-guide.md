# 데이터베이스 초기화 및 권한 설정 가이드

> 개발 환경에서 DB 권한 문제를 방지하기 위한 완전한 설정 가이드

**Last Updated**: 2026-01-14

---

## 🎯 목적

이 문서는 **개발 중 발생하는 DB 권한 문제를 사전에 방지**하기 위해 작성되었습니다.

### 해결하는 문제
- ❌ "permission denied for table" 에러
- ❌ "must be owner of table" 에러
- ❌ "cannot create objects in schema" 에러
- ❌ Role 권한 불일치로 인한 개발 중단

---

## 📋 전제 조건

### 1. PostgreSQL 설치 확인
```bash
psql --version
# PostgreSQL 15.x 이상 권장
```

### 2. 환경 변수 확인
`.env` 파일에 다음 내용이 있어야 합니다:

```bash
# .env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=aegis_v14
DB_USER=aegis_v14
DB_PASSWORD=aegis_v14_won
DATABASE_URL=postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14
```

**주의**: `DB_NAME`이 `aegis_v144`로 오타가 있다면 `aegis_v14`로 수정하세요.

---

## 🚀 Step 1: 데이터베이스 및 Role 생성

### 1.1 PostgreSQL 접속 (Superuser)

```bash
# macOS (Homebrew 설치 시)
psql postgres

# Linux
sudo -u postgres psql
```

### 1.2 Database 및 Role 생성 스크립트

**파일**: `scripts/db/01_create_database.sql`

```sql
-- =====================================================
-- v14 Database 및 Role 생성 스크립트
-- =====================================================

-- 1. Database 존재 확인 및 생성
DROP DATABASE IF EXISTS aegis_v14;
CREATE DATABASE aegis_v14
    WITH
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE aegis_v14 IS 'Aegis v14 Quant Trading System';

-- 2. 기본 Role 생성
DROP ROLE IF EXISTS aegis_v14;
CREATE ROLE aegis_v14 WITH
    LOGIN
    PASSWORD 'aegis_v14_won'
    CREATEDB           -- 로컬 개발용: 테스트 DB 생성 권한
    NOSUPERUSER
    NOCREATEROLE
    NOREPLICATION;

COMMENT ON ROLE aegis_v14 IS 'v14 Application Default User';

-- 3. 읽기 전용 Role (분석/모니터링용)
DROP ROLE IF EXISTS aegis_v14_readonly;
CREATE ROLE aegis_v14_readonly WITH
    LOGIN
    PASSWORD 'aegis_v14_readonly'
    NOSUPERUSER
    NOCREATEROLE
    NOCREATEDB
    NOREPLICATION;

COMMENT ON ROLE aegis_v14_readonly IS 'v14 Read-Only User for Analytics';

-- 4. Database 소유권 변경
ALTER DATABASE aegis_v14 OWNER TO aegis_v14;

\echo 'Database and Roles created successfully!'
\echo 'Next: Connect to aegis_v14 and run 02_create_schemas.sql'
```

### 1.3 실행

```bash
# Superuser로 실행
psql -U postgres -f scripts/db/01_create_database.sql

# 또는 직접 실행
psql postgres < scripts/db/01_create_database.sql
```

---

## 🏗️ Step 2: Schema 및 권한 설정

### 2.1 aegis_v14 데이터베이스에 접속

```bash
psql -U aegis_v14 -d aegis_v14
```

### 2.2 Schema 생성 스크립트

**파일**: `scripts/db/02_create_schemas.sql`

```sql
-- =====================================================
-- v14 Schema 생성 및 권한 설정
-- =====================================================

-- 1. Schema 생성
CREATE SCHEMA IF NOT EXISTS market;
CREATE SCHEMA IF NOT EXISTS trade;
CREATE SCHEMA IF NOT EXISTS system;

-- Schema 소유권 변경
ALTER SCHEMA market OWNER TO aegis_v14;
ALTER SCHEMA trade OWNER TO aegis_v14;
ALTER SCHEMA system OWNER TO aegis_v14;

-- Schema 설명
COMMENT ON SCHEMA market IS '시장 데이터 (종목, 가격, 재무 등)';
COMMENT ON SCHEMA trade IS '거래 데이터 (포지션, 주문, 체결 등)';
COMMENT ON SCHEMA system IS '시스템 설정 및 메타데이터';

-- 2. aegis_v14 Role에 모든 권한 부여 (개발용)
GRANT ALL PRIVILEGES ON SCHEMA market TO aegis_v14;
GRANT ALL PRIVILEGES ON SCHEMA trade TO aegis_v14;
GRANT ALL PRIVILEGES ON SCHEMA system TO aegis_v14;

-- 3. 향후 생성될 테이블에도 자동으로 권한 부여 (중요!)
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;

-- 4. Sequence 권한 (AUTO INCREMENT용)
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;

-- 5. 읽기 전용 Role 권한
GRANT USAGE ON SCHEMA market TO aegis_v14_readonly;
GRANT USAGE ON SCHEMA trade TO aegis_v14_readonly;
GRANT USAGE ON SCHEMA system TO aegis_v14_readonly;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT SELECT ON TABLES TO aegis_v14_readonly;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_v14_readonly;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT SELECT ON TABLES TO aegis_v14_readonly;

-- 6. Search Path 설정 (기본 스키마 순서)
ALTER ROLE aegis_v14 SET search_path TO trade, market, system, public;
ALTER ROLE aegis_v14_readonly SET search_path TO trade, market, system, public;

\echo 'Schemas and permissions configured successfully!'
\echo 'Next: Run migrations to create tables'

-- 7. 권한 확인
SELECT
    nspname AS schema_name,
    nspowner::regrole AS owner
FROM pg_namespace
WHERE nspname IN ('market', 'trade', 'system');
```

### 2.3 실행

```bash
psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql
```

---

## 📦 Step 3: 테이블 생성 (Migration)

### 3.1 마이그레이션 도구 설정

**golang-migrate 사용 권장**

```bash
# 설치 (macOS)
brew install golang-migrate

# 설치 확인
migrate -version
```

### 3.2 마이그레이션 파일 생성

**파일 구조**:
```
backend/migrations/
├── 000001_create_stocks_table.up.sql
├── 000001_create_stocks_table.down.sql
├── 000002_create_prices_table.up.sql
├── 000002_create_prices_table.down.sql
└── ...
```

**예시**: `000001_create_stocks_table.up.sql`

```sql
-- =====================================================
-- 000001: market.stocks 테이블 생성
-- =====================================================

CREATE TABLE IF NOT EXISTS market.stocks (
    code VARCHAR(10) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    market VARCHAR(20) NOT NULL,
    sector VARCHAR(50),
    industry VARCHAR(50),
    listed_date DATE,
    delisted BOOLEAN DEFAULT FALSE,
    delisted_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 인덱스
CREATE INDEX idx_stocks_market ON market.stocks(market);
CREATE INDEX idx_stocks_sector ON market.stocks(sector);
CREATE INDEX idx_stocks_delisted ON market.stocks(delisted) WHERE delisted = FALSE;

-- 설명
COMMENT ON TABLE market.stocks IS '종목 마스터 (SSOT)';
COMMENT ON COLUMN market.stocks.code IS '종목 코드 (KRX)';
COMMENT ON COLUMN market.stocks.name IS '종목명';
COMMENT ON COLUMN market.stocks.market IS '시장 구분 (KOSPI, KOSDAQ, KONEX)';

-- 권한 부여 (명시적)
GRANT SELECT, INSERT, UPDATE, DELETE ON market.stocks TO aegis_v14;
GRANT SELECT ON market.stocks TO aegis_v14_readonly;

\echo 'Table market.stocks created successfully!'
```

### 3.3 마이그레이션 실행

```bash
# 환경 변수 설정
export DATABASE_URL="postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable"

# 마이그레이션 실행
migrate -path backend/migrations -database $DATABASE_URL up

# 특정 버전으로 롤백
migrate -path backend/migrations -database $DATABASE_URL down 1
```

---

## 🔧 Step 4: 권한 문제 해결 (Troubleshooting)

### 4.1 권한 확인 쿼리

```sql
-- 1. Schema 권한 확인
SELECT
    nsp.nspname AS schema_name,
    rol.rolname AS owner,
    pg_catalog.has_schema_privilege('aegis_v14', nsp.nspname, 'CREATE') AS can_create
FROM pg_namespace nsp
JOIN pg_roles rol ON nsp.nspowner = rol.oid
WHERE nsp.nspname IN ('market', 'trade', 'system');

-- 2. 테이블 권한 확인
SELECT
    schemaname,
    tablename,
    tableowner,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'SELECT') AS can_select,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'INSERT') AS can_insert,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'UPDATE') AS can_update,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'DELETE') AS can_delete
FROM pg_tables
WHERE schemaname IN ('market', 'trade', 'system');

-- 3. Default Privileges 확인
SELECT
    pg_get_userbyid(defaclrole) AS grantor,
    nspname AS schema,
    defaclobjtype AS object_type,
    defaclacl AS privileges
FROM pg_default_acl a
JOIN pg_namespace n ON a.defaclnamespace = n.oid
WHERE nspname IN ('market', 'trade', 'system');
```

### 4.2 권한 문제 패턴별 해결

#### Pattern 1: "permission denied for table"

**원인**: 테이블에 대한 권한이 없음

**해결**:
```sql
-- 모든 기존 테이블에 권한 부여
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA market TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA trade TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA system TO aegis_v14;

-- Sequence 권한
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA market TO aegis_v14;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_v14;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA system TO aegis_v14;
```

#### Pattern 2: "must be owner of table"

**원인**: 테이블 소유권이 다른 Role에 있음

**해결**:
```sql
-- 테이블 소유권 변경
ALTER TABLE market.stocks OWNER TO aegis_v14;

-- 모든 테이블 소유권 변경 (일괄)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'market' LOOP
        EXECUTE 'ALTER TABLE market.' || quote_ident(r.tablename) || ' OWNER TO aegis_v14';
    END LOOP;
END $$;
```

#### Pattern 3: "cannot create objects in schema"

**원인**: Schema에 대한 CREATE 권한이 없음

**해결**:
```sql
-- Schema 권한 부여
GRANT CREATE ON SCHEMA market TO aegis_v14;
GRANT CREATE ON SCHEMA trade TO aegis_v14;
GRANT CREATE ON SCHEMA system TO aegis_v14;
```

### 4.3 완전 초기화 스크립트 (Reset)

**파일**: `scripts/db/99_reset_all.sql`

```sql
-- =====================================================
-- 완전 초기화 스크립트 (주의: 모든 데이터 삭제)
-- =====================================================

-- 1. 모든 연결 종료
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'aegis_v14' AND pid <> pg_backend_pid();

-- 2. Database 삭제 및 재생성
DROP DATABASE IF EXISTS aegis_v14;
CREATE DATABASE aegis_v14
    WITH
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0
    OWNER = aegis_v14;

\echo 'Database reset complete. Run 02_create_schemas.sql and migrations.'
```

**실행**:
```bash
psql -U postgres -f scripts/db/99_reset_all.sql
```

---

## 🛠️ 개발 환경별 설정

### 로컬 개발 (Local)

```bash
# .env.local
DB_HOST=localhost
DB_PORT=5432
DB_NAME=aegis_v14
DB_USER=aegis_v14
DB_PASSWORD=aegis_v14_won
DATABASE_URL=postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable
```

**특징**:
- 모든 권한 허용 (개발 편의성)
- `sslmode=disable` (로컬에서는 SSL 불필요)

### 테스트 환경 (Test)

```bash
# .env.test
DB_HOST=localhost
DB_PORT=5432
DB_NAME=aegis_v14_test
DB_USER=aegis_v14_test
DB_PASSWORD=test_password
DATABASE_URL=postgresql://aegis_v14_test:test_password@localhost:5432/aegis_v14_test?sslmode=disable
```

**특징**:
- 독립된 테스트 DB
- 테스트 완료 후 자동 정리

---

## 📝 일일 개발 체크리스트

개발 시작 전 확인:

- [ ] PostgreSQL 서비스 실행 중
- [ ] `.env` 파일 존재 및 정확한 연결 정보
- [ ] `aegis_v14` 데이터베이스 존재
- [ ] Schema (market, trade, system) 존재
- [ ] 권한 확인 쿼리 실행하여 문제 없음

권한 문제 발생 시:

1. **권한 확인 쿼리 실행** (4.1 참고)
2. **해당 패턴의 해결책 적용** (4.2 참고)
3. **여전히 문제 시**: 완전 초기화 후 재설정 (4.3 참고)

---

## 🔍 참고 문서

- [데이터베이스 스키마 설계](./schema.md)
- [데이터베이스 접근 제어](./access-control.md)
- [마이그레이션 계획](./migration-stocks.md)

---

## ⚠️ 주의사항

### 프로덕션 환경에서는

1. **최소 권한 원칙**: `aegis_v14` Role에 CREATE 권한 제거
2. **읽기 전용 분리**: 분석/모니터링은 `aegis_v14_readonly` 사용
3. **SSL 필수**: `sslmode=require`
4. **비밀번호 강화**: 강력한 비밀번호 사용
5. **감사 로그**: `pgaudit` 활성화

### 절대 하지 말 것

- ❌ 프로덕션 DB에서 `DROP DATABASE`
- ❌ Superuser로 애플리케이션 실행
- ❌ 비밀번호를 코드에 하드코딩
- ❌ `DELETE FROM` 권한을 읽기 전용 Role에 부여

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
