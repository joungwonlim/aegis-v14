# 차트 데이터 마이그레이션 계획

v10 → v14 차트 데이터 테이블 마이그레이션

---

## 개요

**목적**: v10의 일봉/수급 데이터 테이블을 v14로 마이그레이션

**우선순위**: Phase 2 (StockDetailSheet 차트 기능 구현 전 필수)

**테이블 목록**:
1. `market.daily_prices` (일봉 데이터)
2. `market.investor_trading` (투자자별 매매동향)

---

## 1. market.daily_prices (일봉 데이터)

### v10 스키마 확인 필요

```sql
-- v10에서 실제 스키마를 확인해야 함
SELECT
  column_name,
  data_type,
  is_nullable,
  column_default
FROM information_schema.columns
WHERE table_schema = 'market'
  AND table_name = 'daily_prices'
ORDER BY ordinal_position;
```

### v14 예상 스키마 (v10 기반)

```sql
CREATE TABLE market.daily_prices (
  symbol VARCHAR(6) NOT NULL,
  date DATE NOT NULL,
  open INTEGER NOT NULL,           -- 시가
  high INTEGER NOT NULL,           -- 고가
  low INTEGER NOT NULL,            -- 저가
  close INTEGER NOT NULL,          -- 종가
  volume BIGINT NOT NULL,          -- 거래량
  value BIGINT,                    -- 거래대금

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (symbol, date),
  FOREIGN KEY (symbol) REFERENCES market.stocks(symbol) ON DELETE CASCADE
);

-- 날짜 내림차순 인덱스 (최신 데이터 우선 조회)
CREATE INDEX idx_daily_prices_date ON market.daily_prices(date DESC);

-- 종목별 날짜 범위 조회 최적화
CREATE INDEX idx_daily_prices_symbol_date ON market.daily_prices(symbol, date DESC);

COMMENT ON TABLE market.daily_prices IS '종목별 일봉 데이터 (OHLCV)';
COMMENT ON COLUMN market.daily_prices.symbol IS '종목코드 (6자리)';
COMMENT ON COLUMN market.daily_prices.date IS '거래일 (YYYY-MM-DD)';
COMMENT ON COLUMN market.daily_prices.open IS '시가 (원)';
COMMENT ON COLUMN market.daily_prices.high IS '고가 (원)';
COMMENT ON COLUMN market.daily_prices.low IS '저가 (원)';
COMMENT ON COLUMN market.daily_prices.close IS '종가 (원)';
COMMENT ON COLUMN market.daily_prices.volume IS '거래량 (주)';
COMMENT ON COLUMN market.daily_prices.value IS '거래대금 (원)';
```

### 권한 설정

```sql
-- trade_rw: 읽기 전용 (차트 조회만)
GRANT SELECT ON market.daily_prices TO trade_rw;

-- trade_admin: 전체 권한 (데이터 수집용)
GRANT ALL ON market.daily_prices TO trade_admin;
```

---

## 2. market.investor_trading (투자자별 매매동향)

### v10 스키마 확인 필요

```sql
-- v10에서 실제 스키마를 확인해야 함
SELECT
  column_name,
  data_type,
  is_nullable,
  column_default
FROM information_schema.columns
WHERE table_schema = 'market'
  AND table_name = 'investor_trading'
ORDER BY ordinal_position;
```

### v14 예상 스키마 (v10 기반)

```sql
CREATE TABLE market.investor_trading (
  symbol VARCHAR(6) NOT NULL,
  date DATE NOT NULL,
  foreign_net BIGINT NOT NULL,      -- 외국인 순매수 (주)
  inst_net BIGINT NOT NULL,         -- 기관 순매수 (주)
  indiv_net BIGINT NOT NULL,        -- 개인 순매수 (주)
  close_price INTEGER NOT NULL,     -- 종가 (원)
  price_change INTEGER NOT NULL,    -- 전일대비 (원)
  change_rate NUMERIC(5,2) NOT NULL, -- 전일대비 (%)
  volume BIGINT NOT NULL,           -- 거래량 (주)

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (symbol, date),
  FOREIGN KEY (symbol) REFERENCES market.stocks(symbol) ON DELETE CASCADE
);

-- 날짜 내림차순 인덱스 (최신 데이터 우선 조회)
CREATE INDEX idx_investor_trading_date ON market.investor_trading(date DESC);

-- 종목별 날짜 범위 조회 최적화
CREATE INDEX idx_investor_trading_symbol_date ON market.investor_trading(symbol, date DESC);

COMMENT ON TABLE market.investor_trading IS '종목별 투자자 매매동향 (외국인/기관/개인)';
COMMENT ON COLUMN market.investor_trading.symbol IS '종목코드 (6자리)';
COMMENT ON COLUMN market.investor_trading.date IS '거래일 (YYYY-MM-DD)';
COMMENT ON COLUMN market.investor_trading.foreign_net IS '외국인 순매수 (주, 양수=매수/음수=매도)';
COMMENT ON COLUMN market.investor_trading.inst_net IS '기관 순매수 (주, 양수=매수/음수=매도)';
COMMENT ON COLUMN market.investor_trading.indiv_net IS '개인 순매수 (주, 양수=매수/음수=매도)';
COMMENT ON COLUMN market.investor_trading.close_price IS '종가 (원)';
COMMENT ON COLUMN market.investor_trading.price_change IS '전일대비 (원)';
COMMENT ON COLUMN market.investor_trading.change_rate IS '전일대비 (%)';
COMMENT ON COLUMN market.investor_trading.volume IS '거래량 (주)';
```

### 권한 설정

```sql
-- trade_rw: 읽기 전용 (차트 조회만)
GRANT SELECT ON market.investor_trading TO trade_rw;

-- trade_admin: 전체 권한 (데이터 수집용)
GRANT ALL ON market.investor_trading TO trade_admin;
```

---

## 마이그레이션 단계

### Phase 1: v10 스키마 분석

**작업**:
1. v10 PostgreSQL 접속
2. 실제 테이블 스키마 확인
3. 데이터 샘플 조회 (최근 10일)
4. 인덱스 및 제약조건 확인

**명령어**:
```bash
# v10 DB 접속
psql -h localhost -U postgres -d aegis_v10

# 스키마 확인
\d market.daily_prices
\d market.investor_trading

# 샘플 데이터 조회
SELECT * FROM market.daily_prices WHERE symbol = '005930' ORDER BY date DESC LIMIT 10;
SELECT * FROM market.investor_trading WHERE symbol = '005930' ORDER BY date DESC LIMIT 10;
```

### Phase 2: v14 테이블 생성

**스크립트**: `scripts/db/06_create_chart_tables.sql`

```sql
-- 1. market.daily_prices 생성
CREATE TABLE IF NOT EXISTS market.daily_prices (
  -- (위 스키마 참조)
);

-- 2. market.investor_trading 생성
CREATE TABLE IF NOT EXISTS market.investor_trading (
  -- (위 스키마 참조)
);

-- 3. 인덱스 생성
-- (위 인덱스 참조)

-- 4. 권한 설정
-- (위 권한 참조)
```

### Phase 3: 데이터 이관 (선택)

**옵션 A: v10 데이터 복사 (필요 시)**

```sql
-- v10 → v14 데이터 복사 (pg_dump 사용)
-- 1. v10에서 데이터 덤프
pg_dump -h localhost -U postgres -d aegis_v10 \
  --table=market.daily_prices \
  --table=market.investor_trading \
  --data-only \
  --file=v10_chart_data.sql

-- 2. v14로 데이터 임포트
psql -h localhost -U postgres -d aegis_v14 -f v10_chart_data.sql
```

**옵션 B: 새로 수집 (권장)**

v14에서 Fetcher 모듈로 새로 데이터 수집
- 장점: 데이터 일관성 보장
- 단점: 과거 데이터 누락 (수집 시점부터만)

### Phase 4: 검증

**데이터 검증**:
```sql
-- 1. 레코드 수 확인
SELECT COUNT(*) FROM market.daily_prices;
SELECT COUNT(*) FROM market.investor_trading;

-- 2. 종목별 최신 데이터 확인
SELECT symbol, MAX(date) AS latest_date
FROM market.daily_prices
GROUP BY symbol
ORDER BY symbol;

SELECT symbol, MAX(date) AS latest_date
FROM market.investor_trading
GROUP BY symbol
ORDER BY symbol;

-- 3. 데이터 무결성 확인 (NULL 체크)
SELECT COUNT(*) FROM market.daily_prices
WHERE open IS NULL OR high IS NULL OR low IS NULL OR close IS NULL;

SELECT COUNT(*) FROM market.investor_trading
WHERE foreign_net IS NULL OR inst_net IS NULL OR indiv_net IS NULL;
```

---

## API 연동 테스트

### 1. Backend API 구현 확인

```bash
# API 엔드포인트 테스트
curl http://localhost:8080/api/stocks/005930/daily-prices?days=90
curl http://localhost:8080/api/stocks/005930/investor-trading?days=90
```

### 2. Frontend 연동 테스트

```tsx
// StockDetailSheet에서 Chart 탭 클릭 시 데이터 로딩 확인
// 1. useDailyPrices 훅 호출
// 2. useInvestorTrading 훅 호출
// 3. 차트 렌더링 확인
```

---

## 롤백 계획

### 테이블 삭제

```sql
-- 롤백 시 테이블 삭제
DROP TABLE IF EXISTS market.investor_trading CASCADE;
DROP TABLE IF EXISTS market.daily_prices CASCADE;
```

---

## 마이그레이션 체크리스트

- [ ] Phase 1: v10 스키마 분석 완료
- [ ] Phase 2: v14 테이블 생성 완료
- [ ] Phase 3: 데이터 이관 또는 수집 완료 (선택)
- [ ] Phase 4: 데이터 검증 완료
- [ ] Backend API 구현 완료
- [ ] Frontend 차트 컴포넌트 구현 완료
- [ ] 통합 테스트 완료

---

## 참고 문서

- [docs/ui/charts.md](../ui/charts.md)
- [docs/database/schema.md](./schema.md)
- [docs/modules/stock-detail-sheet.md](../modules/stock-detail-sheet.md)

---

**작성일**: 2026-01-17
**Phase**: Phase 2 (설계 완료, 구현 대기)
