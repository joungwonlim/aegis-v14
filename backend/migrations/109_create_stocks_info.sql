-- 109_create_stocks_info.sql
-- 종목 기본 정보 (company overview 등) 저장 테이블

CREATE TABLE IF NOT EXISTS stocks_info (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL UNIQUE,
    symbol_name VARCHAR(100),

    -- Company Overview (네이버증권에서 가져온 기업 개요)
    company_overview TEXT,
    overview_source VARCHAR(50) DEFAULT 'naver',  -- 출처 (naver, dart, manual 등)
    overview_updated_at TIMESTAMPTZ,

    -- 추가 정보 (향후 확장)
    sector VARCHAR(100),           -- 업종
    industry VARCHAR(100),         -- 산업
    listing_date DATE,             -- 상장일
    fiscal_month INTEGER,          -- 결산월
    homepage VARCHAR(255),         -- 홈페이지

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 인덱스
CREATE INDEX IF NOT EXISTS idx_stocks_info_symbol ON stocks_info(symbol);

-- updated_at 자동 갱신 트리거
CREATE OR REPLACE FUNCTION update_stocks_info_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_stocks_info_updated_at ON stocks_info;
CREATE TRIGGER trigger_stocks_info_updated_at
    BEFORE UPDATE ON stocks_info
    FOR EACH ROW
    EXECUTE FUNCTION update_stocks_info_updated_at();

-- 코멘트
COMMENT ON TABLE stocks_info IS '종목 기본 정보 (company overview 등)';
COMMENT ON COLUMN stocks_info.company_overview IS '기업 개요 (네이버증권 등에서 가져온 텍스트)';
COMMENT ON COLUMN stocks_info.overview_source IS '개요 출처 (naver, dart, manual 등)';
