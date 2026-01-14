-- ================================================================
-- Aegis v14 - market.stocks 테이블 생성
-- ================================================================
-- 종목 마스터 테이블 (SSOT)
-- 실행: psql -U aegis_v14 -d aegis_v14 -f scripts/db/05_create_stocks_table.sql

\echo '================================================='
\echo 'Creating market.stocks table...'
\echo '================================================='
\echo ''

-- 1. Create stocks table
CREATE TABLE IF NOT EXISTS market.stocks (
    symbol        TEXT PRIMARY KEY,  -- 종목코드 (예: 005930, 069500) - 6자리 숫자
    name          TEXT        NOT NULL,     -- 종목명 (예: 삼성전자)
    market        TEXT        NOT NULL,     -- KOSPI | KOSDAQ | KONEX

    -- 종목 상태
    status        TEXT        NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE | SUSPENDED | DELISTED
    listing_date  DATE,                     -- 상장일
    delisting_date DATE,                    -- 상장폐지일

    -- 메타 정보
    sector        TEXT,                     -- 섹터 (예: 전기전자)
    industry      TEXT,                     -- 업종 (예: 반도체)
    market_cap    BIGINT,                   -- 시가총액 (원)

    -- 거래 제약
    is_tradable   BOOLEAN     NOT NULL DEFAULT true,  -- 현재 거래 가능 여부
    trade_halt_reason TEXT,                -- 거래정지 사유

    -- 감사
    created_ts    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_ts    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_market CHECK (market IN ('KOSPI', 'KOSDAQ', 'KONEX')),
    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELISTED')),
    CONSTRAINT chk_symbol_format CHECK (symbol ~ '^\d{6}$')  -- 6자리 숫자 검증
);

\echo '✅ market.stocks table created'
\echo ''

-- 2. Create indexes
CREATE INDEX IF NOT EXISTS idx_stocks_market ON market.stocks (market);
CREATE INDEX IF NOT EXISTS idx_stocks_status ON market.stocks (status);
CREATE INDEX IF NOT EXISTS idx_stocks_tradable ON market.stocks (is_tradable) WHERE is_tradable = true;
CREATE INDEX IF NOT EXISTS idx_stocks_name ON market.stocks (name);  -- 종목명 검색용

\echo '✅ Indexes created'
\echo ''

-- 3. Insert test data
INSERT INTO market.stocks (symbol, name, market, status, listing_date, sector, industry, market_cap, is_tradable)
VALUES
    ('005930', '삼성전자', 'KOSPI', 'ACTIVE', '1975-06-11', '전기전자', '반도체', 500000000000000, true),
    ('000660', 'SK하이닉스', 'KOSPI', 'ACTIVE', '1996-12-26', '전기전자', '반도체', 80000000000000, true),
    ('035420', 'NAVER', 'KOSPI', 'ACTIVE', '2002-10-29', '서비스업', '인터넷', 30000000000000, true),
    ('035720', '카카오', 'KOSPI', 'ACTIVE', '2017-07-10', '서비스업', '인터넷', 25000000000000, true),
    ('051910', 'LG화학', 'KOSPI', 'ACTIVE', '2001-04-24', '화학', '화학', 50000000000000, true),
    ('006400', '삼성SDI', 'KOSPI', 'ACTIVE', '1979-10-30', '전기전자', '2차전지', 45000000000000, true),
    ('207940', '삼성바이오로직스', 'KOSPI', 'ACTIVE', '2016-11-10', '의약품', '바이오', 70000000000000, true),
    ('068270', '셀트리온', 'KOSPI', 'ACTIVE', '2008-07-02', '의약품', '바이오', 35000000000000, true),
    ('373220', 'LG에너지솔루션', 'KOSPI', 'ACTIVE', '2022-01-27', '전기전자', '2차전지', 120000000000000, true),
    ('352820', '하이브', 'KOSPI', 'ACTIVE', '2020-10-15', '서비스업', '엔터테인먼트', 8000000000000, true),

    -- KOSDAQ stocks
    ('247540', '에코프로비엠', 'KOSDAQ', 'ACTIVE', '2016-11-01', '화학', '2차전지소재', 15000000000000, true),
    ('086520', '에코프로', 'KOSDAQ', 'ACTIVE', '2007-03-23', '화학', '2차전지소재', 20000000000000, true),
    ('091990', '셀트리온헬스케어', 'KOSDAQ', 'ACTIVE', '2012-11-07', '의약품', '바이오', 18000000000000, true),
    ('196170', '알테오젠', 'KOSDAQ', 'ACTIVE', '2008-11-27', '의약품', '바이오', 12000000000000, true),
    ('293490', '카카오게임즈', 'KOSDAQ', 'ACTIVE', '2020-09-10', '서비스업', '게임', 5000000000000, true),

    -- ETF stocks
    ('069500', 'KODEX 200', 'KOSPI', 'ACTIVE', '2002-10-14', 'ETF', 'ETF', 10000000000000, true),
    ('114800', 'KODEX 인버스', 'KOSPI', 'ACTIVE', '2009-01-08', 'ETF', 'ETF', 2000000000000, true),
    ('102110', 'TIGER 200', 'KOSPI', 'ACTIVE', '2001-10-25', 'ETF', 'ETF', 8000000000000, true),

    -- Suspended stock (for testing)
    ('999999', '테스트정지', 'KOSPI', 'SUSPENDED', '2020-01-01', '테스트', '테스트', 1000000000, false)
ON CONFLICT (symbol) DO NOTHING;

\echo '✅ Test data inserted (19 stocks)'
\echo ''

-- 4. Verify
SELECT
    market,
    COUNT(*) as count,
    COUNT(*) FILTER (WHERE is_tradable = true) as tradable_count
FROM market.stocks
GROUP BY market
ORDER BY market;

\echo ''
\echo '================================================='
\echo '✅ market.stocks table setup complete'
\echo '================================================='
