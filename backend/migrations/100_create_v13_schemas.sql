-- v13 → v14 마이그레이션: 스키마 생성
-- v14.0.0
-- 2026-01-17

-- ================================================
-- 1. DATA Schema (Fetcher 모듈 소유)
-- ================================================
CREATE SCHEMA IF NOT EXISTS data;

COMMENT ON SCHEMA data IS 'v13 Fetcher 모듈 - 종목/가격/재무/공시 데이터';

-- ================================================
-- 2. SIGNALS Schema (Signals 모듈 소유)
-- ================================================
CREATE SCHEMA IF NOT EXISTS signals;

COMMENT ON SCHEMA signals IS 'v13 Signals 모듈 - 6팩터 시그널 점수';

-- ================================================
-- 3. AUDIT Schema (Audit 모듈 소유)
-- ================================================
CREATE SCHEMA IF NOT EXISTS audit;

COMMENT ON SCHEMA audit IS 'v13 Audit 모듈 - 성과 분석 및 리스크 모니터링';
