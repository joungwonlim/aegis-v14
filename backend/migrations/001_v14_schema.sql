-- V14 Schema Migration
-- 기존 v13 테이블을 v14 스키마로 변경

-- 1. order_intents 테이블 재생성 (v14 스타일)
DROP TABLE IF EXISTS trade.order_intents CASCADE;
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
DROP TABLE IF EXISTS trade.orders CASCADE;
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
DROP TABLE IF EXISTS trade.fills CASCADE;
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
