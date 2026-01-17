-- v13 → v14 마이그레이션: Data 테이블 생성
-- v14.0.0
-- 2026-01-17

-- ================================================
-- 1. data.stocks (종목 마스터)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.stocks (
    code            VARCHAR(20) PRIMARY KEY,
    name            VARCHAR(200) NOT NULL,
    market          VARCHAR(20) NOT NULL,           -- KOSPI, KOSDAQ, KONEX
    sector          VARCHAR(100),
    listing_date    DATE NOT NULL,
    delisting_date  DATE,
    status          VARCHAR(20) DEFAULT 'active',   -- active, delisted, suspended
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_data_stocks_market ON data.stocks(market);
CREATE INDEX idx_data_stocks_sector ON data.stocks(sector);
CREATE INDEX idx_data_stocks_status ON data.stocks(status);

COMMENT ON TABLE data.stocks IS '종목 마스터 - Fetcher 소유';

-- ================================================
-- 2. data.daily_prices (일봉 데이터 - 파티션)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.daily_prices (
    stock_code      VARCHAR(20) NOT NULL,
    trade_date      DATE NOT NULL,
    open_price      NUMERIC(12,2) NOT NULL,
    high_price      NUMERIC(12,2) NOT NULL,
    low_price       NUMERIC(12,2) NOT NULL,
    close_price     NUMERIC(12,2) NOT NULL,
    volume          BIGINT NOT NULL,
    trading_value   NUMERIC(15,0),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 2026년 상반기 파티션
CREATE TABLE data.daily_prices_2026_h1 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2026-01-01') TO ('2026-07-01');

-- 2026년 하반기 파티션
CREATE TABLE data.daily_prices_2026_h2 PARTITION OF data.daily_prices
    FOR VALUES FROM ('2026-07-01') TO ('2027-01-01');

CREATE INDEX idx_daily_prices_date ON data.daily_prices(trade_date);
CREATE INDEX idx_daily_prices_code_date ON data.daily_prices(stock_code, trade_date DESC);

COMMENT ON TABLE data.daily_prices IS '일봉 데이터 - 반기별 파티션';

-- ================================================
-- 3. data.investor_flow (투자자별 수급 - 파티션)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.investor_flow (
    stock_code          VARCHAR(20) NOT NULL,
    trade_date          DATE NOT NULL,
    foreign_net_qty     BIGINT DEFAULT 0,
    foreign_net_value   BIGINT DEFAULT 0,
    inst_net_qty        BIGINT DEFAULT 0,
    inst_net_value      BIGINT DEFAULT 0,
    indiv_net_qty       BIGINT DEFAULT 0,
    indiv_net_value     BIGINT DEFAULT 0,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 2026년 상반기 파티션
CREATE TABLE data.investor_flow_2026_h1 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2026-01-01') TO ('2026-07-01');

-- 2026년 하반기 파티션
CREATE TABLE data.investor_flow_2026_h2 PARTITION OF data.investor_flow
    FOR VALUES FROM ('2026-07-01') TO ('2027-01-01');

CREATE INDEX idx_investor_flow_date ON data.investor_flow(trade_date);
CREATE INDEX idx_investor_flow_code_date ON data.investor_flow(stock_code, trade_date DESC);

COMMENT ON TABLE data.investor_flow IS '투자자별 수급 데이터 - 외국인/기관/개인';

-- ================================================
-- 4. data.fundamentals (재무 데이터)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.fundamentals (
    stock_code          VARCHAR(20) NOT NULL,
    report_date         DATE NOT NULL,
    per                 NUMERIC(10,2),
    pbr                 NUMERIC(10,2),
    psr                 NUMERIC(10,2),
    roe                 NUMERIC(10,2),
    debt_ratio          NUMERIC(10,2),
    revenue             BIGINT,
    operating_profit    BIGINT,
    net_profit          BIGINT,
    eps                 NUMERIC(10,2),
    bps                 NUMERIC(10,2),
    dps                 NUMERIC(10,2),
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, report_date)
);

CREATE INDEX idx_fundamentals_date ON data.fundamentals(report_date);
CREATE INDEX idx_fundamentals_code ON data.fundamentals(stock_code, report_date DESC);

COMMENT ON TABLE data.fundamentals IS '재무 데이터 - PER, PBR, ROE 등';

-- ================================================
-- 5. data.market_cap (시가총액)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.market_cap (
    stock_code      VARCHAR(20) NOT NULL,
    trade_date      DATE NOT NULL,
    market_cap      BIGINT NOT NULL,            -- 시가총액 (원)
    shares_out      BIGINT,                     -- 발행주식수
    float_shares    BIGINT,                     -- 유동주식수
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (stock_code, trade_date)
);

CREATE INDEX idx_market_cap_date ON data.market_cap(trade_date);
CREATE INDEX idx_market_cap_size ON data.market_cap(trade_date, market_cap DESC);

COMMENT ON TABLE data.market_cap IS '시가총액 데이터';

-- ================================================
-- 6. data.disclosures (DART 공시)
-- SSOT: Fetcher만 쓰기 가능
-- ================================================
CREATE TABLE data.disclosures (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL,
    disclosed_at    TIMESTAMPTZ NOT NULL,
    title           TEXT NOT NULL,
    category        VARCHAR(100),
    subcategory     VARCHAR(100),
    content         TEXT,
    url             TEXT,
    dart_rcept_no   VARCHAR(50),                -- DART 접수번호
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_disclosures_stock ON data.disclosures(stock_code);
CREATE INDEX idx_disclosures_date ON data.disclosures(disclosed_at DESC);
CREATE INDEX idx_disclosures_category ON data.disclosures(category, disclosed_at DESC);

COMMENT ON TABLE data.disclosures IS 'DART 공시 데이터';

-- ================================================
-- 7. data.universe_snapshots (유니버스 스냅샷)
-- SSOT: Universe 서비스만 쓰기 가능
-- ================================================
CREATE TABLE data.universe_snapshots (
    snapshot_date   DATE PRIMARY KEY,
    market          VARCHAR(20) NOT NULL DEFAULT 'ALL',
    eligible_stocks JSONB NOT NULL,             -- [code1, code2, ...]
    total_count     INT NOT NULL,
    criteria        JSONB,                      -- {min_market_cap: 100억, ...}
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_universe_market_date ON data.universe_snapshots(market, snapshot_date DESC);

COMMENT ON TABLE data.universe_snapshots IS '투자 가능 종목 유니버스 (일별)';
