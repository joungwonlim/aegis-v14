# market.stocks 테이블 마이그레이션 계획

> 종목 마스터 테이블 추가 및 FK 제약조건 적용

**목적**: 종목 정보의 SSOT 확립 및 참조 무결성 보장

**Last Updated**: 2026-01-13

---

## 📋 목차

1. [현재 문제점](#1-현재-문제점)
2. [목표](#2-목표)
3. [마이그레이션 전략](#3-마이그레이션-전략)
4. [Phase 1: stocks 테이블 생성](#phase-1-stocks-테이블-생성)
5. [Phase 2: 데이터 적재](#phase-2-데이터-적재)
6. [Phase 3: FK 제약조건 추가](#phase-3-fk-제약조건-추가)
7. [Phase 4: 권한 설정](#phase-4-권한-설정)
8. [Phase 5: 검증](#phase-5-검증)
9. [롤백 계획](#롤백-계획)
10. [운영 영향](#운영-영향)

---

## 1. 현재 문제점

### ❌ 참조 무결성 없음

```sql
-- 현재: 모든 테이블에서 symbol이 TEXT (FK 없음)
CREATE TABLE market.prices_best (
    symbol TEXT PRIMARY KEY,  -- ❌ 잘못된 종목코드 입력 가능
    ...
);

CREATE TABLE trade.positions (
    symbol TEXT NOT NULL,     -- ❌ 존재하지 않는 종목코드 가능
    ...
);
```

**위험**:
1. 잘못된 종목코드 입력 가능 (`"삼성전자"` vs `"005930"` 혼용)
2. 종목명/시장구분 정보 없음 또는 중복 관리
3. 상장폐지 종목 필터링 불가
4. 거래정지 종목 체크 불가

---

## 2. 목표

### ✅ 달성 목표

1. **종목 마스터 SSOT 확립**
   - `market.stocks` 테이블이 모든 종목 정보의 단일 원천

2. **참조 무결성 보장**
   - 모든 `symbol` 컬럼이 `market.stocks(symbol)`을 FK 참조

3. **종목코드 표준화**
   - 6자리 숫자 형식 강제 (`005930`, `069500`)

4. **거래 가능 여부 관리**
   - `is_tradable` 플래그로 거래정지/상장폐지 종목 필터링

5. **권한 분리**
   - DataSync: stocks 관리 전담
   - 다른 모듈: 읽기 전용

---

## 3. 마이그레이션 전략

### 원칙

1. **무중단 마이그레이션**: 서비스 중단 없이 단계적 적용
2. **검증 우선**: 각 단계마다 철저한 검증
3. **롤백 가능**: 문제 발생 시 즉시 롤백 가능
4. **영향 최소화**: 기존 데이터/로직 변경 최소화

### 단계별 적용

```
Phase 1: stocks 테이블 생성 (DDL)
   ↓
Phase 2: 초기 데이터 적재 (KIS API)
   ↓
Phase 3: FK 제약조건 추가 (검증 후)
   ↓
Phase 4: 권한 설정 (DataSync role)
   ↓
Phase 5: 검증 및 모니터링
```

---

## Phase 1: stocks 테이블 생성

### 1.1. DDL 실행

```sql
-- 1. stocks 테이블 생성
CREATE TABLE market.stocks (
    symbol        TEXT PRIMARY KEY,
    name          TEXT        NOT NULL,
    market        TEXT        NOT NULL,

    -- 종목 상태
    status        TEXT        NOT NULL DEFAULT 'ACTIVE',
    listing_date  DATE,
    delisting_date DATE,

    -- 메타 정보
    sector        TEXT,
    industry      TEXT,
    market_cap    BIGINT,

    -- 거래 제약
    is_tradable   BOOLEAN     NOT NULL DEFAULT true,
    trade_halt_reason TEXT,

    -- 감사
    created_ts    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_ts    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_market CHECK (market IN ('KOSPI', 'KOSDAQ', 'KONEX')),
    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELISTED')),
    CONSTRAINT chk_symbol_format CHECK (symbol ~ '^\d{6}$')
);

-- 2. 인덱스 생성
CREATE INDEX idx_stocks_market ON market.stocks (market);
CREATE INDEX idx_stocks_status ON market.stocks (status);
CREATE INDEX idx_stocks_tradable ON market.stocks (is_tradable) WHERE is_tradable = true;
CREATE INDEX idx_stocks_name ON market.stocks (name);

-- 3. updated_ts 자동 갱신 트리거
CREATE OR REPLACE FUNCTION update_stocks_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_ts = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_stocks_updated_ts
    BEFORE UPDATE ON market.stocks
    FOR EACH ROW
    EXECUTE FUNCTION update_stocks_updated_ts();
```

### 1.2. 검증

```sql
-- 테이블 생성 확인
SELECT tablename, schemaname
FROM pg_tables
WHERE schemaname = 'market' AND tablename = 'stocks';

-- 인덱스 확인
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'market' AND tablename = 'stocks';

-- 제약조건 확인
SELECT conname, contype, pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conrelid = 'market.stocks'::regclass;
```

---

## Phase 2: 데이터 적재

### 2.1. 기존 symbol 추출

```sql
-- 1. 기존 데이터에서 symbol 목록 추출
CREATE TEMP TABLE temp_symbols AS
SELECT DISTINCT symbol
FROM (
    SELECT symbol FROM market.prices_best
    UNION
    SELECT symbol FROM trade.positions
    UNION
    SELECT symbol FROM trade.orders
) t
WHERE symbol ~ '^\d{6}$'  -- 6자리 숫자만
ORDER BY symbol;

-- 2. 중복 확인
SELECT symbol, COUNT(*)
FROM temp_symbols
GROUP BY symbol
HAVING COUNT(*) > 1;
-- 결과: 0 rows (중복 없어야 함)

-- 3. 개수 확인
SELECT COUNT(*) FROM temp_symbols;
```

### 2.2. KIS API로 종목 정보 조회

**외부 스크립트 (Python/Go)**:

```python
# kis_fetch_stocks.py
import requests
import psycopg2

def fetch_stock_info(symbol):
    """KIS API로 종목 정보 조회"""
    # KIS API 호출
    response = requests.get(
        f"https://openapi.koreainvestment.com:9443/uapi/domestic-stock/v1/quotations/inquire-price",
        headers={"authorization": "Bearer ...", ...},
        params={"FID_COND_MRKT_DIV_CODE": "J", "FID_INPUT_ISCD": symbol}
    )

    data = response.json()["output"]
    return {
        "symbol": symbol,
        "name": data["hts_kor_isnm"],  # 종목명
        "market": "KOSPI" if data["rprs_mrkt_kor_name"] == "KOSPI" else "KOSDAQ",
        "sector": data.get("bstp_kor_isnm"),  # 업종
        "market_cap": int(data.get("stck_avls", 0))  # 시가총액
    }

def main():
    conn = psycopg2.connect("dbname=aegis_v14 user=aegis_datasync")
    cur = conn.cursor()

    # temp_symbols에서 symbol 목록 가져오기
    cur.execute("SELECT symbol FROM temp_symbols")
    symbols = [row[0] for row in cur.fetchall()]

    for symbol in symbols:
        try:
            info = fetch_stock_info(symbol)

            # market.stocks에 INSERT
            cur.execute("""
                INSERT INTO market.stocks (symbol, name, market, sector, market_cap, status, is_tradable)
                VALUES (%(symbol)s, %(name)s, %(market)s, %(sector)s, %(market_cap)s, 'ACTIVE', true)
                ON CONFLICT (symbol) DO UPDATE
                SET name = EXCLUDED.name,
                    market = EXCLUDED.market,
                    sector = EXCLUDED.sector,
                    market_cap = EXCLUDED.market_cap,
                    updated_ts = now()
            """, info)

            conn.commit()
            print(f"✓ {symbol}: {info['name']}")

        except Exception as e:
            print(f"✗ {symbol}: {e}")
            conn.rollback()

    cur.close()
    conn.close()

if __name__ == "__main__":
    main()
```

### 2.3. 수동 보완 (옵션)

```sql
-- KIS API에서 못 가져온 종목은 수동 입력
INSERT INTO market.stocks (symbol, name, market, status, is_tradable)
VALUES
    ('005930', '삼성전자', 'KOSPI', 'ACTIVE', true),
    ('000660', 'SK하이닉스', 'KOSPI', 'ACTIVE', true),
    ('035720', '카카오', 'KOSDAQ', 'ACTIVE', true)
ON CONFLICT (symbol) DO NOTHING;
```

### 2.4. 검증

```sql
-- 1. 적재 건수 확인
SELECT COUNT(*) FROM market.stocks;
-- 기대값: temp_symbols와 동일

-- 2. 종목명 누락 확인
SELECT symbol, name
FROM market.stocks
WHERE name IS NULL OR name = '';
-- 결과: 0 rows (모두 종목명 있어야 함)

-- 3. 시장구분 확인
SELECT market, COUNT(*)
FROM market.stocks
GROUP BY market;
-- 기대값: KOSPI > 800, KOSDAQ > 1500

-- 4. temp 테이블 정리
DROP TABLE IF EXISTS temp_symbols;
```

---

## Phase 3: FK 제약조건 추가

### ⚠️ 주의사항

- **무결성 검증 필수**: FK 추가 전 orphan rows 확인
- **단계적 적용**: 한 테이블씩 적용 및 검증
- **운영 시간 고려**: 장 마감 후 적용 권장

### 3.1. 무결성 검증

```sql
-- 1. market.prices_best
SELECT p.symbol
FROM market.prices_best p
LEFT JOIN market.stocks s ON p.symbol = s.symbol
WHERE s.symbol IS NULL;
-- 결과: 0 rows (orphan 없어야 함)

-- 2. market.prices_ticks
SELECT DISTINCT pt.symbol
FROM market.prices_ticks pt
LEFT JOIN market.stocks s ON pt.symbol = s.symbol
WHERE s.symbol IS NULL;
-- 결과: 0 rows

-- 3. trade.positions
SELECT DISTINCT p.symbol
FROM trade.positions p
LEFT JOIN market.stocks s ON p.symbol = s.symbol
WHERE s.symbol IS NULL;
-- 결과: 0 rows

-- 4. trade.orders
SELECT DISTINCT o.symbol
FROM trade.orders o
LEFT JOIN market.stocks s ON o.symbol = s.symbol
WHERE s.symbol IS NULL;
-- 결과: 0 rows
```

**Orphan rows 발견 시**:
```sql
-- 임시로 stocks에 추가 (나중에 정리)
INSERT INTO market.stocks (symbol, name, market, status)
SELECT DISTINCT
    symbol,
    '종목명미상-' || symbol,
    'KOSPI',  -- 기본값
    'ACTIVE'
FROM market.prices_best
WHERE symbol NOT IN (SELECT symbol FROM market.stocks)
ON CONFLICT (symbol) DO NOTHING;
```

### 3.2. FK 제약조건 추가 (단계적)

```sql
-- 1. market.prices_best (READ-HEAVY, 조심스럽게)
ALTER TABLE market.prices_best
    ADD CONSTRAINT fk_prices_best_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE CASCADE;  -- 종목 삭제 시 가격 데이터도 삭제

-- 검증
SELECT conname, contype, confdeltype
FROM pg_constraint
WHERE conrelid = 'market.prices_best'::regclass
  AND conname = 'fk_prices_best_symbol';

-- 2. market.prices_ticks
ALTER TABLE market.prices_ticks
    ADD CONSTRAINT fk_prices_ticks_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE CASCADE;

-- 3. market.freshness
ALTER TABLE market.freshness
    ADD CONSTRAINT fk_freshness_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE CASCADE;

-- 4. trade.positions (CRITICAL - 조심)
ALTER TABLE trade.positions
    ADD CONSTRAINT fk_positions_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE RESTRICT;  -- 포지션 있으면 종목 삭제 불가

-- 5. trade.orders
ALTER TABLE trade.orders
    ADD CONSTRAINT fk_orders_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE RESTRICT;

-- 6. trade.order_intents
ALTER TABLE trade.order_intents
    ADD CONSTRAINT fk_order_intents_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE RESTRICT;

-- 7. trade.exit_events
ALTER TABLE trade.exit_events
    ADD CONSTRAINT fk_exit_events_symbol
    FOREIGN KEY (symbol) REFERENCES market.stocks(symbol)
    ON DELETE RESTRICT;
```

### 3.3. FK 제약조건 검증

```sql
-- 모든 FK 확인
SELECT
    tc.table_schema,
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    rc.delete_rule
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
JOIN information_schema.referential_constraints AS rc
  ON rc.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND ccu.table_name = 'stocks'
ORDER BY tc.table_schema, tc.table_name;

-- 기대 결과:
-- market.prices_best -> market.stocks (CASCADE)
-- market.prices_ticks -> market.stocks (CASCADE)
-- market.freshness -> market.stocks (CASCADE)
-- trade.positions -> market.stocks (RESTRICT)
-- trade.orders -> market.stocks (RESTRICT)
-- ...
```

---

## Phase 4: 권한 설정

### 4.1. DataSync Role 생성

```sql
-- 1. Role 생성
CREATE ROLE aegis_datasync WITH LOGIN PASSWORD 'CHANGE_ME_IN_PRODUCTION';

-- 2. market.stocks 권한
GRANT SELECT, INSERT, UPDATE, DELETE ON market.stocks TO aegis_datasync;

-- 3. 시퀀스 권한 (id 컬럼 있는 경우)
-- market.stocks는 시퀀스 없음 (symbol이 PK)

-- 4. updated_ts 트리거 실행 권한
GRANT EXECUTE ON FUNCTION update_stocks_updated_ts() TO aegis_datasync;
```

### 4.2. 기존 Role 권한 조정

```sql
-- 읽기 전용 권한 부여 (전략/실행 모듈)
GRANT SELECT ON market.stocks TO aegis_trade;   -- Exit Engine
GRANT SELECT ON market.stocks TO aegis_exec;    -- Execution
GRANT SELECT ON market.stocks TO aegis_router;  -- Router

-- 쓰기 권한 없음 확인
SELECT grantee, privilege_type
FROM information_schema.table_privileges
WHERE table_schema = 'market'
  AND table_name = 'stocks'
  AND grantee IN ('aegis_trade', 'aegis_exec', 'aegis_router')
  AND privilege_type IN ('INSERT', 'UPDATE', 'DELETE');
-- 결과: 0 rows (쓰기 권한 없어야 함)
```

---

## Phase 5: 검증

### 5.1. 기능 테스트

```sql
-- 1. 종목 조회 (읽기)
SELECT symbol, name, market, is_tradable
FROM market.stocks
WHERE symbol = '005930';

-- 2. 종목 추가 (DataSync만 가능)
SET ROLE aegis_datasync;
INSERT INTO market.stocks (symbol, name, market, status)
VALUES ('999999', '테스트종목', 'KOSDAQ', 'ACTIVE');
-- 성공

-- 3. 잘못된 종목코드 INSERT 시도 (실패해야 함)
INSERT INTO market.stocks (symbol, name, market)
VALUES ('ABC123', '잘못된코드', 'KOSPI');
-- ERROR: new row for relation "stocks" violates check constraint "chk_symbol_format"

-- 4. FK 제약 확인 (존재하지 않는 종목으로 주문 시도)
SET ROLE aegis_exec;
INSERT INTO trade.orders (order_id, symbol, ...)
VALUES (gen_random_uuid(), '888888', ...);
-- ERROR: insert or update on table "orders" violates foreign key constraint "fk_orders_symbol"

-- 5. 거래정지 종목 필터링
SELECT symbol, name, trade_halt_reason
FROM market.stocks
WHERE is_tradable = false;
```

### 5.2. 성능 테스트

```sql
-- 1. JOIN 성능 확인 (FK 인덱스 활용)
EXPLAIN ANALYZE
SELECT p.symbol, s.name, s.market, p.last_price
FROM market.prices_best p
JOIN market.stocks s ON p.symbol = s.symbol
WHERE s.is_tradable = true
  AND s.market = 'KOSPI';

-- 기대값: Index Scan on idx_stocks_tradable

-- 2. INSERT 성능 (FK 검증 오버헤드)
-- 가격 데이터 INSERT 시 stocks FK 검증 시간 측정
```

### 5.3. 모니터링

```sql
-- 1. stocks 테이블 크기
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'market' AND tablename = 'stocks';

-- 2. FK 제약조건 위반 모니터링 (로그)
-- PostgreSQL 로그에서 "violates foreign key constraint" 검색

-- 3. 일일 종목 변경 추적
SELECT
    DATE(updated_ts) AS date,
    COUNT(*) AS updated_count
FROM market.stocks
WHERE updated_ts > NOW() - INTERVAL '7 days'
GROUP BY DATE(updated_ts)
ORDER BY date DESC;
```

---

## 롤백 계획

### 문제 발생 시 즉시 롤백

```sql
-- Phase 3 롤백: FK 제약조건 제거
ALTER TABLE market.prices_best DROP CONSTRAINT IF EXISTS fk_prices_best_symbol;
ALTER TABLE market.prices_ticks DROP CONSTRAINT IF EXISTS fk_prices_ticks_symbol;
ALTER TABLE market.freshness DROP CONSTRAINT IF EXISTS fk_freshness_symbol;
ALTER TABLE trade.positions DROP CONSTRAINT IF EXISTS fk_positions_symbol;
ALTER TABLE trade.orders DROP CONSTRAINT IF EXISTS fk_orders_symbol;
ALTER TABLE trade.order_intents DROP CONSTRAINT IF EXISTS fk_order_intents_symbol;
ALTER TABLE trade.exit_events DROP CONSTRAINT IF EXISTS fk_exit_events_symbol;

-- Phase 4 롤백: 권한 제거
REVOKE ALL ON market.stocks FROM aegis_datasync;
DROP ROLE IF EXISTS aegis_datasync;

-- Phase 2 롤백: 데이터 삭제 (주의!)
-- TRUNCATE market.stocks;  -- ⚠️ FK CASCADE 주의!

-- Phase 1 롤백: 테이블 삭제 (최후 수단)
-- DROP TABLE IF EXISTS market.stocks CASCADE;  -- ⚠️ CASCADE 주의!
```

### 롤백 검증

```sql
-- FK 제약조건 제거 확인
SELECT conname
FROM pg_constraint
WHERE conrelid IN (
    'market.prices_best'::regclass,
    'trade.positions'::regclass
)
AND conname LIKE 'fk_%_symbol';
-- 결과: 0 rows (모두 제거됨)
```

---

## 운영 영향

### 영향도 분석

| Phase | 다운타임 | 영향 범위 | 리스크 |
|-------|----------|----------|--------|
| Phase 1 (DDL) | 없음 | 없음 | 낮음 (새 테이블 생성) |
| Phase 2 (데이터) | 없음 | 없음 | 낮음 (읽기 전용) |
| Phase 3 (FK) | **있음** | **중간** | **중간** (쓰기 성능 영향) |
| Phase 4 (권한) | 없음 | 낮음 | 낮음 |
| Phase 5 (검증) | 없음 | 없음 | 낮음 |

### Phase 3 주의사항 (FK 제약조건 추가)

**영향**:
- INSERT/UPDATE 성능 약간 저하 (FK 검증 오버헤드)
- 잘못된 symbol INSERT 시 즉시 에러 발생

**권장 적용 시간**:
- 장 마감 후 (15:30~익일 08:00)
- 거래량 낮은 시간대

**비상 대응**:
- FK 제약조건 제거 스크립트 준비
- 롤백 테스트 사전 실행

### 성능 영향 예측

**Before (FK 없음)**:
```sql
INSERT INTO trade.orders (symbol, ...)
VALUES ('005930', ...);
-- 시간: ~1ms
```

**After (FK 있음)**:
```sql
INSERT INTO trade.orders (symbol, ...)
VALUES ('005930', ...);
-- 시간: ~1.2ms (FK 검증 +0.2ms)
```

**영향**: 무시할 수준 (초당 수백 건 처리 가능)

---

## 일정

| Phase | 예상 소요 시간 | 권장 작업 시간 |
|-------|---------------|---------------|
| Phase 1 | 10분 | 언제나 가능 |
| Phase 2 | 30분~1시간 | 장 마감 후 |
| Phase 3 | 20분 (테이블당 2~3분) | 장 마감 후 (필수) |
| Phase 4 | 5분 | 언제나 가능 |
| Phase 5 | 30분 | 장 마감 후 |
| **합계** | **2시간** | **장 마감 후 1회** |

---

## 체크리스트

### 마이그레이션 전

- [ ] **백업 완료** (`pg_dump aegis_v14 > backup_$(date +%Y%m%d).sql`)
- [ ] **롤백 스크립트 준비**
- [ ] **KIS API 토큰 확인** (종목 정보 조회용)
- [ ] **운영 시간 확인** (장 마감 후)
- [ ] **DataSync 모듈 준비** (종목 정보 동기화 로직)

### 마이그레이션 후

- [ ] **FK 제약조건 확인** (모든 테이블)
- [ ] **권한 확인** (DataSync만 쓰기 가능)
- [ ] **기능 테스트** (종목 조회/추가/거래정지)
- [ ] **성능 모니터링** (INSERT 성능 확인)
- [ ] **로그 모니터링** (FK 위반 에러 확인)

---

## 관련 문서

- [database/schema.md](./schema.md) - market.stocks 테이블 스키마
- [database/access-control.md](./access-control.md) - DataSync role 권한

---

**Version**: v14.0.0-design
**Last Updated**: 2026-01-13
