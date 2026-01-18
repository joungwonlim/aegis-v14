-- Database Tuning: VACUUM ANALYZE
-- Description: Update table statistics and clean up dead tuples
-- This improves query planning and performance

-- ============================================================================
-- 1. VACUUM ANALYZE 새로 마이그레이션한 테이블들
-- ============================================================================
\echo '=== Vacuum Analyze: Newly Migrated Tables ==='

VACUUM ANALYZE data.consensus;
\echo 'Completed: data.consensus'

VACUUM ANALYZE data.news;
\echo 'Completed: data.news'

VACUUM ANALYZE data.research;
\echo 'Completed: data.research'

VACUUM ANALYZE data.fundamentals;
\echo 'Completed: data.fundamentals'

VACUUM ANALYZE data.disclosures;
\echo 'Completed: data.disclosures'

-- ============================================================================
-- 2. VACUUM ANALYZE 주요 테이블들
-- ============================================================================
\echo '=== Vacuum Analyze: Main Tables ==='

VACUUM ANALYZE data.stocks;
\echo 'Completed: data.stocks'

-- Daily prices partitions (최근 파티션만)
VACUUM ANALYZE data.daily_prices_2026_h1;
VACUUM ANALYZE data.daily_prices_2025_h2;
VACUUM ANALYZE data.daily_prices_2025_h1;
\echo 'Completed: daily_prices partitions'

-- Investor flow partitions (최근 파티션만)
VACUUM ANALYZE data.investor_flow_2026_h1;
VACUUM ANALYZE data.investor_flow_2025_h2;
VACUUM ANALYZE data.investor_flow_2025_h1;
\echo 'Completed: investor_flow partitions'

-- Market tables
VACUUM ANALYZE market.prices_best;
VACUUM ANALYZE market.stocks;
\echo 'Completed: market tables'

-- Trade tables
VACUUM ANALYZE trade.holdings;
VACUUM ANALYZE trade.positions;
VACUUM ANALYZE trade.orders;
VACUUM ANALYZE trade.fills;
VACUUM ANALYZE trade.order_intents;
\echo 'Completed: trade tables'

-- Signals tables
VACUUM ANALYZE signals.factor_scores;
\echo 'Completed: signals tables'

-- Audit tables
VACUUM ANALYZE audit.daily_snapshots;
VACUUM ANALYZE audit.daily_pnl;
\echo 'Completed: audit tables'

\echo '=== VACUUM ANALYZE Complete ==='
