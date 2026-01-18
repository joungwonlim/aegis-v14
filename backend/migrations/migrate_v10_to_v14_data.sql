-- Migration Script: v10 → v14 Data Migration
-- Description: Copy consensus, news, research data from aegis_v10 to aegis_v14
--
-- Prerequisites:
--   - dblink extension must be enabled
--   - v10 database must be accessible
--
-- Usage:
--   psql -h localhost -U wonny -d aegis_v14 -f migrate_v10_to_v14_data.sql

-- Enable dblink extension if not already enabled
CREATE EXTENSION IF NOT EXISTS dblink;

-- ============================================================================
-- 1. Migrate Consensus Data
-- ============================================================================
DO $$
DECLARE
    v10_conn TEXT := 'host=localhost dbname=aegis_v10 user=wonny';
    rows_inserted INTEGER;
BEGIN
    RAISE NOTICE '=== Starting Consensus Migration ===';

    -- Insert data from v10.analysis.consensus to v14.data.consensus
    INSERT INTO data.consensus (
        stock_code,
        consensus_date,
        target_price,
        current_price,
        upside_potential,
        buy_count,
        hold_count,
        sell_count,
        consensus_score,
        eps_estimate,
        per_estimate,
        source,
        created_at,
        updated_at
    )
    SELECT
        stock_code,
        consensus_date,
        target_price,
        current_price,
        upside_potential,
        COALESCE(buy_count, 0),
        COALESCE(hold_count, 0),
        COALESCE(sell_count, 0),
        consensus_score,
        eps_estimate,
        per_estimate,
        COALESCE(source, 'naver'),
        COALESCE(collected_at, NOW()),
        NOW()
    FROM dblink(
        v10_conn,
        'SELECT
            stock_code, consensus_date, target_price, current_price, upside_potential,
            buy_count, hold_count, sell_count, consensus_score, eps_estimate, per_estimate,
            source, collected_at
         FROM analysis.consensus
         ORDER BY consensus_date DESC'
    ) AS t(
        stock_code VARCHAR(10),
        consensus_date DATE,
        target_price NUMERIC,
        current_price NUMERIC,
        upside_potential NUMERIC,
        buy_count INTEGER,
        hold_count INTEGER,
        sell_count INTEGER,
        consensus_score NUMERIC,
        eps_estimate NUMERIC,
        per_estimate NUMERIC,
        source VARCHAR(20),
        collected_at TIMESTAMPTZ
    )
    ON CONFLICT (stock_code, consensus_date) DO NOTHING;

    GET DIAGNOSTICS rows_inserted = ROW_COUNT;
    RAISE NOTICE 'Consensus: % rows inserted', rows_inserted;
END $$;

-- ============================================================================
-- 2. Migrate News Data
-- ============================================================================
DO $$
DECLARE
    v10_conn TEXT := 'host=localhost dbname=aegis_v10 user=wonny';
    rows_inserted INTEGER;
BEGIN
    RAISE NOTICE '=== Starting News Migration ===';

    -- Insert data from v10.analysis.news to v14.data.news
    INSERT INTO data.news (
        stock_code,
        article_id,
        title,
        summary,
        source,
        author,
        published_at,
        url,
        sentiment_score,
        importance_score,
        keywords,
        category,
        is_major,
        ai_analyzed,
        ai_model,
        ai_analyzed_at,
        ai_sentiment,
        ai_risk_score,
        ai_event_tags,
        ai_summary,
        ai_key_findings,
        created_at,
        updated_at
    )
    SELECT
        stock_code,
        article_id,
        title,
        summary,
        source,
        author,
        published_at,
        url,
        sentiment_score,
        importance_score,
        keywords,
        category,
        COALESCE(is_major, false),
        COALESCE(ai_analyzed, false),
        ai_model,
        ai_analyzed_at,
        ai_sentiment,
        ai_risk_score,
        ai_event_tags,
        ai_summary,
        ai_key_findings,
        COALESCE(fetched_at, NOW()),
        NOW()
    FROM dblink(
        v10_conn,
        'SELECT
            stock_code, article_id, title, summary, source, author, published_at, url,
            sentiment_score, importance_score, keywords, category, is_major,
            ai_analyzed, ai_model, ai_analyzed_at, ai_sentiment, ai_risk_score,
            ai_event_tags, ai_summary, ai_key_findings, fetched_at
         FROM analysis.news
         ORDER BY published_at DESC'
    ) AS t(
        stock_code VARCHAR(6),
        article_id VARCHAR(50),
        title VARCHAR(500),
        summary TEXT,
        source VARCHAR(100),
        author VARCHAR(100),
        published_at TIMESTAMPTZ,
        url TEXT,
        sentiment_score DOUBLE PRECISION,
        importance_score DOUBLE PRECISION,
        keywords JSONB,
        category VARCHAR(50),
        is_major BOOLEAN,
        ai_analyzed BOOLEAN,
        ai_model VARCHAR(50),
        ai_analyzed_at TIMESTAMPTZ,
        ai_sentiment VARCHAR(20),
        ai_risk_score INTEGER,
        ai_event_tags JSONB,
        ai_summary TEXT,
        ai_key_findings JSONB,
        fetched_at TIMESTAMPTZ
    )
    ON CONFLICT (stock_code, article_id) DO NOTHING;

    GET DIAGNOSTICS rows_inserted = ROW_COUNT;
    RAISE NOTICE 'News: % rows inserted', rows_inserted;
END $$;

-- ============================================================================
-- 3. Migrate Research Data
-- ============================================================================
DO $$
DECLARE
    v10_conn TEXT := 'host=localhost dbname=aegis_v10 user=wonny';
    rows_inserted INTEGER;
BEGIN
    RAISE NOTICE '=== Starting Research Migration ===';

    -- Insert data from v10.analysis.research to v14.data.research
    INSERT INTO data.research (
        stock_code,
        title,
        analyst,
        firm,
        target_price,
        opinion,
        published_at,
        summary,
        sentiment_score,
        source_url,
        created_at,
        updated_at
    )
    SELECT
        stock_code,
        title,
        analyst,
        firm,
        target_price,
        opinion,
        published_at,
        summary,
        sentiment_score,
        source_url,
        COALESCE(collected_at, NOW()),
        NOW()
    FROM dblink(
        v10_conn,
        'SELECT
            stock_code, title, analyst, firm, target_price, opinion,
            published_at, summary, sentiment_score, source_url, collected_at
         FROM analysis.research
         ORDER BY published_at DESC'
    ) AS t(
        stock_code VARCHAR(10),
        title VARCHAR(500),
        analyst VARCHAR(100),
        firm VARCHAR(100),
        target_price NUMERIC,
        opinion VARCHAR(20),
        published_at TIMESTAMPTZ,
        summary TEXT,
        sentiment_score NUMERIC,
        source_url VARCHAR(500),
        collected_at TIMESTAMPTZ
    )
    ON CONFLICT (stock_code, title, firm) DO NOTHING;

    GET DIAGNOSTICS rows_inserted = ROW_COUNT;
    RAISE NOTICE 'Research: % rows inserted', rows_inserted;
END $$;

-- ============================================================================
-- Summary
-- ============================================================================
DO $$
DECLARE
    consensus_count INTEGER;
    news_count INTEGER;
    research_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO consensus_count FROM data.consensus;
    SELECT COUNT(*) INTO news_count FROM data.news;
    SELECT COUNT(*) INTO research_count FROM data.research;

    RAISE NOTICE '=== Migration Complete ===';
    RAISE NOTICE 'Consensus records: %', consensus_count;
    RAISE NOTICE 'News records: %', news_count;
    RAISE NOTICE 'Research records: %', research_count;
END $$;
