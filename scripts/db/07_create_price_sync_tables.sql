-- ================================================================
-- Aegis v14 - PriceSync 모듈 테이블 생성
-- ================================================================
-- Module: PriceSync
-- Owner: market.prices_ticks, market.prices_best, market.freshness
-- Purpose: 가격 동기화 및 신선도 관리
-- ================================================================

-- 1. prices_ticks: 모든 가격 틱 데이터 (시계열)
-- ================================================================
DROP TABLE IF EXISTS market.prices_ticks CASCADE;

CREATE TABLE market.prices_ticks (
    id BIGSERIAL PRIMARY KEY,
    symbol CHAR(6) NOT NULL,
    source TEXT NOT NULL,  -- KIS_WS | KIS_REST | NAVER

    -- 가격 정보
    last_price BIGINT NOT NULL,  -- 현재가 (원 단위, 정수)
    change_price BIGINT,          -- 전일대비
    change_rate FLOAT,            -- 등락률 (%)
    volume BIGINT,                -- 거래량

    -- 호가 정보 (선택)
    bid_price BIGINT,
    ask_price BIGINT,
    bid_volume BIGINT,
    ask_volume BIGINT,

    -- 메타데이터
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- 수신 시각
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 인덱스
CREATE INDEX idx_prices_ticks_symbol_ts ON market.prices_ticks(symbol, ts DESC);
CREATE INDEX idx_prices_ticks_source_ts ON market.prices_ticks(source, ts DESC);
CREATE INDEX idx_prices_ticks_ts ON market.prices_ticks(ts DESC);

COMMENT ON TABLE market.prices_ticks IS 'PriceSync: 모든 가격 틱 데이터 (시계열)';
COMMENT ON COLUMN market.prices_ticks.source IS 'KIS_WS (실시간 WS) | KIS_REST (REST 폴링) | NAVER (Naver 증권)';
COMMENT ON COLUMN market.prices_ticks.last_price IS '현재가 (원 단위 정수, 예: 71000)';

-- TimescaleDB hypertable (선택, TimescaleDB 확장 설치 시)
-- SELECT create_hypertable('market.prices_ticks', 'ts', chunk_time_interval => INTERVAL '1 day');

-- ================================================================
-- 2. prices_best: 심볼별 최적 가격 (UPSERT로 1행 유지)
-- ================================================================
DROP TABLE IF EXISTS market.prices_best CASCADE;

CREATE TABLE market.prices_best (
    symbol CHAR(6) PRIMARY KEY,

    -- Best Price (신선도 기준 최적 소스 선택)
    best_price BIGINT NOT NULL,
    best_source TEXT NOT NULL,  -- KIS_WS | KIS_REST | NAVER
    best_ts TIMESTAMPTZ NOT NULL,

    -- 추가 정보
    change_price BIGINT,
    change_rate FLOAT,
    volume BIGINT,

    bid_price BIGINT,
    ask_price BIGINT,

    -- 상태
    is_stale BOOLEAN NOT NULL DEFAULT false,  -- 모든 소스가 stale 시 true

    -- 메타데이터
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prices_best_updated_ts ON market.prices_best(updated_ts DESC);
CREATE INDEX idx_prices_best_stale ON market.prices_best(is_stale, updated_ts DESC);

COMMENT ON TABLE market.prices_best IS 'PriceSync: 심볼별 최적 가격 (신선도 기준 선택)';
COMMENT ON COLUMN market.prices_best.is_stale IS '모든 소스가 stale(오래됨) 시 true';

-- ================================================================
-- 3. freshness: 심볼별 소스별 신선도 추적
-- ================================================================
DROP TABLE IF EXISTS market.freshness CASCADE;

CREATE TABLE market.freshness (
    symbol CHAR(6) NOT NULL,
    source TEXT NOT NULL,  -- KIS_WS | KIS_REST | NAVER

    -- 최근 수신 정보
    last_ts TIMESTAMPTZ,      -- 마지막 수신 시각
    last_price BIGINT,        -- 마지막 가격

    -- 신선도 판정
    is_stale BOOLEAN NOT NULL DEFAULT false,
    staleness_ms BIGINT,      -- 현재 - last_ts (밀리초)

    -- 품질 점수 (0~100)
    quality_score INT,

    -- 메타데이터
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (symbol, source)
);

CREATE INDEX idx_freshness_symbol ON market.freshness(symbol);
CREATE INDEX idx_freshness_stale ON market.freshness(is_stale, updated_ts DESC);
CREATE INDEX idx_freshness_quality ON market.freshness(quality_score DESC, symbol);

COMMENT ON TABLE market.freshness IS 'PriceSync: 심볼별 소스별 신선도 추적';
COMMENT ON COLUMN market.freshness.staleness_ms IS '현재 시각 - last_ts (밀리초)';
COMMENT ON COLUMN market.freshness.quality_score IS '0~100 점수 (높을수록 좋음)';

-- ================================================================
-- 4. (선택) sync_jobs: 동기화 작업 큐
-- ================================================================
DROP TABLE IF EXISTS market.sync_jobs CASCADE;

CREATE TABLE market.sync_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_type TEXT NOT NULL,  -- WS_SUBSCRIBE | REST_POLL | NAVER_FETCH
    symbol CHAR(6) NOT NULL,
    priority INT NOT NULL DEFAULT 0,  -- 우선순위 (높을수록 우선)

    -- 상태
    status TEXT NOT NULL DEFAULT 'PENDING',  -- PENDING | RUNNING | DONE | FAILED

    -- 실행 정보
    assigned_worker TEXT,     -- 작업을 처리하는 워커 ID
    started_ts TIMESTAMPTZ,
    completed_ts TIMESTAMPTZ,

    -- 에러 정보
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,

    -- 메타데이터
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_jobs_status_priority ON market.sync_jobs(status, priority DESC, created_ts);
CREATE INDEX idx_sync_jobs_symbol ON market.sync_jobs(symbol);

COMMENT ON TABLE market.sync_jobs IS 'PriceSync: 동기화 작업 큐 (FOR UPDATE SKIP LOCKED 패턴)';

-- ================================================================
-- 5. (선택) discrepancies: 가격 불일치 추적
-- ================================================================
DROP TABLE IF EXISTS market.discrepancies CASCADE;

CREATE TABLE market.discrepancies (
    id BIGSERIAL PRIMARY KEY,
    symbol CHAR(6) NOT NULL,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 가격 비교
    kis_price BIGINT NOT NULL,
    naver_price BIGINT NOT NULL,
    diff_pct FLOAT NOT NULL,  -- 차이 비율 (%)

    kis_source TEXT NOT NULL,  -- KIS_WS | KIS_REST

    -- 심각도
    severity TEXT NOT NULL,  -- LOW | MEDIUM | HIGH

    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_discrepancies_symbol_ts ON market.discrepancies(symbol, ts DESC);
CREATE INDEX idx_discrepancies_severity ON market.discrepancies(severity, ts DESC);

COMMENT ON TABLE market.discrepancies IS 'PriceSync: KIS vs Naver 가격 불일치 추적';
COMMENT ON COLUMN market.discrepancies.severity IS 'LOW (0.1~0.5%) | MEDIUM (0.5~1.0%) | HIGH (>1.0%)';

-- ================================================================
-- 샘플 데이터 (테스트용)
-- ================================================================

-- 삼성전자 예시 데이터
INSERT INTO market.prices_ticks (symbol, source, last_price, change_price, change_rate, volume, ts)
VALUES
    ('005930', 'KIS_WS', 71000, 500, 0.71, 10000, NOW() - INTERVAL '5 seconds'),
    ('005930', 'KIS_REST', 71000, 500, 0.71, 10000, NOW() - INTERVAL '10 seconds'),
    ('005930', 'NAVER', 71100, 600, 0.85, 10000, NOW() - INTERVAL '30 seconds');

-- Best Price (WS 우선)
INSERT INTO market.prices_best (symbol, best_price, best_source, best_ts, change_price, change_rate, volume, is_stale)
VALUES ('005930', 71000, 'KIS_WS', NOW() - INTERVAL '5 seconds', 500, 0.71, 10000, false);

-- Freshness
INSERT INTO market.freshness (symbol, source, last_ts, last_price, is_stale, staleness_ms, quality_score)
VALUES
    ('005930', 'KIS_WS', NOW() - INTERVAL '5 seconds', 71000, false, 5000, 95),
    ('005930', 'KIS_REST', NOW() - INTERVAL '10 seconds', 71000, false, 10000, 85),
    ('005930', 'NAVER', NOW() - INTERVAL '30 seconds', 71100, false, 30000, 60);

-- ================================================================
-- 권한 설정
-- ================================================================

-- aegis_v14: PriceSync 모듈이 사용하는 role
GRANT SELECT, INSERT, UPDATE, DELETE ON market.prices_ticks TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON market.prices_best TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON market.freshness TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON market.sync_jobs TO aegis_v14;
GRANT SELECT, INSERT, UPDATE, DELETE ON market.discrepancies TO aegis_v14;

GRANT USAGE, SELECT ON SEQUENCE market.prices_ticks_id_seq TO aegis_v14;
GRANT USAGE, SELECT ON SEQUENCE market.sync_jobs_id_seq TO aegis_v14;
GRANT USAGE, SELECT ON SEQUENCE market.discrepancies_id_seq TO aegis_v14;

-- ================================================================
-- 검증
-- ================================================================

-- 테이블 목록
SELECT
    schemaname,
    tablename,
    tableowner
FROM pg_tables
WHERE schemaname = 'market'
    AND tablename LIKE 'price%' OR tablename IN ('freshness', 'sync_jobs', 'discrepancies')
ORDER BY tablename;

-- 샘플 데이터 확인
SELECT * FROM market.prices_ticks ORDER BY ts DESC LIMIT 5;
SELECT * FROM market.prices_best LIMIT 5;
SELECT * FROM market.freshness ORDER BY symbol, quality_score DESC;

-- ================================================================
-- 완료
-- ================================================================
\echo ''
\echo '========================================='
\echo '✅ PriceSync 테이블 생성 완료'
\echo '========================================='
\echo ''
\echo 'Created tables:'
\echo '  - market.prices_ticks   (가격 틱 데이터)'
\echo '  - market.prices_best    (최적 가격)'
\echo '  - market.freshness      (신선도 추적)'
\echo '  - market.sync_jobs      (작업 큐)'
\echo '  - market.discrepancies  (불일치 추적)'
\echo ''
