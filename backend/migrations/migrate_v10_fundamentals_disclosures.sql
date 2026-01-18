-- Migration Script: v10 → v14 Fundamentals & Disclosures Migration
-- Description: Copy fundamentals and disclosures data from aegis_v10 to aegis_v14
--
-- Prerequisites:
--   - dblink extension must be enabled
--   - v10 database must be accessible
--
-- Usage:
--   psql -h localhost -U wonny -d aegis_v14 -f migrate_v10_fundamentals_disclosures.sql

-- Enable dblink extension if not already enabled
CREATE EXTENSION IF NOT EXISTS dblink;

-- ============================================================================
-- 1. Migrate Fundamentals Data
-- ============================================================================
DO $$
DECLARE
    v10_conn TEXT := 'host=localhost dbname=aegis_v10 user=wonny';
    rows_inserted INTEGER;
BEGIN
    RAISE NOTICE '=== Starting Fundamentals Migration ===';

    -- Insert data from v10.analysis.fundamentals to v14.data.fundamentals
    INSERT INTO data.fundamentals (
        stock_code,
        report_date,
        per,
        pbr,
        psr,
        roe,
        debt_ratio,
        revenue,
        operating_profit,
        net_profit,
        eps,
        bps,
        dps,
        created_at
    )
    SELECT
        TRIM(stock_code),              -- CHAR(6) → VARCHAR(20), trim spaces
        as_of_date,                    -- as_of_date → report_date
        per,
        pbr,
        psr,
        roe,
        debt_ratio,
        NULL,                          -- revenue (v10에 없음)
        NULL,                          -- operating_profit (v10에 없음)
        NULL,                          -- net_profit (v10에 없음)
        NULL,                          -- eps (v10의 consensus_eps는 컨센서스 값이므로 제외)
        NULL,                          -- bps (v10에 없음)
        dividend_per_share,            -- dividend_per_share → dps
        COALESCE(created_at, NOW())
    FROM dblink(
        v10_conn,
        'SELECT
            stock_code, as_of_date, per, pbr, psr, roe, debt_ratio,
            dividend_per_share, created_at
         FROM analysis.fundamentals
         ORDER BY as_of_date DESC'
    ) AS t(
        stock_code CHAR(6),
        as_of_date DATE,
        per NUMERIC(10,2),
        pbr NUMERIC(10,2),
        psr NUMERIC(10,2),
        roe NUMERIC(10,2),
        debt_ratio NUMERIC(10,2),
        dividend_per_share BIGINT,
        created_at TIMESTAMPTZ
    )
    ON CONFLICT (stock_code, report_date) DO NOTHING;

    GET DIAGNOSTICS rows_inserted = ROW_COUNT;
    RAISE NOTICE 'Fundamentals: % rows inserted', rows_inserted;
END $$;

-- ============================================================================
-- 2. Migrate Disclosures Data
-- ============================================================================
DO $$
DECLARE
    v10_conn TEXT := 'host=localhost dbname=aegis_v10 user=wonny';
    rows_inserted INTEGER;
BEGIN
    RAISE NOTICE '=== Starting Disclosures Migration ===';

    -- Insert data from v10.analysis.disclosures to v14.data.disclosures
    INSERT INTO data.disclosures (
        stock_code,
        disclosed_at,
        title,
        category,
        subcategory,
        content,
        url,
        dart_rcept_no,
        created_at
    )
    SELECT
        stock_code,
        rcept_dt::TIMESTAMPTZ,         -- DATE → TIMESTAMPTZ
        report_name,                   -- report_name → title
        report_type,                   -- report_type → category
        category,                      -- category → subcategory
        NULL,                          -- content (v10에 없음)
        url,
        rcept_no,                      -- rcept_no → dart_rcept_no
        COALESCE(created_at, NOW())
    FROM dblink(
        v10_conn,
        'SELECT
            stock_code, rcept_dt, report_name, report_type, category,
            url, rcept_no, created_at
         FROM analysis.disclosures
         ORDER BY rcept_dt DESC'
    ) AS t(
        stock_code VARCHAR(10),
        rcept_dt DATE,
        report_name TEXT,
        report_type VARCHAR(50),
        category VARCHAR(50),
        url TEXT,
        rcept_no VARCHAR(20),
        created_at TIMESTAMPTZ
    )
    ON CONFLICT DO NOTHING;

    GET DIAGNOSTICS rows_inserted = ROW_COUNT;
    RAISE NOTICE 'Disclosures: % rows inserted', rows_inserted;
END $$;

-- ============================================================================
-- Summary
-- ============================================================================
DO $$
DECLARE
    fundamentals_count INTEGER;
    disclosures_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO fundamentals_count FROM data.fundamentals;
    SELECT COUNT(*) INTO disclosures_count FROM data.disclosures;

    RAISE NOTICE '=== Migration Complete ===';
    RAISE NOTICE 'Fundamentals records: %', fundamentals_count;
    RAISE NOTICE 'Disclosures records: %', disclosures_count;
END $$;
