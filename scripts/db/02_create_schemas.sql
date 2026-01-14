-- =====================================================
-- v14 Schema 생성 및 권한 설정
-- =====================================================
-- 실행: psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql
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

-- 7. 완료 메시지 및 권한 확인
\echo ''
\echo '========================================='
\echo 'Schemas and permissions configured!'
\echo '========================================='
\echo ''
\echo 'Schemas created:'

SELECT
    nspname AS schema_name,
    nspowner::regrole AS owner
FROM pg_namespace
WHERE nspname IN ('market', 'trade', 'system')
ORDER BY nspname;

\echo ''
\echo 'Next step: Run migrations to create tables'
\echo '  migrate -path backend/migrations -database $DATABASE_URL up'
\echo ''
