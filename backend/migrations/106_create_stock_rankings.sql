-- =====================================================
-- Stock Rankings Table
-- =====================================================
-- Purpose: 네이버에서 수집한 순위 데이터 저장
-- Fetcher가 10분마다 갱신

CREATE TABLE IF NOT EXISTS data.stock_rankings (
    id BIGSERIAL PRIMARY KEY,

    -- 순위 정보
    category VARCHAR(50) NOT NULL,  -- volume, trading_value, gainers, foreign_net_buy, inst_net_buy, volume_surge, high_52week
    market VARCHAR(20) NOT NULL,    -- ALL, KOSPI, KOSDAQ
    rank INT NOT NULL,

    -- 종목 정보
    stock_code VARCHAR(20) NOT NULL,
    stock_name VARCHAR(100) NOT NULL,

    -- 가격 정보 (모든 카테고리 공통)
    current_price NUMERIC(15,2),
    change_rate NUMERIC(10,4),
    volume BIGINT,
    trading_value BIGINT,
    high_price NUMERIC(15,2),
    low_price NUMERIC(15,2),

    -- 카테고리별 특화 데이터
    foreign_net_value BIGINT,      -- 외국인 순매수
    inst_net_value BIGINT,          -- 기관 순매수
    volume_surge_rate NUMERIC(10,2), -- 거래량 급증률
    high_52week NUMERIC(15,2),      -- 52주 최고가
    market_cap BIGINT,              -- 시가총액

    -- 타임스탬프
    collected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 인덱스: 조회 성능 최적화
CREATE INDEX idx_stock_rankings_category_market_rank
    ON data.stock_rankings(category, market, rank);

CREATE INDEX idx_stock_rankings_category_market_collected
    ON data.stock_rankings(category, market, collected_at DESC);

CREATE INDEX idx_stock_rankings_stock_code
    ON data.stock_rankings(stock_code);

CREATE INDEX idx_stock_rankings_collected_at
    ON data.stock_rankings(collected_at DESC);

-- 코멘트
COMMENT ON TABLE data.stock_rankings IS '네이버 순위 데이터 (10분마다 갱신)';
COMMENT ON COLUMN data.stock_rankings.category IS '순위 카테고리: volume, trading_value, gainers, foreign_net_buy, inst_net_buy, volume_surge, high_52week';
COMMENT ON COLUMN data.stock_rankings.market IS '시장 구분: ALL, KOSPI, KOSDAQ';
COMMENT ON COLUMN data.stock_rankings.collected_at IS '데이터 수집 시각';
