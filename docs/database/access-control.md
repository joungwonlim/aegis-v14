# 데이터베이스 접근 제어 설계

> PostgreSQL Role 기반 접근 제어 (RBAC)

---

## 📐 Access Control Map

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                             PostgreSQL Roles                                     │
├──────────────┬───────────────┬──────────────┬──────────────┬──────────────┬─────┤
│ aegis_admin  │ aegis_datasync│ aegis_price  │ aegis_router │ aegis_trade  │ ... │
│ (슈퍼관리자)  │ (종목마스터)   │ (PriceSync)  │ (Pick Router)│ (Exit/Reentry)│    │
└──────┬───────┴───────┬───────┴──────┬───────┴──────┬───────┴──────┬───────┴─────┘
       │               │              │              │              │
       ▼               ▼              ▼              ▼              ▼
   ALL ACCESS    market.stocks  market.prices* trade.picks    trade.positions
                 (READ/WRITE)   (READ/WRITE)  (READ/WRITE)   (READ/WRITE)
```

---

## 🎯 설계 원칙

### 1. 최소 권한 원칙 (Principle of Least Privilege)

각 모듈은 **자신의 책임 범위에만 쓰기 권한**을 가짐:
- PriceSync → `market.*` 테이블만 쓰기
- Exit → `trade.positions`, `trade.position_state` 쓰기
- Reentry → `trade.reentry_candidates` 쓰기
- Execution → `trade.orders`, `trade.fills` 쓰기

### 2. SSOT 강제 (Database Level Enforcement)

**문제**: 코드 레벨 SSOT 규칙은 실수로 위반 가능
**해결**: PostgreSQL GRANT/REVOKE로 DB 레벨 강제

```sql
-- ❌ 금지: Exit Engine이 market.prices_best 수정
REVOKE UPDATE, DELETE ON market.prices_best FROM aegis_trade;

-- ✅ 허용: Exit Engine이 market.prices_best 읽기
GRANT SELECT ON market.prices_best TO aegis_trade;
```

### 3. Role 계층 구조

```
aegis_admin (슈퍼관리자)
├── aegis_datasync (DataSync 전용)
│   └── market.stocks (READ/WRITE)
│
├── aegis_price (PriceSync 전용)
│   ├── market.prices_*, freshness, sync_jobs, discrepancies (READ/WRITE)
│   ├── market.stocks (READ ONLY)
│   └── trade.* (READ ONLY)
│
├── aegis_router (Router 전용)
│   ├── market.* (READ ONLY)
│   ├── trade.picks (READ/WRITE)
│   ├── trade.pick_decisions (READ/WRITE)
│   └── trade.order_intents (READ/WRITE, ENTRY only)
│
├── aegis_trade (Strategy 전용: Exit/Reentry)
│   ├── market.* (READ ONLY)
│   ├── trade.positions (READ/WRITE, 일부 컬럼)
│   ├── trade.position_state (READ/WRITE)
│   ├── trade.reentry_candidates (READ/WRITE)
│   └── trade.order_intents (READ/WRITE, EXIT_*/ENTRY)
│
├── aegis_exec (Execution 전용)
│   ├── market.stocks, prices_best (READ ONLY)
│   ├── trade.order_intents (READ ONLY)
│   ├── trade.orders (READ/WRITE)
│   ├── trade.fills (READ/WRITE)
│   ├── trade.holdings (READ/WRITE)
│   ├── trade.exit_events (READ/WRITE)
│   └── trade.positions (UPDATE ONLY, qty/avg_price 컬럼만)
│
└── aegis_readonly (조회 전용)
    └── ALL TABLES (SELECT ONLY)
```

---

## 📊 Role 정의

### 1. aegis_admin (슈퍼관리자)

**목적**: 스키마 생성, 마이그레이션, 긴급 복구

```sql
CREATE ROLE aegis_admin WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    SUPERUSER
    CREATEDB
    CREATEROLE
    REPLICATION;

COMMENT ON ROLE aegis_admin IS '슈퍼관리자 - 스키마 생성/마이그레이션 전용';
```

**사용 시점**:
- 초기 스키마 생성
- 마이그레이션 실행
- 긴급 데이터 복구
- Role 생성/변경

**⚠️ 주의**: 애플리케이션 코드에서 사용 금지!

---

### 2. aegis_datasync (DataSync 모듈)

**목적**: 종목 마스터 데이터 동기화 (KIS API → market.stocks)

```sql
-- Role 생성
CREATE ROLE aegis_datasync WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_datasync IS 'DataSync 모듈 전용 - market.stocks 쓰기 권한';

-- market schema 권한
GRANT USAGE ON SCHEMA market TO aegis_datasync;

-- market.stocks 전체 권한 (SSOT 소유자)
GRANT SELECT, INSERT, UPDATE, DELETE ON market.stocks TO aegis_datasync;

-- 기본 권한 설정 (향후 생성되는 테이블 대비)
ALTER DEFAULT PRIVILEGES IN SCHEMA market
    GRANT SELECT ON TABLES TO aegis_datasync;
```

**쓰기 가능 테이블**:
- ✅ `market.stocks` (종목 마스터 - SSOT)

**중요**:
- DataSync는 `market.stocks`의 유일한 쓰기 소유자
- 다른 모든 모듈은 `market.stocks` READ ONLY
- KIS API 종목 정보를 주기적으로 동기화 (상장/폐지/거래정지 등)
- is_tradable 플래그 관리 (거래 가능 여부)

---

### 3. aegis_price (PriceSync 모듈)

**목적**: 가격 데이터 수집 및 저장

```sql
-- Role 생성
CREATE ROLE aegis_price WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_price IS 'PriceSync 모듈 전용 - market.* 쓰기 권한';

-- market schema 권한 (READ/WRITE)
GRANT USAGE ON SCHEMA market TO aegis_price;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA market TO aegis_price;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA market TO aegis_price;

-- market.stocks는 READ ONLY (DataSync 소유)
REVOKE INSERT, UPDATE, DELETE ON market.stocks FROM aegis_price;

-- trade schema 권한 (READ ONLY)
GRANT USAGE ON SCHEMA trade TO aegis_price;
GRANT SELECT ON ALL TABLES IN SCHEMA trade TO aegis_price;

-- 기본 권한 설정 (향후 생성되는 테이블에도 적용)
ALTER DEFAULT PRIVILEGES IN SCHEMA market
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_price;

ALTER DEFAULT PRIVILEGES IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_price;
```

**쓰기 가능 테이블**:
- ✅ `market.prices_ticks`
- ✅ `market.prices_best`
- ✅ `market.freshness`
- ✅ `market.sync_jobs` (작업 큐)
- ✅ `market.discrepancies` (가격 불일치 기록)

**읽기 전용 테이블**:
- 👁️ `trade.*` (WS 구독 대상 결정용)

---

### 4. aegis_router (Router 모듈: Pick 처리)

**목적**: 종목 선정 결과 수신 및 ENTRY intent 생성

```sql
-- Role 생성
CREATE ROLE aegis_router WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_router IS 'Router 모듈 - picks/pick_decisions/order_intents(ENTRY) 쓰기';

-- market schema 권한 (READ ONLY)
GRANT USAGE ON SCHEMA market TO aegis_router;
GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_router;

-- market.stocks READ ONLY (신선도 체크, 종목 유효성 검증)
GRANT SELECT ON market.stocks TO aegis_router;

-- trade schema 권한
GRANT USAGE ON SCHEMA trade TO aegis_router;

-- 쓰기 가능 테이블 (SSOT 소유)
GRANT SELECT, INSERT, UPDATE ON trade.picks TO aegis_router;
GRANT SELECT, INSERT, UPDATE ON trade.pick_decisions TO aegis_router;
GRANT SELECT, INSERT, UPDATE ON trade.order_intents TO aegis_router;  -- ENTRY only

-- 읽기 전용 테이블
GRANT SELECT ON trade.positions TO aegis_router;
GRANT SELECT ON trade.exit_events TO aegis_router;
GRANT SELECT ON trade.orders TO aegis_router;
GRANT SELECT ON trade.fills TO aegis_router;
GRANT SELECT ON trade.holdings TO aegis_router;

-- Sequence 권한
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_router;
```

**쓰기 가능 테이블**:
- ✅ `trade.picks` (선정 결과 수신)
- ✅ `trade.pick_decisions` (충돌 해결 결과)
- ✅ `trade.order_intents` (ENTRY 타입만)

**읽기 전용 테이블**:
- 👁️ `market.*` (신선도 체크)
- 👁️ `trade.positions` (중복 진입 방지)
- 👁️ `trade.holdings` (실제 보유 확인)

---

### 5. aegis_trade (Exit/Reentry 모듈)

**목적**: 포지션 관리 및 청산/재진입 로직

```sql
-- Role 생성
CREATE ROLE aegis_trade WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_trade IS 'Exit/Reentry 모듈 - positions/order_intents(EXIT_*/ENTRY) 쓰기';

-- market schema 권한 (READ ONLY)
GRANT USAGE ON SCHEMA market TO aegis_trade;
GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_trade;

-- market.stocks READ ONLY (종목 유효성 검증)
GRANT SELECT ON market.stocks TO aegis_trade;

-- trade schema 권한
GRANT USAGE ON SCHEMA trade TO aegis_trade;

-- 쓰기 가능 테이블 (Exit Engine SSOT)
-- positions: 컬럼별 권한 (SSOT: Exit는 전략 상태만, Execution이 qty/avg_price 소유)
GRANT SELECT, INSERT ON trade.positions TO aegis_trade;
GRANT UPDATE (status, exit_mode, exit_profile_id, updated_ts) ON trade.positions TO aegis_trade;

GRANT SELECT, INSERT, UPDATE ON trade.position_state TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.order_intents TO aegis_trade;  -- EXIT_*, ENTRY (reentry)
GRANT SELECT, INSERT, UPDATE ON trade.exit_signals TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.exit_control TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.exit_profiles TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.symbol_exit_overrides TO aegis_trade;

-- 쓰기 가능 테이블 (Reentry Engine SSOT)
GRANT SELECT, INSERT, UPDATE ON trade.reentry_candidates TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.reentry_control TO aegis_trade;

-- 읽기 전용 테이블
GRANT SELECT ON trade.picks TO aegis_trade;
GRANT SELECT ON trade.pick_decisions TO aegis_trade;
GRANT SELECT ON trade.exit_events TO aegis_trade;  -- Reentry가 소비
GRANT SELECT ON trade.orders TO aegis_trade;
GRANT SELECT ON trade.fills TO aegis_trade;
GRANT SELECT ON trade.holdings TO aegis_trade;

-- Sequence 권한
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_trade;
```

**쓰기 가능 테이블**:
- ✅ `trade.positions` (포지션 마스터 - Exit 소유)
- ✅ `trade.position_state` (Exit FSM 상태)
- ✅ `trade.reentry_candidates` (Reentry FSM 상태)
- ✅ `trade.order_intents` (EXIT_*, ENTRY for reentry)
- ✅ `trade.exit_signals` (Exit 트리거 평가 기록)
- ✅ `trade.exit_control` (Exit 전역 제어)
- ✅ `trade.exit_profiles` (Exit 프로파일)
- ✅ `trade.symbol_exit_overrides` (종목별 Exit 오버라이드)
- ✅ `trade.reentry_control` (Reentry 전역 제어)

**읽기 전용 테이블**:
- 👁️ `market.*` (현재가 조회)
- 👁️ `trade.orders` (주문 상태 확인)
- 👁️ `trade.fills` (체결 내역 확인)
- 👁️ `trade.holdings` (KIS 보유 현황 조회)

**중요 노트**:
- 👁️ `trade.picks`, `trade.pick_decisions` - READ ONLY (Router가 소유)
- 👁️ `trade.exit_events` - READ ONLY (Execution이 소유, Reentry가 소비)

---

### 6. aegis_exec (Execution 모듈)

**목적**: 주문 제출 및 체결 관리

```sql
-- Role 생성
CREATE ROLE aegis_exec WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_exec IS 'Execution 모듈 - 주문/체결 쓰기 권한';

-- market schema 권한 (READ ONLY, 선택적)
GRANT USAGE ON SCHEMA market TO aegis_exec;
GRANT SELECT ON market.prices_best TO aegis_exec;
GRANT SELECT ON market.stocks TO aegis_exec;

-- trade schema 권한
GRANT USAGE ON SCHEMA trade TO aegis_exec;

-- 쓰기 가능 테이블 (SSOT 소유)
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.orders TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.fills TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.holdings TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.exit_events TO aegis_exec;

-- 읽기 전용 테이블
GRANT SELECT ON trade.order_intents TO aegis_exec;
GRANT SELECT ON trade.positions TO aegis_exec;

-- 특별 권한: positions 체결 동기화 (SSOT: Execution이 qty/avg_price 소유)
GRANT UPDATE (qty, avg_price, updated_ts) ON trade.positions TO aegis_exec;

-- Sequence 권한
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_exec;
```

**쓰기 가능 테이블**:
- ✅ `trade.orders` (주문 상태)
- ✅ `trade.fills` (체결 내역)
- ✅ `trade.holdings` (KIS 보유종목 현황)
- ✅ `trade.exit_events` (청산 확정 이벤트 생성 - SSOT)
- ✅ `trade.positions` (qty 컬럼만 UPDATE)

**읽기 전용 테이블**:
- 👁️ `market.prices_best` (주문 가격 참조, 선택)
- 👁️ `trade.order_intents` (주문 의도 읽기)
- 👁️ `trade.positions` (포지션 정보)

---

### 7. aegis_readonly (조회 전용)

**목적**: BFF API 조회, 모니터링, 대시보드

```sql
-- Role 생성
CREATE ROLE aegis_readonly WITH
    LOGIN
    PASSWORD 'CHANGE_ME'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_readonly IS '조회 전용 - 모든 테이블 SELECT만 가능';

-- 모든 schema 읽기 권한
GRANT USAGE ON SCHEMA market TO aegis_readonly;
GRANT USAGE ON SCHEMA trade TO aegis_readonly;

-- 모든 테이블 SELECT 권한
GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA trade TO aegis_readonly;

-- 향후 생성되는 테이블에도 적용
ALTER DEFAULT PRIVILEGES IN SCHEMA market
    GRANT SELECT ON TABLES TO aegis_readonly;

ALTER DEFAULT PRIVILEGES IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_readonly;
```

**사용 시점**:
- BFF API 조회 엔드포인트
- Grafana 대시보드
- 데이터 분석 도구
- 수동 쿼리 (psql)

---

## 🔒 접근 제어 매트릭스

| 테이블 | aegis_admin | aegis_datasync | aegis_price | aegis_router | aegis_trade | aegis_exec | aegis_readonly |
|--------|-------------|----------------|-------------|--------------|-------------|------------|----------------|
| **market.stocks** | ALL | READ/WRITE | READ | READ | READ | READ | READ |
| **market.prices_ticks** | ALL | - | READ/WRITE | - | READ | - | READ |
| **market.prices_best** | ALL | - | READ/WRITE | - | READ | READ | READ |
| **market.freshness** | ALL | - | READ/WRITE | - | READ | - | READ |
| **market.sync_jobs** | ALL | - | READ/WRITE | - | - | - | READ |
| **market.discrepancies** | ALL | - | READ/WRITE | - | - | - | READ |
| **trade.positions** | ALL | - | - | READ | READ/WRITE(일부) | READ + UPDATE(qty) | READ |
| **trade.position_state** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.reentry_candidates** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.reentry_control** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.order_intents** | ALL | - | - | READ/WRITE | READ/WRITE | READ | READ |
| **trade.orders** | ALL | - | - | READ | READ | READ/WRITE | READ |
| **trade.fills** | ALL | - | - | READ | READ | READ/WRITE | READ |
| **trade.exit_signals** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.exit_control** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.exit_profiles** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.symbol_exit_overrides** | ALL | - | - | - | READ/WRITE | READ | READ |
| **trade.exit_events** | ALL | - | - | - | READ | READ/WRITE | READ |
| **trade.holdings** | ALL | - | - | READ | READ | READ/WRITE | READ |
| **trade.picks** | ALL | - | - | READ/WRITE | READ | READ | READ |
| **trade.pick_decisions** | ALL | - | - | READ/WRITE | READ | READ | READ |
| **system.process_locks** | ALL | - | - | READ/WRITE | READ/WRITE | READ/WRITE | READ |

**범례**:
- `ALL` = SUPERUSER (모든 권한)
- `READ/WRITE` = SELECT, INSERT, UPDATE, DELETE
- `READ` = SELECT만
- `UPDATE(컬럼)` = 특정 컬럼만 UPDATE
- `-` = 접근 불가 (REVOKE)

---

## 🚨 SSOT 위반 방지 (DB 강제)

### 문제 시나리오

**상황**: 개발자가 실수로 Exit Engine에서 `market.prices_best` 수정 시도

```go
// ❌ 금지 패턴 (Exit Engine 코드)
db.Exec("UPDATE market.prices_best SET last_price = $1 WHERE symbol = $2", price, symbol)
```

**결과**:
```
ERROR: permission denied for table prices_best (SQLSTATE 42501)
```

### 해결: Role 기반 강제

```sql
-- PriceSync만 market.* 쓰기 가능
GRANT UPDATE ON market.prices_best TO aegis_price;

-- Strategy는 읽기만 가능
GRANT SELECT ON market.prices_best TO aegis_trade;
REVOKE UPDATE, DELETE ON market.prices_best FROM aegis_trade;
```

**효과**:
- 코드 레벨 실수 → DB 레벨에서 차단
- 런타임 에러 (컴파일 시점 불가)
- 로그에 명확한 에러 메시지

---

## 🔧 애플리케이션 연결 설정

### 환경 변수

```bash
# PriceSync 모듈
DB_PRICE_HOST=localhost
DB_PRICE_PORT=5432
DB_PRICE_USER=aegis_price
DB_PRICE_PASSWORD=CHANGE_ME
DB_PRICE_DBNAME=aegis_v14

# Strategy 모듈 (Exit/Reentry)
DB_TRADE_HOST=localhost
DB_TRADE_PORT=5432
DB_TRADE_USER=aegis_trade
DB_TRADE_PASSWORD=CHANGE_ME
DB_TRADE_DBNAME=aegis_v14

# Execution 모듈
DB_EXEC_HOST=localhost
DB_EXEC_PORT=5432
DB_EXEC_USER=aegis_exec
DB_EXEC_PASSWORD=CHANGE_ME
DB_EXEC_DBNAME=aegis_v14

# BFF API (조회 전용)
DB_READONLY_HOST=localhost
DB_READONLY_PORT=5432
DB_READONLY_USER=aegis_readonly
DB_READONLY_PASSWORD=CHANGE_ME
DB_READONLY_DBNAME=aegis_v14
```

### Go 연결 예시

```go
// PriceSync 모듈
func NewPriceSyncDB() (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_PRICE_HOST"),
        os.Getenv("DB_PRICE_PORT"),
        os.Getenv("DB_PRICE_USER"),
        os.Getenv("DB_PRICE_PASSWORD"),
        os.Getenv("DB_PRICE_DBNAME"),
    )
    return pgxpool.New(context.Background(), dsn)
}

// Strategy 모듈
func NewStrategyDB() (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_TRADE_HOST"),
        os.Getenv("DB_TRADE_PORT"),
        os.Getenv("DB_TRADE_USER"),
        os.Getenv("DB_TRADE_PASSWORD"),
        os.Getenv("DB_TRADE_DBNAME"),
    )
    return pgxpool.New(context.Background(), dsn)
}
```

---

## 📋 마이그레이션 스크립트

### 001_create_roles.sql

```sql
-- =====================================================
-- v14 PostgreSQL Role 생성 스크립트
-- =====================================================

-- 1. 슈퍼관리자
CREATE ROLE aegis_admin WITH
    LOGIN
    PASSWORD 'CHANGE_ME_IN_PRODUCTION'
    SUPERUSER
    CREATEDB
    CREATEROLE;

COMMENT ON ROLE aegis_admin IS '슈퍼관리자 - 스키마 생성/마이그레이션 전용';

-- 2. PriceSync 모듈
CREATE ROLE aegis_price WITH
    LOGIN
    PASSWORD 'CHANGE_ME_IN_PRODUCTION'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_price IS 'PriceSync 모듈 - market.* 쓰기 권한';

-- 3. Strategy 모듈 (Exit/Reentry)
CREATE ROLE aegis_trade WITH
    LOGIN
    PASSWORD 'CHANGE_ME_IN_PRODUCTION'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_trade IS 'Strategy 모듈 - trade.* 일부 쓰기 권한';

-- 4. Execution 모듈
CREATE ROLE aegis_exec WITH
    LOGIN
    PASSWORD 'CHANGE_ME_IN_PRODUCTION'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_exec IS 'Execution 모듈 - 주문/체결 쓰기 권한';

-- 5. 조회 전용
CREATE ROLE aegis_readonly WITH
    LOGIN
    PASSWORD 'CHANGE_ME_IN_PRODUCTION'
    NOCREATEDB
    NOCREATEROLE;

COMMENT ON ROLE aegis_readonly IS '조회 전용 - 모든 테이블 SELECT만';

-- =====================================================
-- 초기 비밀번호 변경 강제 (선택)
-- =====================================================
-- ALTER ROLE aegis_price VALID UNTIL '2026-02-01';
-- ALTER ROLE aegis_trade VALID UNTIL '2026-02-01';
-- ALTER ROLE aegis_exec VALID UNTIL '2026-02-01';
-- ALTER ROLE aegis_readonly VALID UNTIL '2026-02-01';
```

### 002_grant_permissions.sql

```sql
-- =====================================================
-- v14 PostgreSQL 권한 부여 스크립트
-- =====================================================
-- 실행 순서: schema 생성 후, 테이블 생성 후 실행

-- aegis_price 권한 (PriceSync)
-- =====================================================
GRANT USAGE ON SCHEMA market TO aegis_price;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA market TO aegis_price;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA market TO aegis_price;

GRANT USAGE ON SCHEMA trade TO aegis_price;
GRANT SELECT ON ALL TABLES IN SCHEMA trade TO aegis_price;

ALTER DEFAULT PRIVILEGES IN SCHEMA market
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aegis_price;

ALTER DEFAULT PRIVILEGES IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_price;

-- aegis_trade 권한 (Strategy: Exit/Reentry)
-- =====================================================
GRANT USAGE ON SCHEMA market TO aegis_trade;
GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_trade;

GRANT USAGE ON SCHEMA trade TO aegis_trade;
-- positions: 컬럼별 권한
GRANT SELECT, INSERT ON trade.positions TO aegis_trade;
GRANT UPDATE (status, exit_mode, exit_profile_id, updated_ts) ON trade.positions TO aegis_trade;

GRANT SELECT, INSERT, UPDATE ON trade.position_state TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.reentry_candidates TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.order_intents TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.exit_signals TO aegis_trade;
GRANT SELECT, INSERT, UPDATE ON trade.exit_control TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.exit_profiles TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.symbol_exit_overrides TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.reentry_control TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.picks TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.pick_decisions TO aegis_trade;

GRANT SELECT ON trade.exit_events TO aegis_trade;
GRANT SELECT ON trade.orders TO aegis_trade;
GRANT SELECT ON trade.fills TO aegis_trade;
GRANT SELECT ON trade.holdings TO aegis_trade;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_trade;

-- aegis_exec 권한 (Execution)
-- =====================================================
GRANT USAGE ON SCHEMA market TO aegis_exec;
GRANT SELECT ON market.prices_best TO aegis_exec;

GRANT USAGE ON SCHEMA trade TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.orders TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.fills TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.holdings TO aegis_exec;
GRANT SELECT, INSERT, UPDATE, DELETE ON trade.exit_events TO aegis_exec;

GRANT SELECT ON trade.order_intents TO aegis_exec;
GRANT SELECT ON trade.positions TO aegis_exec;
GRANT UPDATE (qty, avg_price, updated_ts) ON trade.positions TO aegis_exec;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trade TO aegis_exec;

-- system schema 권한 (모든 모듈)
-- =====================================================
-- process_locks는 모든 모듈이 advisory lock 사용 가능
GRANT USAGE ON SCHEMA system TO aegis_price;
GRANT USAGE ON SCHEMA system TO aegis_trade;
GRANT USAGE ON SCHEMA system TO aegis_exec;

GRANT SELECT, INSERT, UPDATE, DELETE ON system.process_locks TO aegis_price;
GRANT SELECT, INSERT, UPDATE, DELETE ON system.process_locks TO aegis_trade;
GRANT SELECT, INSERT, UPDATE, DELETE ON system.process_locks TO aegis_exec;

-- aegis_readonly 권한 (조회 전용)
-- =====================================================
GRANT USAGE ON SCHEMA market TO aegis_readonly;
GRANT USAGE ON SCHEMA trade TO aegis_readonly;
GRANT USAGE ON SCHEMA system TO aegis_readonly;

GRANT SELECT ON ALL TABLES IN SCHEMA market TO aegis_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA trade TO aegis_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA system TO aegis_readonly;

ALTER DEFAULT PRIVILEGES IN SCHEMA market
    GRANT SELECT ON TABLES TO aegis_readonly;

ALTER DEFAULT PRIVILEGES IN SCHEMA trade
    GRANT SELECT ON TABLES TO aegis_readonly;

ALTER DEFAULT PRIVILEGES IN SCHEMA system
    GRANT SELECT ON TABLES TO aegis_readonly;
```

---

## 🧪 테스트 시나리오

### 1. SSOT 위반 테스트

```sql
-- aegis_trade 로그인
SET ROLE aegis_trade;

-- ❌ 실패해야 함: market.prices_best 수정 시도
UPDATE market.prices_best SET last_price = 100000 WHERE symbol = '005930';
-- 예상 결과: ERROR: permission denied for table prices_best

-- ✅ 성공해야 함: market.prices_best 조회
SELECT * FROM market.prices_best WHERE symbol = '005930';
-- 예상 결과: 1 row

-- ✅ 성공해야 함: trade.positions 전략 상태 수정 (Exit 소유 컬럼)
UPDATE trade.positions SET status = 'CLOSING', exit_mode = 'DEFAULT' WHERE position_id = '...';
-- 예상 결과: UPDATE 1

-- ❌ 실패해야 함: positions qty 수정 시도 (Execution 소유 컬럼)
UPDATE trade.positions SET qty = 10 WHERE position_id = '...';
-- 예상 결과: ERROR: permission denied for column "qty" of relation "positions"
```

### 2. Execution 권한 테스트

```sql
-- aegis_exec 로그인
SET ROLE aegis_exec;

-- ❌ 실패해야 함: position_state 수정 시도
UPDATE trade.position_state SET phase = 'TP1_DONE' WHERE position_id = '...';
-- 예상 결과: ERROR: permission denied for table position_state

-- ✅ 성공해야 함: positions 체결 동기화 (Execution 소유 컬럼)
UPDATE trade.positions SET qty = 5, avg_price = 71500, updated_ts = NOW() WHERE position_id = '...';
-- 예상 결과: UPDATE 1

-- ❌ 실패해야 함: positions status 수정 시도 (Exit 소유 컬럼)
UPDATE trade.positions SET status = 'CLOSED' WHERE position_id = '...';
-- 예상 결과: ERROR: permission denied for column "status" of relation "positions"

-- ❌ 실패해야 함: positions exit_mode 수정 시도 (Exit 소유 컬럼)
UPDATE trade.positions SET exit_mode = 'DISABLED' WHERE position_id = '...';
-- 예상 결과: ERROR: permission denied for column "exit_mode" of relation "positions"
```

---

## 🔗 관련 문서

- [schema.md](./schema.md) - 전체 테이블 스키마
- [system-overview.md](../architecture/system-overview.md) - SSOT 원칙
- [price-sync.md](../modules/price-sync.md) - PriceSync 모듈
- [exit-engine.md](../modules/exit-engine.md) - Exit Engine 모듈

---

**Module Owner**: Database
**Dependencies**: None (Infrastructure)
**Version**: v14.0.0-design
**Last Updated**: 2026-01-13
