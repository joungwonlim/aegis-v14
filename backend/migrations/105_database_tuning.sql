-- Migration: 105_database_tuning.sql
-- Description: Database performance tuning and optimization
-- - Add missing indexes for common query patterns
-- - Set table statistics targets for better query planning
-- - Configure autovacuum settings for high-volume tables

-- ============================================================================
-- 1. 추가 인덱스 생성 (자주 사용되는 쿼리 패턴)
-- ============================================================================

-- Disclosures: stock_code + disclosed_at 복합 인덱스
CREATE INDEX IF NOT EXISTS idx_disclosures_stock_disclosed
ON data.disclosures(stock_code, disclosed_at DESC);

COMMENT ON INDEX data.idx_disclosures_stock_disclosed IS '종목별 최신 공시 조회 최적화';

-- Fundamentals: report_date 역순 인덱스 (최신 데이터 조회용)
CREATE INDEX IF NOT EXISTS idx_fundamentals_date_desc
ON data.fundamentals(report_date DESC);

COMMENT ON INDEX data.idx_fundamentals_date_desc IS '최신 재무 데이터 조회 최적화';

-- ============================================================================
-- 2. 테이블 통계 타겟 설정
-- ============================================================================

-- News 테이블: 데이터가 많으므로 통계 샘플 크기 증가
ALTER TABLE data.news ALTER COLUMN stock_code SET STATISTICS 1000;
ALTER TABLE data.news ALTER COLUMN published_at SET STATISTICS 1000;

-- Consensus 테이블: 통계 샘플 크기 증가
ALTER TABLE data.consensus ALTER COLUMN stock_code SET STATISTICS 500;
ALTER TABLE data.consensus ALTER COLUMN consensus_date SET STATISTICS 500;

-- Fundamentals 테이블: 통계 샘플 크기 증가
ALTER TABLE data.fundamentals ALTER COLUMN stock_code SET STATISTICS 500;
ALTER TABLE data.fundamentals ALTER COLUMN report_date SET STATISTICS 500;

-- Disclosures 테이블: 통계 샘플 크기 증가
ALTER TABLE data.disclosures ALTER COLUMN stock_code SET STATISTICS 500;
ALTER TABLE data.disclosures ALTER COLUMN disclosed_at SET STATISTICS 500;

-- ============================================================================
-- 3. Autovacuum 설정 (대용량 테이블)
-- ============================================================================

-- News 테이블: 삽입이 빈번하므로 autovacuum 더 자주 실행
ALTER TABLE data.news SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);

-- Daily prices 파티션들: 읽기 전용이므로 autovacuum 빈도 낮춤
ALTER TABLE data.daily_prices_2024_h1 SET (
    autovacuum_vacuum_scale_factor = 0.2,
    autovacuum_analyze_scale_factor = 0.1
);

ALTER TABLE data.daily_prices_2024_h2 SET (
    autovacuum_vacuum_scale_factor = 0.2,
    autovacuum_analyze_scale_factor = 0.1
);

-- ============================================================================
-- 4. Partial Indexes (조건부 인덱스)
-- ============================================================================

-- News: AI 미분석 뉴스만 인덱싱 (이미 존재하지만 확인)
-- (이미 idx_news_ai_analyzed가 있음)

-- Consensus: 최근 1년 데이터 인덱스
CREATE INDEX IF NOT EXISTS idx_consensus_recent
ON data.consensus(stock_code, consensus_date DESC)
WHERE consensus_date >= CURRENT_DATE - INTERVAL '1 year';

COMMENT ON INDEX data.idx_consensus_recent IS '최근 1년 컨센서스 데이터 빠른 조회';

-- Disclosures: 최근 6개월 공시 인덱스
CREATE INDEX IF NOT EXISTS idx_disclosures_recent
ON data.disclosures(stock_code, disclosed_at DESC)
WHERE disclosed_at >= NOW() - INTERVAL '6 months';

COMMENT ON INDEX data.idx_disclosures_recent IS '최근 6개월 공시 빠른 조회';

-- ============================================================================
-- 5. 인덱스 정리 완료 메시지
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '=== Database Tuning Complete ===';
    RAISE NOTICE 'Added indexes for:';
    RAISE NOTICE '  - Disclosures: stock_code + disclosed_at';
    RAISE NOTICE '  - Fundamentals: report_date DESC';
    RAISE NOTICE '  - Partial indexes for recent data';
    RAISE NOTICE 'Updated statistics targets for better query planning';
    RAISE NOTICE 'Configured autovacuum settings for high-volume tables';
END $$;
