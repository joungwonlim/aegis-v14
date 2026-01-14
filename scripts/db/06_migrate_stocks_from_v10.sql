-- ================================================================
-- Aegis v14 - v10에서 stocks 데이터 마이그레이션
-- ================================================================
-- v10의 market.stocks → v14의 market.stocks
-- 실행: psql -U aegis_v14 -d aegis_v14 -f scripts/db/06_migrate_stocks_from_v10.sql

\echo '================================================='
\echo 'v10 → v14 Stocks 마이그레이션'
\echo '================================================='
\echo ''

-- 1. 기존 데이터 백업
\echo '1. 기존 데이터 백업...'
CREATE TABLE IF NOT EXISTS market.stocks_backup_before_v10_migration AS
SELECT * FROM market.stocks;

\echo '✅ 백업 완료'
\echo ''

-- 2. 기존 데이터 삭제
\echo '2. 기존 테스트 데이터 삭제...'
TRUNCATE market.stocks CASCADE;
\echo '✅ 삭제 완료'
\echo ''

-- 3. v10 데이터 복사
\echo '3. v10 데이터 복사 중...'
\echo '   (v10 DB에서 postgres_fdw 또는 덤프 필요)'
\echo ''

-- Option A: postgres_fdw 사용 (v10 DB가 실행 중인 경우)
-- CREATE EXTENSION IF NOT EXISTS postgres_fdw;
--
-- CREATE SERVER IF NOT EXISTS v10_server
--     FOREIGN DATA WRAPPER postgres_fdw
--     OPTIONS (host 'localhost', port '5432', dbname 'aegis_v10');
--
-- CREATE USER MAPPING IF NOT EXISTS FOR aegis_v14
--     SERVER v10_server
--     OPTIONS (user 'aegis_v10', password 'aegis_v10_won');
--
-- CREATE FOREIGN TABLE IF NOT EXISTS v10_stocks (
--     stock_code CHAR(6),
--     stock_name VARCHAR(100),
--     market VARCHAR(20),
--     sector VARCHAR(50),
--     is_active BOOLEAN,
--     listed_date DATE,
--     delisted_date DATE
-- ) SERVER v10_server OPTIONS (schema_name 'market', table_name 'stocks');
--
-- INSERT INTO market.stocks (
--     symbol, name, market, status, listing_date, delisting_date,
--     sector, industry, is_tradable, created_ts, updated_ts
-- )
-- SELECT
--     stock_code,
--     stock_name,
--     market,
--     CASE
--         WHEN is_active = true THEN 'ACTIVE'
--         WHEN delisted_date IS NOT NULL THEN 'DELISTED'
--         ELSE 'SUSPENDED'
--     END,
--     listed_date,
--     delisted_date,
--     sector,
--     sector,  -- industry도 sector로 매핑
--     is_active,
--     NOW(),
--     NOW()
-- FROM v10_stocks;

\echo '⚠️  수동 실행 필요:'
\echo '   아래 명령어를 별도로 실행하세요:'
\echo ''
\echo '   # v10 데이터 덤프'
\echo '   psql -U aegis_v10 -d aegis_v10 -c "\\copy (SELECT stock_code, stock_name, market, sector, is_active, listed_date, delisted_date FROM market.stocks) TO '\''/tmp/v10_stocks.csv'\'' CSV HEADER"'
\echo ''
\echo '   # v14로 import'
\echo '   psql -U aegis_v14 -d aegis_v14 -c "\\copy market.stocks_temp FROM '\''/tmp/v10_stocks.csv'\'' CSV HEADER"'
\echo ''

-- 4. 임시 테이블 생성 (수동 import용)
CREATE TABLE IF NOT EXISTS market.stocks_temp (
    stock_code CHAR(6),
    stock_name VARCHAR(100),
    market VARCHAR(20),
    sector VARCHAR(50),
    is_active BOOLEAN,
    listed_date DATE,
    delisted_date DATE
);

\echo '✅ 임시 테이블 생성 완료 (market.stocks_temp)'
\echo ''
\echo '================================================='
\echo '다음 단계: 외부에서 데이터 복사 필요'
\echo '================================================='
