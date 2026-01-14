-- =====================================================
-- v14 권한 문제 수정 스크립트
-- =====================================================
-- 권한 문제 발생 시 이 스크립트 실행
-- 실행: psql -U aegis_v14 -d aegis_v14 -f scripts/db/04_fix_permissions.sql
-- =====================================================

\echo ''
\echo '========================================='
\echo 'Fixing Database Permissions...'
\echo '========================================='

-- 1. 모든 기존 테이블에 권한 부여
\echo ''
\echo '1. Granting permissions on existing tables...'

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA market TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA trade TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA system TO aegis_v14;

GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_v14_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA trade TO aegis_v14_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA system TO aegis_v14_readonly;

-- 2. Sequence 권한
\echo '2. Granting permissions on sequences...'

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA market TO aegis_v14;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_v14;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA system TO aegis_v14;

-- 3. 테이블 소유권 변경 (market schema)
\echo '3. Changing ownership of tables in market schema...'

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'market' LOOP
        EXECUTE 'ALTER TABLE market.' || quote_ident(r.tablename) || ' OWNER TO aegis_v14';
    END LOOP;
END $$;

-- 4. 테이블 소유권 변경 (trade schema)
\echo '4. Changing ownership of tables in trade schema...'

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'trade' LOOP
        EXECUTE 'ALTER TABLE trade.' || quote_ident(r.tablename) || ' OWNER TO aegis_v14';
    END LOOP;
END $$;

-- 5. 테이블 소유권 변경 (system schema)
\echo '5. Changing ownership of tables in system schema...'

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'system' LOOP
        EXECUTE 'ALTER TABLE system.' || quote_ident(r.tablename) || ' OWNER TO aegis_v14';
    END LOOP;
END $$;

-- 6. 향후 생성될 객체에 대한 권한 재설정
\echo '6. Resetting default privileges...'

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT USAGE, SELECT ON SEQUENCES TO aegis_v14;

ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA market
    GRANT SELECT ON TABLES TO aegis_v14_readonly;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_v14_readonly;
ALTER DEFAULT PRIVILEGES FOR ROLE aegis_v14 IN SCHEMA system
    GRANT SELECT ON TABLES TO aegis_v14_readonly;

\echo ''
\echo '========================================='
\echo 'Permissions fixed successfully!'
\echo '========================================='
\echo ''
\echo 'Run 03_check_permissions.sql to verify:'
\echo '  psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql'
\echo ''
