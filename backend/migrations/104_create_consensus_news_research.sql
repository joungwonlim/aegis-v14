-- Migration: 104_create_consensus_news_research.sql
-- Description: Create consensus, news, and research tables in data schema
-- Migrated from v10 analysis schema

-- ============================================================================
-- 1. Consensus Table (애널리스트 컨센서스)
-- ============================================================================
CREATE TABLE IF NOT EXISTS data.consensus (
    id BIGSERIAL PRIMARY KEY,
    stock_code VARCHAR(10) NOT NULL,
    consensus_date DATE NOT NULL,
    target_price NUMERIC,
    current_price NUMERIC,
    upside_potential NUMERIC,
    buy_count INTEGER DEFAULT 0,
    hold_count INTEGER DEFAULT 0,
    sell_count INTEGER DEFAULT 0,
    consensus_score NUMERIC,
    eps_estimate NUMERIC,
    per_estimate NUMERIC,
    source VARCHAR(20) DEFAULT 'naver',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT consensus_unique_stock_date UNIQUE (stock_code, consensus_date)
);

-- Indexes for consensus
CREATE INDEX IF NOT EXISTS idx_consensus_stock_code ON data.consensus(stock_code);
CREATE INDEX IF NOT EXISTS idx_consensus_date ON data.consensus(consensus_date DESC);
CREATE INDEX IF NOT EXISTS idx_consensus_stock_date ON data.consensus(stock_code, consensus_date DESC);
CREATE INDEX IF NOT EXISTS idx_consensus_created_at ON data.consensus(created_at DESC);

COMMENT ON TABLE data.consensus IS '애널리스트 컨센서스 데이터 (목표가, 투자의견 등)';
COMMENT ON COLUMN data.consensus.stock_code IS '종목코드';
COMMENT ON COLUMN data.consensus.consensus_date IS '컨센서스 기준일';
COMMENT ON COLUMN data.consensus.target_price IS '목표가';
COMMENT ON COLUMN data.consensus.current_price IS '현재가';
COMMENT ON COLUMN data.consensus.upside_potential IS '상승여력 (%)';
COMMENT ON COLUMN data.consensus.buy_count IS '매수 의견 수';
COMMENT ON COLUMN data.consensus.hold_count IS '보유 의견 수';
COMMENT ON COLUMN data.consensus.sell_count IS '매도 의견 수';
COMMENT ON COLUMN data.consensus.consensus_score IS '컨센서스 점수';
COMMENT ON COLUMN data.consensus.eps_estimate IS 'EPS 추정치';
COMMENT ON COLUMN data.consensus.per_estimate IS 'PER 추정치';

-- ============================================================================
-- 2. News Table (뉴스 기사)
-- ============================================================================
CREATE TABLE IF NOT EXISTS data.news (
    id BIGSERIAL PRIMARY KEY,
    stock_code VARCHAR(6) NOT NULL,
    article_id VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    summary TEXT,
    source VARCHAR(100),
    author VARCHAR(100),
    published_at TIMESTAMPTZ NOT NULL,
    url TEXT,
    sentiment_score DOUBLE PRECISION,
    importance_score DOUBLE PRECISION,
    keywords JSONB,
    category VARCHAR(50),
    is_major BOOLEAN DEFAULT false,

    -- AI 분석 필드
    ai_analyzed BOOLEAN DEFAULT false,
    ai_model VARCHAR(50),
    ai_analyzed_at TIMESTAMPTZ,
    ai_sentiment VARCHAR(20),
    ai_risk_score INTEGER,
    ai_event_tags JSONB,
    ai_summary TEXT,
    ai_key_findings JSONB,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT news_unique_stock_article UNIQUE (stock_code, article_id)
);

-- Indexes for news
CREATE INDEX IF NOT EXISTS idx_news_stock_code ON data.news(stock_code);
CREATE INDEX IF NOT EXISTS idx_news_published_at ON data.news(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_stock_published ON data.news(stock_code, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_ai_analyzed ON data.news(ai_analyzed) WHERE ai_analyzed = false;
CREATE INDEX IF NOT EXISTS idx_news_ai_sentiment ON data.news(ai_sentiment);
CREATE INDEX IF NOT EXISTS idx_news_ai_risk ON data.news(ai_risk_score);
CREATE INDEX IF NOT EXISTS idx_news_created_at ON data.news(created_at DESC);

COMMENT ON TABLE data.news IS '뉴스 기사 데이터 (AI 분석 포함)';
COMMENT ON COLUMN data.news.stock_code IS '종목코드';
COMMENT ON COLUMN data.news.article_id IS '기사 ID (source별 unique)';
COMMENT ON COLUMN data.news.title IS '기사 제목';
COMMENT ON COLUMN data.news.summary IS '기사 요약';
COMMENT ON COLUMN data.news.source IS '출처 (네이버, 한경 등)';
COMMENT ON COLUMN data.news.published_at IS '발행 시간';
COMMENT ON COLUMN data.news.sentiment_score IS '감성 점수 (-1.0 ~ 1.0)';
COMMENT ON COLUMN data.news.importance_score IS '중요도 점수 (0 ~ 1.0)';
COMMENT ON COLUMN data.news.is_major IS '주요 기사 여부';
COMMENT ON COLUMN data.news.ai_analyzed IS 'AI 분석 완료 여부';
COMMENT ON COLUMN data.news.ai_sentiment IS 'AI 판단 감성 (POSITIVE, NEGATIVE, NEUTRAL)';
COMMENT ON COLUMN data.news.ai_risk_score IS 'AI 리스크 점수 (0-100)';

-- ============================================================================
-- 3. Research Table (리서치 보고서)
-- ============================================================================
CREATE TABLE IF NOT EXISTS data.research (
    id BIGSERIAL PRIMARY KEY,
    stock_code VARCHAR(10) NOT NULL,
    title VARCHAR(500) NOT NULL,
    analyst VARCHAR(100),
    firm VARCHAR(100),
    target_price NUMERIC,
    opinion VARCHAR(20),
    published_at TIMESTAMPTZ,
    summary TEXT,
    sentiment_score NUMERIC,
    source_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT research_unique_stock_title_firm UNIQUE (stock_code, title, firm)
);

-- Indexes for research
CREATE INDEX IF NOT EXISTS idx_research_stock_code ON data.research(stock_code);
CREATE INDEX IF NOT EXISTS idx_research_published_at ON data.research(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_research_stock_published ON data.research(stock_code, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_research_firm ON data.research(firm);
CREATE INDEX IF NOT EXISTS idx_research_opinion ON data.research(opinion);
CREATE INDEX IF NOT EXISTS idx_research_created_at ON data.research(created_at DESC);

COMMENT ON TABLE data.research IS '증권사 리서치 보고서';
COMMENT ON COLUMN data.research.stock_code IS '종목코드';
COMMENT ON COLUMN data.research.title IS '리포트 제목';
COMMENT ON COLUMN data.research.analyst IS '애널리스트명';
COMMENT ON COLUMN data.research.firm IS '증권사명';
COMMENT ON COLUMN data.research.target_price IS '목표가';
COMMENT ON COLUMN data.research.opinion IS '투자의견 (BUY, HOLD, SELL 등)';
COMMENT ON COLUMN data.research.published_at IS '발행일시';
COMMENT ON COLUMN data.research.sentiment_score IS '감성 점수';
