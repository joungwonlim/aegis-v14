-- V14 Schema Migration (Clean Install)
-- WARNING: This will DROP existing exit engine tables and recreate them

-- Drop existing tables in reverse dependency order
DROP TABLE IF EXISTS trade.exit_signals CASCADE;
DROP TABLE IF EXISTS trade.symbol_exit_overrides CASCADE;
DROP TABLE IF EXISTS trade.exit_profiles CASCADE;
DROP TABLE IF EXISTS trade.exit_control CASCADE;
DROP TABLE IF EXISTS trade.position_state CASCADE;
DROP TABLE IF EXISTS trade.order_intents CASCADE;
DROP TABLE IF EXISTS trade.orders CASCADE;
DROP TABLE IF EXISTS trade.fills CASCADE;

-- 1. order_intents 테이블 재생성 (v14 스타일)
CREATE TABLE trade.order_intents (
    intent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    intent_type TEXT NOT NULL, -- EXIT_PARTIAL, EXIT_FULL
    qty BIGINT NOT NULL,
    order_type TEXT NOT NULL, -- MKT, LMT
    limit_price NUMERIC(20,4),
    reason_code TEXT NOT NULL, -- SL1, SL2, TP1, TP2, TRAILING
    action_key TEXT NOT NULL UNIQUE, -- Idempotency key
    status TEXT NOT NULL DEFAULT 'NEW', -- NEW, SUBMITTED, DUPLICATE, FAILED
    created_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_intents_position ON trade.order_intents(position_id);
CREATE INDEX idx_order_intents_status ON trade.order_intents(status);
CREATE INDEX idx_order_intents_created_ts ON trade.order_intents(created_ts DESC);

-- 2. orders 테이블 재생성 (v14 스타일)
CREATE TABLE trade.orders (
    order_id TEXT PRIMARY KEY, -- KIS order ID
    intent_id UUID REFERENCES trade.order_intents(intent_id),
    submitted_ts TIMESTAMP NOT NULL,
    status TEXT NOT NULL, -- SUBMITTED, PARTIAL, FILLED, CANCELLED, REJECTED
    broker_status TEXT NOT NULL,
    qty BIGINT NOT NULL,
    open_qty BIGINT NOT NULL,
    filled_qty BIGINT NOT NULL,
    raw JSONB,
    updated_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_intent_id ON trade.orders(intent_id);
CREATE INDEX idx_orders_status ON trade.orders(status);
CREATE INDEX idx_orders_submitted_ts ON trade.orders(submitted_ts DESC);

-- 3. fills 테이블 재생성 (v14 스타일)
CREATE TABLE trade.fills (
    fill_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id TEXT NOT NULL REFERENCES trade.orders(order_id),
    kis_exec_id TEXT NOT NULL,
    ts TIMESTAMP NOT NULL,
    qty BIGINT NOT NULL,
    price NUMERIC(20,4) NOT NULL,
    fee NUMERIC(20,4) NOT NULL DEFAULT 0,
    tax NUMERIC(20,4) NOT NULL DEFAULT 0,
    seq INTEGER NOT NULL,
    UNIQUE(order_id, kis_exec_id, seq)
);

CREATE INDEX idx_fills_order_id ON trade.fills(order_id);
CREATE INDEX idx_fills_ts ON trade.fills(ts DESC);
CREATE INDEX idx_fills_ts_seq ON trade.fills(ts DESC, seq DESC);

-- 4. position_state 테이블 (Exit Engine FSM 상태)
CREATE TABLE trade.position_state (
    position_id UUID PRIMARY KEY,
    phase TEXT NOT NULL DEFAULT 'OPEN',
    hwm_price NUMERIC(20,4),
    stop_floor_price NUMERIC(20,4),
    atr NUMERIC(20,4),
    cooldown_until TIMESTAMP,
    last_eval_ts TIMESTAMP,
    last_avg_price NUMERIC(20,4),
    breach_ticks INTEGER NOT NULL DEFAULT 0,
    updated_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_position_state_phase ON trade.position_state(phase);

-- 5. exit_control 테이블 (Kill Switch - Singleton)
CREATE TABLE trade.exit_control (
    id INTEGER PRIMARY KEY DEFAULT 1,
    mode TEXT NOT NULL DEFAULT 'RUNNING',
    reason TEXT,
    updated_by TEXT NOT NULL,
    updated_ts TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT exit_control_singleton CHECK (id = 1)
);

INSERT INTO trade.exit_control (id, mode, updated_by)
VALUES (1, 'RUNNING', 'system')
ON CONFLICT (id) DO NOTHING;

-- 6. exit_profiles 테이블 (Exit 규칙 프로필)
CREATE TABLE trade.exit_profiles (
    profile_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT NOT NULL,
    created_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exit_profiles_is_active ON trade.exit_profiles(is_active);

-- Default profile 초기화
INSERT INTO trade.exit_profiles (profile_id, name, description, config, created_by)
VALUES (
    'default',
    'Default Exit Profile',
    'Conservative exit strategy with ATR scaling',
    '{
        "atr": {"ref": 0.02, "factor_min": 0.7, "factor_max": 1.6},
        "sl1": {"base_pct": -0.03, "min_pct": -0.025, "max_pct": -0.05, "qty_pct": 0.5},
        "sl2": {"base_pct": -0.07, "min_pct": -0.05, "max_pct": -0.10, "qty_pct": 1.0},
        "tp1": {"base_pct": 0.05, "min_pct": 0.03, "max_pct": 0.08, "qty_pct": 0.1, "stop_floor_profit": 0.01},
        "tp2": {"base_pct": 0.10, "min_pct": 0.07, "max_pct": 0.15, "qty_pct": 0.2},
        "tp3": {"base_pct": 0.15, "min_pct": 0.12, "max_pct": 0.20, "qty_pct": 0.3, "start_trailing": true},
        "trailing": {"pct_trail": 0.04, "atr_k": 2.0},
        "time_stop": {"max_hold_days": 10, "no_momentum_days": 3, "no_momentum_profit": 0.02},
        "hardstop": {"enabled": true, "pct": -0.10}
    }'::jsonb,
    'system'
)
ON CONFLICT (profile_id) DO NOTHING;

-- 7. symbol_exit_overrides 테이블 (종목별 Override)
CREATE TABLE trade.symbol_exit_overrides (
    symbol TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES trade.exit_profiles(profile_id),
    enabled BOOLEAN NOT NULL DEFAULT true,
    effective_from TIMESTAMP,
    reason TEXT,
    created_by TEXT NOT NULL,
    created_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_symbol_exit_overrides_enabled ON trade.symbol_exit_overrides(enabled);

-- 8. exit_signals 테이블 (디버깅/백테스트 신호 기록)
CREATE TABLE trade.exit_signals (
    signal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id UUID NOT NULL,
    rule_name TEXT NOT NULL,
    is_triggered BOOLEAN NOT NULL,
    reason TEXT,
    distance NUMERIC(20,4),
    price NUMERIC(20,4) NOT NULL,
    evaluated_ts TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exit_signals_position ON trade.exit_signals(position_id);
CREATE INDEX idx_exit_signals_evaluated_ts ON trade.exit_signals(evaluated_ts DESC);

-- 9. 추가 인덱스 (getActiveIntents 최적화)
CREATE INDEX idx_order_intents_position_status ON trade.order_intents(position_id, status);
