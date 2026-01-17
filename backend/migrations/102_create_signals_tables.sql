-- v13 → v14 마이그레이션: Signals 테이블 생성
-- v14.0.0
-- 2026-01-17

-- ================================================
-- 1. signals.factor_scores (6팩터 종합 점수)
-- SSOT: Signals 서비스만 쓰기 가능
-- ================================================
CREATE TABLE signals.factor_scores (
    stock_code      VARCHAR(20) NOT NULL,
    calc_date       DATE NOT NULL,

    -- 6가지 팩터 점수 (-1.0 ~ 1.0)
    momentum        NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    technical       NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    value           NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    quality         NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    flow            NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    event           NUMERIC(5,4) NOT NULL DEFAULT 0.0,

    -- 종합 점수 (가중 평균)
    total_score     NUMERIC(5,4),

    -- 메타
    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_factor_scores_date ON signals.factor_scores(calc_date);
CREATE INDEX idx_factor_scores_total ON signals.factor_scores(calc_date, total_score DESC);
CREATE INDEX idx_factor_scores_momentum ON signals.factor_scores(calc_date, momentum DESC);

COMMENT ON TABLE signals.factor_scores IS '6팩터 종합 점수 - Momentum/Technical/Value/Quality/Flow/Event';

-- ================================================
-- 2. signals.flow_details (수급 상세)
-- SSOT: Signals 서비스만 쓰기 가능
-- ================================================
CREATE TABLE signals.flow_details (
    stock_code          VARCHAR(20) NOT NULL,
    calc_date           DATE NOT NULL,

    -- 외국인 순매수 (5일/10일/20일)
    foreign_net_5d      BIGINT DEFAULT 0,
    foreign_net_10d     BIGINT DEFAULT 0,
    foreign_net_20d     BIGINT DEFAULT 0,

    -- 기관 순매수 (5일/10일/20일)
    inst_net_5d         BIGINT DEFAULT 0,
    inst_net_10d        BIGINT DEFAULT 0,
    inst_net_20d        BIGINT DEFAULT 0,

    -- 개인 순매수 (5일/10일/20일) - 역지표 참고용
    indiv_net_5d        BIGINT DEFAULT 0,
    indiv_net_10d       BIGINT DEFAULT 0,
    indiv_net_20d       BIGINT DEFAULT 0,

    -- 정규화된 점수
    foreign_score       NUMERIC(5,4) DEFAULT 0.0,
    inst_score          NUMERIC(5,4) DEFAULT 0.0,

    updated_at          TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_flow_details_date ON signals.flow_details(calc_date);
CREATE INDEX idx_flow_details_foreign ON signals.flow_details(calc_date, foreign_score DESC);

COMMENT ON TABLE signals.flow_details IS '수급 상세 - 5D/10D/20D 누적 순매수';

-- ================================================
-- 3. signals.technical_details (기술적 지표 상세)
-- SSOT: Signals 서비스만 쓰기 가능
-- ================================================
CREATE TABLE signals.technical_details (
    stock_code      VARCHAR(20) NOT NULL,
    calc_date       DATE NOT NULL,

    -- 이동평균선
    ma5             NUMERIC(12,2),
    ma10            NUMERIC(12,2),
    ma20            NUMERIC(12,2),
    ma60            NUMERIC(12,2),
    ma120           NUMERIC(12,2),

    -- RSI (14일)
    rsi14           NUMERIC(5,2),

    -- MACD (12, 26, 9)
    macd            NUMERIC(12,4),
    macd_signal     NUMERIC(12,4),
    macd_hist       NUMERIC(12,4),

    -- Bollinger Bands
    bb_upper        NUMERIC(12,2),
    bb_middle       NUMERIC(12,2),
    bb_lower        NUMERIC(12,2),

    -- EMA
    ema12           NUMERIC(12,2),
    ema26           NUMERIC(12,2),

    -- ATR (14일) - 변동성 지표
    atr14           NUMERIC(12,4),

    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_technical_details_date ON signals.technical_details(calc_date);
CREATE INDEX idx_technical_details_rsi ON signals.technical_details(calc_date, rsi14);

COMMENT ON TABLE signals.technical_details IS '기술적 지표 - MA/RSI/MACD/BB';

-- ================================================
-- 4. signals.momentum_details (모멘텀 상세)
-- SSOT: Signals 서비스만 쓰기 가능
-- ================================================
CREATE TABLE signals.momentum_details (
    stock_code      VARCHAR(20) NOT NULL,
    calc_date       DATE NOT NULL,

    -- 수익률
    return_5d       NUMERIC(10,6),      -- 5일 수익률
    return_20d      NUMERIC(10,6),      -- 20일 수익률 (1M)
    return_60d      NUMERIC(10,6),      -- 60일 수익률 (3M)

    -- 거래량
    volume_5d_avg   BIGINT,             -- 5일 평균 거래량
    volume_20d_avg  BIGINT,             -- 20일 평균 거래량
    volume_ratio    NUMERIC(10,4),      -- 거래량 비율 (5D/20D)

    -- 정규화된 점수
    price_momentum  NUMERIC(5,4) DEFAULT 0.0,
    volume_momentum NUMERIC(5,4) DEFAULT 0.0,

    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (stock_code, calc_date)
);

CREATE INDEX idx_momentum_details_date ON signals.momentum_details(calc_date);

COMMENT ON TABLE signals.momentum_details IS '모멘텀 상세 - 수익률/거래량 성장';

-- ================================================
-- 5. signals.event_signals (이벤트 시그널)
-- SSOT: Signals 서비스만 쓰기 가능
-- ================================================
CREATE TABLE signals.event_signals (
    id              SERIAL PRIMARY KEY,
    stock_code      VARCHAR(20) NOT NULL,
    event_date      DATE NOT NULL,

    -- 이벤트 분류
    event_type      VARCHAR(50) NOT NULL,       -- disclosure, news, earning
    event_subtype   VARCHAR(50),                -- positive, negative

    -- 이벤트 내용
    title           TEXT,
    description     TEXT,
    source          VARCHAR(100),               -- dart, naver, bloomberg

    -- 영향도 점수 (-1.0 ~ 1.0)
    impact_score    NUMERIC(5,4) DEFAULT 0.0,

    -- 시간 가중치 (현재 날짜 기준 계산)
    time_weight     NUMERIC(5,4) DEFAULT 1.0,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_event_signals_stock ON signals.event_signals(stock_code);
CREATE INDEX idx_event_signals_date ON signals.event_signals(event_date DESC);
CREATE INDEX idx_event_signals_type ON signals.event_signals(event_type, event_date DESC);
CREATE INDEX idx_event_signals_impact ON signals.event_signals(stock_code, event_date DESC, impact_score DESC);

COMMENT ON TABLE signals.event_signals IS '이벤트 시그널 - 공시/뉴스/실적 영향도';

-- ================================================
-- 이벤트 유형 참조 테이블 (선택적)
-- ================================================
CREATE TABLE signals.event_types (
    event_type      VARCHAR(50) PRIMARY KEY,
    category        VARCHAR(50) NOT NULL,       -- positive, negative, neutral
    default_score   NUMERIC(5,4) NOT NULL,      -- 기본 영향도
    description     TEXT
);

-- 기본 이벤트 유형 INSERT
INSERT INTO signals.event_types (event_type, category, default_score, description) VALUES
-- 긍정적 이벤트
('earnings_positive', 'positive', 1.0, '실적 개선'),
('merger_positive', 'positive', 0.9, '인수합병 긍정'),
('share_buyback', 'positive', 0.8, '자사주 매입'),
('new_product', 'positive', 0.7, '신제품 출시'),
('dividend_increase', 'positive', 0.6, '배당 증가'),
('partnership', 'positive', 0.6, '파트너십 체결'),
('capex_increase', 'positive', 0.5, '설비 투자'),
('patent', 'positive', 0.5, '특허 취득'),
-- 부정적 이벤트
('earnings_negative', 'negative', -1.0, '실적 악화'),
('audit_opinion', 'negative', -0.9, '감사 의견'),
('merger_negative', 'negative', -0.8, '인수합병 부정'),
('recall', 'negative', -0.8, '제품 리콜'),
('lawsuit', 'negative', -0.7, '소송'),
('regulatory', 'negative', -0.7, '규제 이슈'),
('dividend_decrease', 'negative', -0.6, '배당 감소'),
('management_change', 'negative', -0.5, '경영진 교체');

COMMENT ON TABLE signals.event_types IS '이벤트 유형별 기본 영향도';
