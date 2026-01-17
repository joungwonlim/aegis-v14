-- v14 Watchlist 테이블 생성
-- 관심종목(watch) 및 투자할종목(candidate) 관리

CREATE TABLE IF NOT EXISTS portfolio.watchlist (
    id SERIAL PRIMARY KEY,
    stock_code CHAR(6) NOT NULL,                  -- 종목코드
    category VARCHAR(20) NOT NULL,                -- 'watch' | 'candidate'

    -- 추가 정보
    memo TEXT,                                    -- 메모/선정이유
    target_price BIGINT,                          -- 목표가

    -- AI 분석 필드 (candidate용)
    grok_analysis TEXT,
    gemini_analysis TEXT,
    chatgpt_analysis TEXT,
    claude_analysis TEXT,

    -- 알림 설정
    alert_enabled BOOLEAN DEFAULT false,
    alert_price BIGINT,
    alert_condition VARCHAR(20),                  -- 'above' | 'below'

    -- 타임스탬프
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- 제약조건: 같은 종목은 카테고리별로 하나만
    UNIQUE(stock_code, category)
);

-- 인덱스
CREATE INDEX idx_watchlist_stock_code ON portfolio.watchlist(stock_code);
CREATE INDEX idx_watchlist_category ON portfolio.watchlist(category);
CREATE INDEX idx_watchlist_created_at ON portfolio.watchlist(created_at DESC);

-- 타임스탬프 자동 갱신 트리거
CREATE OR REPLACE FUNCTION portfolio.update_watchlist_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_watchlist_updated_at
    BEFORE UPDATE ON portfolio.watchlist
    FOR EACH ROW
    EXECUTE FUNCTION portfolio.update_watchlist_timestamp();

-- Comment
COMMENT ON TABLE portfolio.watchlist IS '관심종목 및 투자할종목 관리';
COMMENT ON COLUMN portfolio.watchlist.category IS 'watch: 관심종목, candidate: 투자할종목';
COMMENT ON COLUMN portfolio.watchlist.memo IS '메모 또는 선정이유';
COMMENT ON COLUMN portfolio.watchlist.target_price IS '목표 매도가';
