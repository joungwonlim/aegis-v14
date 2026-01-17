-- v13 → v14 마이그레이션: Audit 테이블 생성
-- v14.0.0
-- 2026-01-17

-- ================================================
-- 1. audit.performance_reports (성과 보고서)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.performance_reports (
    report_date         DATE PRIMARY KEY,
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,
    period_code         VARCHAR(10) NOT NULL,       -- 1M, 3M, 6M, 1Y, YTD

    -- 수익률 지표
    total_return        NUMERIC(10,6),              -- 누적 수익률
    annual_return       NUMERIC(10,6),              -- 연환산 수익률

    -- 벤치마크 비교
    benchmark_return    NUMERIC(10,6),              -- 벤치마크 수익률
    alpha               NUMERIC(10,6),              -- 초과 수익률
    beta                NUMERIC(10,6),              -- 시장 민감도

    -- 리스크 지표
    volatility          NUMERIC(10,6),              -- 연환산 변동성
    sharpe_ratio        NUMERIC(10,6),              -- 샤프 비율
    sortino_ratio       NUMERIC(10,6),              -- 소르티노 비율
    max_drawdown        NUMERIC(10,6),              -- 최대 낙폭 (음수)

    -- 트레이딩 지표
    win_rate            NUMERIC(5,4),               -- 승률
    avg_win             NUMERIC(15,2),              -- 평균 이익 (원)
    avg_loss            NUMERIC(15,2),              -- 평균 손실 (원)
    profit_factor       NUMERIC(10,6),              -- 수익 팩터
    total_trades        INT,                        -- 총 거래 수

    -- 메타
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_performance_period ON audit.performance_reports(period_code, report_date DESC);

COMMENT ON TABLE audit.performance_reports IS '성과 보고서 - 수익률/리스크/트레이딩 지표';

-- ================================================
-- 2. audit.attribution_analysis (귀속 분석)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.attribution_analysis (
    analysis_date       DATE PRIMARY KEY,
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,

    -- 전체 수익률
    total_return        NUMERIC(10,6),

    -- 팩터별 기여도 (JSONB)
    -- 예: [{"factor": "momentum", "contribution": 0.05, ...}, ...]
    factor_contrib      JSONB,

    -- 섹터별 기여도 (JSONB)
    -- 예: {"전기전자": 0.05, "금융": -0.02, ...}
    sector_contrib      JSONB,

    -- 종목별 기여도 (JSONB)
    -- 예: {"005930": 0.03, "000660": 0.02, ...}
    stock_contrib       JSONB,

    -- 메타
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_attribution_date ON audit.attribution_analysis(analysis_date DESC);

COMMENT ON TABLE audit.attribution_analysis IS '귀속 분석 - 팩터/섹터/종목별 기여도';

-- ================================================
-- 3. audit.benchmark_data (벤치마크 데이터)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.benchmark_data (
    benchmark_date      DATE NOT NULL,
    benchmark_code      VARCHAR(20) NOT NULL,       -- KOSPI, KOSDAQ, KOSPI200
    close_price         NUMERIC(12,2) NOT NULL,     -- 종가
    daily_return        NUMERIC(10,6),              -- 일간 수익률
    cumulative_return   NUMERIC(10,6),              -- 누적 수익률 (연초 대비)
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (benchmark_date, benchmark_code)
);

CREATE INDEX idx_benchmark_code ON audit.benchmark_data(benchmark_code, benchmark_date DESC);

COMMENT ON TABLE audit.benchmark_data IS '벤치마크 데이터 - KOSPI/KOSDAQ';

-- ================================================
-- 4. audit.daily_pnl (일별 손익)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.daily_pnl (
    pnl_date            DATE PRIMARY KEY,

    -- 손익
    realized_pnl        BIGINT DEFAULT 0,           -- 실현 손익 (원)
    unrealized_pnl      BIGINT DEFAULT 0,           -- 미실현 손익 (원)
    total_pnl           BIGINT,                     -- 총 손익 (원)

    -- 수익률
    daily_return        NUMERIC(10,6),              -- 일간 수익률
    cumulative_return   NUMERIC(10,6),              -- 누적 수익률

    -- 포트폴리오 상태
    portfolio_value     BIGINT,                     -- 포트폴리오 총 가치 (원)
    cash_balance        BIGINT,                     -- 현금 잔고 (원)
    position_count      INT,                        -- 보유 종목 수

    -- 메타
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_daily_pnl_date ON audit.daily_pnl(pnl_date DESC);

COMMENT ON TABLE audit.daily_pnl IS '일별 손익 - 실현/미실현 PnL';

-- ================================================
-- 5. audit.trade_history (거래 내역)
-- SSOT: Audit 서비스만 쓰기 가능 (집계용)
-- ================================================
CREATE TABLE audit.trade_history (
    trade_id            UUID PRIMARY KEY,
    stock_code          VARCHAR(20) NOT NULL,
    stock_name          VARCHAR(200),

    -- 진입
    entry_date          DATE NOT NULL,
    entry_price         NUMERIC(12,2) NOT NULL,
    entry_qty           BIGINT NOT NULL,
    entry_reason        VARCHAR(100),               -- 진입 사유

    -- 청산 (NULL이면 미청산)
    exit_date           DATE,
    exit_price          NUMERIC(12,2),
    exit_qty            BIGINT,
    exit_reason         VARCHAR(100),               -- SL1, SL2, TP1, TP2, TP3, TRAIL, TIME, MANUAL

    -- 손익
    realized_pnl        NUMERIC(15,2),              -- 실현 손익 (원)
    realized_pnl_pct    NUMERIC(10,6),              -- 실현 손익률

    -- 보유 기간
    holding_days        INT,                        -- 보유일 수

    -- 팩터 점수 (진입 시점)
    entry_momentum      NUMERIC(5,4),
    entry_technical     NUMERIC(5,4),
    entry_value         NUMERIC(5,4),
    entry_quality       NUMERIC(5,4),
    entry_flow          NUMERIC(5,4),
    entry_event         NUMERIC(5,4),
    entry_total_score   NUMERIC(5,4),

    -- 메타
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_trade_history_stock ON audit.trade_history(stock_code);
CREATE INDEX idx_trade_history_entry ON audit.trade_history(entry_date DESC);
CREATE INDEX idx_trade_history_exit ON audit.trade_history(exit_date DESC) WHERE exit_date IS NOT NULL;
CREATE INDEX idx_trade_history_exit_reason ON audit.trade_history(exit_reason, exit_date DESC);

COMMENT ON TABLE audit.trade_history IS '거래 내역 - 진입/청산/손익 기록';

-- ================================================
-- 6. audit.daily_snapshots (일별 스냅샷)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.daily_snapshots (
    date                DATE PRIMARY KEY,
    total_value         BIGINT NOT NULL,            -- 총 포트폴리오 가치 (원)
    cash                BIGINT NOT NULL,            -- 현금 잔고 (원)
    positions           JSONB,                      -- 포지션 상세 (PositionSnapshot[])
    daily_return        NUMERIC(10,6),              -- 일간 수익률
    cum_return          NUMERIC(10,6),              -- 누적 수익률
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_daily_snapshots_date ON audit.daily_snapshots(date DESC);

COMMENT ON TABLE audit.daily_snapshots IS '일별 포트폴리오 스냅샷 - 시점별 상태 기록';

-- ================================================
-- 7. audit.risk_metrics (리스크 메트릭스)
-- SSOT: Audit 서비스만 쓰기 가능
-- ================================================
CREATE TABLE audit.risk_metrics (
    metric_date         DATE PRIMARY KEY,

    -- 포트폴리오 리스크
    portfolio_var_95    NUMERIC(15,2),              -- VaR 95% (원)
    portfolio_var_99    NUMERIC(15,2),              -- VaR 99% (원)
    concentration       NUMERIC(5,4),               -- 집중도 (HHI)

    -- 시장 리스크
    market_correlation  NUMERIC(5,4),               -- 시장 상관관계
    sector_exposure     JSONB,                      -- 섹터 노출도

    -- 유동성 리스크
    avg_turnover_ratio  NUMERIC(10,6),              -- 평균 회전율
    illiquid_weight     NUMERIC(5,4),               -- 비유동 종목 비중

    -- 메타
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_risk_metrics_date ON audit.risk_metrics(metric_date DESC);

COMMENT ON TABLE audit.risk_metrics IS '리스크 메트릭스 - VaR/집중도/유동성';

-- ================================================
-- 8. audit.trades (거래 요약 - 뷰)
-- Repository에서 사용하는 거래 데이터
-- ================================================
CREATE VIEW audit.trades AS
SELECT
    trade_id::TEXT as id,
    stock_code as symbol,
    CASE WHEN exit_date IS NOT NULL THEN 'SELL' ELSE 'BUY' END as side,
    entry_qty::INT as quantity,
    entry_price::BIGINT as price,
    COALESCE(realized_pnl, 0) as pnl,
    COALESCE(realized_pnl_pct, 0) as pnl_percent,
    entry_date,
    COALESCE(exit_date, entry_date) as exit_date,
    COALESCE(holding_days, 0) as hold_days
FROM audit.trade_history
WHERE exit_date IS NOT NULL;

COMMENT ON VIEW audit.trades IS '거래 요약 뷰 - 청산 완료된 거래만';

-- ================================================
-- 9. audit.job_runs (작업 실행 기록)
-- SSOT: 모든 모듈 쓰기 가능
-- ================================================
CREATE TABLE audit.job_runs (
    run_id              VARCHAR(100) PRIMARY KEY,   -- 예: run_20260117_153000
    job_type            VARCHAR(100) NOT NULL,      -- fetch_prices, calc_signals, etc.
    started_at          TIMESTAMPTZ NOT NULL,
    completed_at        TIMESTAMPTZ,
    status              VARCHAR(20) NOT NULL,       -- running, success, failed
    records_processed   INT,
    records_success     INT,
    records_failed      INT,
    error_message       TEXT,
    metadata            JSONB,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_runs_type ON audit.job_runs(job_type, started_at DESC);
CREATE INDEX idx_job_runs_status ON audit.job_runs(status, started_at DESC);

COMMENT ON TABLE audit.job_runs IS '작업 실행 기록 - 재현성 보장';
