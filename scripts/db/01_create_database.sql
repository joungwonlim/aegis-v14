-- =====================================================
-- v14 Database 및 Role 생성 스크립트
-- =====================================================
-- 실행: psql -U postgres -f scripts/db/01_create_database.sql
-- =====================================================

-- 1. 기존 연결 종료 (있다면)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'aegis_v14' AND pid <> pg_backend_pid();

-- 2. Database 존재 확인 및 생성
DROP DATABASE IF EXISTS aegis_v14;
CREATE DATABASE aegis_v14
    WITH
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TEMPLATE = template0;

COMMENT ON DATABASE aegis_v14 IS 'Aegis v14 Quant Trading System';

-- 3. 기본 Role 생성
DROP ROLE IF EXISTS aegis_v14;
CREATE ROLE aegis_v14 WITH
    LOGIN
    PASSWORD 'aegis_v14_won'
    CREATEDB           -- 로컬 개발용: 테스트 DB 생성 권한
    NOSUPERUSER
    NOCREATEROLE
    NOREPLICATION;

COMMENT ON ROLE aegis_v14 IS 'v14 Application Default User';

-- 4. 읽기 전용 Role (분석/모니터링용)
DROP ROLE IF EXISTS aegis_v14_readonly;
CREATE ROLE aegis_v14_readonly WITH
    LOGIN
    PASSWORD 'aegis_v14_readonly'
    NOSUPERUSER
    NOCREATEROLE
    NOCREATEDB
    NOREPLICATION;

COMMENT ON ROLE aegis_v14_readonly IS 'v14 Read-Only User for Analytics';

-- 5. Database 소유권 변경
ALTER DATABASE aegis_v14 OWNER TO aegis_v14;

-- 6. 완료 메시지
\echo ''
\echo '========================================='
\echo 'Database and Roles created successfully!'
\echo '========================================='
\echo ''
\echo 'Database: aegis_v14'
\echo 'Owner:    aegis_v14'
\echo 'Roles:    aegis_v14, aegis_v14_readonly'
\echo ''
\echo 'Next step: Run 02_create_schemas.sql'
\echo '  psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql'
\echo ''
