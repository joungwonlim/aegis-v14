#!/bin/bash
# ================================================================
# Aegis v14 - v10 stocks 데이터 마이그레이션 자동화 스크립트
# ================================================================
set -e

echo "================================================="
echo "v10 → v14 Stocks 데이터 마이그레이션"
echo "================================================="
echo ""

# 변수
V10_DB="aegis_v10"
V10_USER="aegis_v10"
V14_DB="aegis_v14"
V14_USER="aegis_v14"
DUMP_FILE="/tmp/v10_stocks_dump.csv"

# 1. v10에서 데이터 덤프
echo "1. v10 데이터베이스에서 종목 데이터 덤프 (6자리 숫자 종목만)..."
psql -U "$V10_USER" -d "$V10_DB" << EOF
\\copy (
    SELECT
        stock_code,
        stock_name,
        market,
        sector,
        is_active,
        listed_date,
        delisted_date
    FROM market.stocks
    WHERE stock_code ~ '^[0-9]{6}\$'
    ORDER BY stock_code
) TO '$DUMP_FILE' CSV HEADER
EOF

RECORD_COUNT=$(wc -l < "$DUMP_FILE" | xargs)
RECORD_COUNT=$((RECORD_COUNT - 1))  # Subtract header
echo "✅ $RECORD_COUNT 개 종목 덤프 완료: $DUMP_FILE"
echo ""

# 2. v14 기존 데이터 백업
echo "2. v14 기존 데이터 백업..."
psql -U "$V14_USER" -d "$V14_DB" << 'EOF'
DROP TABLE IF EXISTS market.stocks_backup_before_v10_migration;
CREATE TABLE market.stocks_backup_before_v10_migration AS
SELECT * FROM market.stocks;
EOF
echo "✅ 백업 완료 (market.stocks_backup_before_v10_migration)"
echo ""

# 3. v14 데이터 삭제
echo "3. v14 기존 데이터 삭제..."
psql -U "$V14_USER" -d "$V14_DB" -c "TRUNCATE market.stocks CASCADE;"
echo "✅ 기존 데이터 삭제 완료"
echo ""

# 4. 임시 테이블 생성 및 데이터 로드
echo "4. v14에 데이터 임포트..."
psql -U "$V14_USER" -d "$V14_DB" << EOF
-- 임시 테이블 생성
DROP TABLE IF EXISTS market.stocks_temp;
CREATE TABLE market.stocks_temp (
    stock_code CHAR(6),
    stock_name VARCHAR(100),
    market VARCHAR(20),
    sector VARCHAR(50),
    is_active BOOLEAN,
    listed_date DATE,
    delisted_date DATE
);

-- CSV 데이터 로드
\\copy market.stocks_temp FROM '$DUMP_FILE' CSV HEADER

-- v14 스키마에 맞게 변환하여 INSERT
INSERT INTO market.stocks (
    symbol,
    name,
    market,
    status,
    listing_date,
    delisting_date,
    sector,
    industry,
    is_tradable,
    created_ts,
    updated_ts
)
SELECT
    stock_code,
    stock_name,
    market,
    CASE
        WHEN is_active = true AND delisted_date IS NULL THEN 'ACTIVE'
        WHEN delisted_date IS NOT NULL THEN 'DELISTED'
        ELSE 'SUSPENDED'
    END,
    listed_date,
    delisted_date,
    sector,
    sector,  -- industry는 sector와 동일하게 설정
    is_active,
    NOW(),
    NOW()
FROM market.stocks_temp;

-- 임시 테이블 삭제
DROP TABLE market.stocks_temp;

-- 결과 확인
SELECT
    market,
    status,
    COUNT(*) as count
FROM market.stocks
GROUP BY market, status
ORDER BY market, status;
EOF

echo "✅ 데이터 임포트 완료"
echo ""

# 5. 검증
echo "5. 마이그레이션 검증..."
psql -U "$V14_USER" -d "$V14_DB" << 'EOF'
SELECT
    'Total' as category,
    COUNT(*) as count
FROM market.stocks
UNION ALL
SELECT
    'ACTIVE' as category,
    COUNT(*) as count
FROM market.stocks
WHERE status = 'ACTIVE'
UNION ALL
SELECT
    'KOSPI' as category,
    COUNT(*) as count
FROM market.stocks
WHERE market = 'KOSPI'
UNION ALL
SELECT
    'KOSDAQ' as category,
    COUNT(*) as count
FROM market.stocks
WHERE market = 'KOSDAQ';

\echo ''
\echo '샘플 데이터 (KOSPI 상위 5개):'
SELECT symbol, name, market, sector, status
FROM market.stocks
WHERE market = 'KOSPI'
ORDER BY symbol
LIMIT 5;
EOF

echo ""
echo "================================================="
echo "✅ 마이그레이션 완료!"
echo "================================================="
echo ""
echo "임시 파일: $DUMP_FILE (수동 삭제 필요)"
echo "백업 테이블: market.stocks_backup_before_v10_migration"
echo ""
