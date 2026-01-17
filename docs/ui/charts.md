# 차트 컴포넌트 설계 (Charts Design)

v14 StockDetailSheet의 차트 기능 설계 (v10 포팅)

---

## 📋 개요

**목적**: v10의 일봉 차트 및 수급 차트를 v14 StockDetailSheet에 통합

**위치**: `frontend/components/stock-detail-sheet/tabs/chart-tab.tsx`

**Phase**: Phase 2 완료 ✅ (2026-01-17)

---

## 🎨 컴포넌트 구조

```
frontend/components/stock-detail-sheet/
├── tabs/
│   └── chart-tab.tsx                        # Chart 탭 (일봉 + 수급 통합)
│       ├── PriceChart                       # 일봉 차트 컴포넌트
│       ├── InvestorTradingChart             # 수급 차트 컴포넌트
│       └── ChartTab                         # 메인 export
└── types.ts                                 # DailyPrice, InvestorFlow 타입
```

---

## 📊 1. PriceChart (일봉 차트)

### 목적
종목의 일봉 가격 데이터를 Candlestick 차트로 표시

### 기능

| 기능 | 설명 | v10 포팅 |
|------|------|----------|
| Candlestick | 고가/저가/시가/종가를 캔들 형태로 표시 (상승: 빨강 테두리+투명, 하락: 파랑 fill) | ✅ |
| 거래량 | 하단에 Bar 차트로 거래량 표시 (100px, 상승/하락 색상 구분) | ✅ |
| 기간 필터 | 1M, 3M, 6M, 1Y 버튼으로 기간 선택 (기본값: 3M) | ✅ |
| 평단가 선 | 보유 종목인 경우 평균매입단가 표시 (노란색 점선, fontSize:14, bold) | ✅ |
| Crosshair | 마우스 위치의 가격을 파란색 점선으로 표시 | ✅ |
| Y축 가격 라벨 | 마우스 위치 가격을 Y축 왼쪽에 파란색으로 표시 | ✅ |
| Tooltip | 마우스 호버 시 고가/저가/종가 표시 (천 단위 구분자) | ✅ |

### Props

```typescript
interface PriceChartProps {
  data: DailyPrice[]       // 일봉 데이터 배열
  isLoading?: boolean      // 로딩 상태
  avgBuyPrice?: number     // 평균 매입 단가 (보유 종목만)
}

interface DailyPrice {
  date: string             // YYYY-MM-DD 형식
  open: number             // 시가
  high: number             // 고가
  low: number              // 저가
  close: number            // 종가
  volume: number           // 거래량
}
```

### 기술 스택

- **recharts**: ComposedChart (Candlestick + 거래량)
- **Bar + Custom Shape**: Candlestick 구현
- **ReferenceLine**: 평단가, Crosshair 표시

### 레이아웃

```
┌────────────────────────────────────────────┐
│ 일봉 차트                 [1M][3M][6M][1Y] │  ← 헤더 + 기간 필터
├────────────────────────────────────────────┤
│                                            │
│   ┃                                        │  ← Candlestick 차트
│   ┃  ╱╲  ┃                                 │     (400px)
│ ──┃──────┃──────── (평단가 선)             │
│   ┃      ┃                                 │
├────────────────────────────────────────────┤
│   ▌▌▌  ▌▌  ▌▌▌                            │  ← 거래량 Bar 차트
│                                            │     (100px)
└────────────────────────────────────────────┘
```

### v10 참조 파일
`/Users/wonny/Dev/aegis/v10/frontend/src/modules/stock/components/PriceChart.tsx`

---

## 📈 2. InvestorTradingChart (수급 차트)

### 목적
외국인/기관/개인의 순매수량 추이를 라인 차트로 표시

### 기능

| 기능 | 설명 |
|------|------|
| 3개 라인 차트 | 외국인(빨강), 기관(보라), 개인(노랑) |
| 기간 필터 | 1M, 3M, 6M, 1Y 버튼으로 기간 선택 |
| 0 기준선 | Y축 0에 ReferenceLine 표시 |
| 데이터 테이블 | 최근 10일 데이터를 테이블로 표시 (하단) |
| Y축 포맷팅 | 억/만/천 단위 자동 변환 |
| Tooltip | 마우스 호버 시 외국인/기관/개인 순매수량 표시 |

### Props

```typescript
interface InvestorTradingChartProps {
  data: InvestorTrading[]   // 수급 데이터 배열
  isLoading?: boolean       // 로딩 상태
}

interface InvestorTrading {
  date: string              // YYYY-MM-DD 형식
  foreign_net: number       // 외국인 순매수 (주)
  inst_net: number          // 기관 순매수 (주)
  indiv_net: number         // 개인 순매수 (주)
  close_price: number       // 종가
  price_change: number      // 전일대비 (원)
  change_rate: number       // 전일대비 (%)
  volume: number            // 거래량
}
```

### 기술 스택

- **recharts**: LineChart
- **Line (3개)**: 외국인/기관/개인
- **ReferenceLine**: Y축 0 기준선

### 레이아웃

```
┌────────────────────────────────────────────┐
│ 투자자별 매매동향         [1M][3M][6M][1Y] │  ← 헤더 + 기간 필터
│ 2026.01.01 - 2026.01.17 기준               │  ← 날짜 범위
│                      [외국인][기관][개인]  │  ← 범례
├────────────────────────────────────────────┤
│        ╱──╲                                │  ← 라인 차트 (200px)
│  ─────────────0─────────────               │     0 기준선
│             ╲──╱                           │
├────────────────────────────────────────────┤
│ 날짜   종가  전일대비  외국인  기관  개인   │  ← 데이터 테이블
│ 01.17  1000   +10    +100   -50   -50     │     (최근 10일)
│ 01.16   990   -05    -200  +100  +100     │
│ ...                                        │
└────────────────────────────────────────────┘
```

### 색상 코드

| 투자자 | 색상 | HEX |
|--------|------|-----|
| 외국인 | 빨강/주황 | `#F04452` |
| 기관 | 보라 | `#7B61FF` |
| 개인 | 노랑/주황 | `#F2A93B` |

### v10 참조 파일
`/Users/wonny/Dev/aegis/v10/frontend/src/modules/stock/components/InvestorTradingChart.tsx`

---

## 🗄️ 데이터 소스 (DB 마이그레이션 완료 ✅)

### 1. data.daily_prices (일봉 데이터)

**스키마**: `data` (파티션 테이블)

```sql
CREATE TABLE data.daily_prices (
  stock_code VARCHAR(6) NOT NULL,
  trade_date DATE NOT NULL,
  open_price NUMERIC(10,2) NOT NULL,
  high_price NUMERIC(10,2) NOT NULL,
  low_price NUMERIC(10,2) NOT NULL,
  close_price NUMERIC(10,2) NOT NULL,
  volume BIGINT NOT NULL,

  PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 주의: 실제 컬럼명은 snake_case 사용
-- trade_date, open_price, high_price, low_price, close_price
```

### 2. data.investor_flow (수급 데이터)

**스키마**: `data` (파티션 테이블)

```sql
CREATE TABLE data.investor_flow (
  stock_code VARCHAR(6) NOT NULL,
  trade_date DATE NOT NULL,
  foreign_net_qty BIGINT NOT NULL,  -- 외국인 순매수 (주)
  inst_net_qty BIGINT NOT NULL,     -- 기관 순매수 (주)
  indiv_net_qty BIGINT NOT NULL,    -- 개인 순매수 (주)

  PRIMARY KEY (stock_code, trade_date)
) PARTITION BY RANGE (trade_date);

-- 주의: 가격 정보는 daily_prices와 LEFT JOIN으로 조회
```

---

## 🌐 API 설계 (Backend 구현 완료 ✅)

### 1. GET /api/v1/fetcher/prices/{code}/history

**목적**: 종목의 일봉 데이터 조회

**쿼리 파라미터**:
```
?start_date=2025-10-17  # 시작일 (기본값: 3개월 전)
?end_date=2026-01-17    # 종료일 (기본값: 오늘)
```

**응답**:
```json
{
  "success": true,
  "data": [
    {
      "date": "2026-01-17",
      "open": 1000,
      "high": 1050,
      "low": 990,
      "close": 1020,
      "volume": 1000000
    }
  ]
}
```

**구현 파일**:
- `backend/internal/api/handlers/chart_handler.go`
- `backend/internal/api/routes/chart_routes.go`

**에러**:
- `500`: DB 조회 실패

---

### 2. GET /api/v1/fetcher/flows/{code}/history

**목적**: 종목의 투자자별 매매동향 조회

**쿼리 파라미터**:
```
?start_date=2025-12-17  # 시작일 (기본값: 1개월 전)
?end_date=2026-01-17    # 종료일 (기본값: 오늘)
```

**응답**:
```json
{
  "success": true,
  "data": [
    {
      "date": "2026-01-17",
      "foreign_net": 100000,
      "inst_net": -50000,
      "retail_net": -50000,
      "close_price": 1020,
      "price_change": 10,
      "change_rate": 0.99,
      "volume": 1000000
    }
  ]
}
```

**구현 세부사항**:
- `data.investor_flow`와 `data.daily_prices` LEFT JOIN
- 가격 정보는 daily_prices에서 조회
- COALESCE로 null 처리

**구현 파일**:
- `backend/internal/api/handlers/chart_handler.go`
- `backend/internal/api/routes/chart_routes.go`

**에러**:
- `500`: DB 조회 실패

---

## 🔄 데이터 흐름

```
[StockDetailSheet]
       │
       ├─ [Chart 탭 클릭]
       │         │
       │         ▼
       │   [chart-tab.tsx]
       │         │
       │         ├─ useQuery('priceHistory')
       │         │         │
       │         │         └─► GET /api/v1/fetcher/prices/{code}/history
       │         │                        │
       │         │                        └─► data.daily_prices
       │         │
       │         ├─ useQuery('flowHistory')
       │         │         │
       │         │         └─► GET /api/v1/fetcher/flows/{code}/history
       │         │                        │
       │         │                        └─► data.investor_flow (LEFT JOIN daily_prices)
       │         │
       │         ▼
       │   [PriceChart 렌더링]
       │   [InvestorTradingChart 렌더링]
       │
       └─ [TanStack Query 자동 캐싱, 탭 전환 시 캐시 사용]
```

---

## ⚙️ 데이터 페칭 (TanStack Query)

**구현 방식**: 커스텀 훅 대신 TanStack Query 직접 사용

```typescript
// frontend/components/stock-detail-sheet/tabs/chart-tab.tsx

export function ChartTab({ symbol, symbolName, avgBuyPrice }: ChartTabProps) {
  // 일봉 데이터 조회 (최근 1년)
  const { data: priceData = [] } = useQuery({
    queryKey: ['priceHistory', symbol],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setFullYear(startDate.getFullYear() - 1)
      return getPriceHistory(symbol, startDate.toISOString().slice(0, 10), endDate)
    },
    enabled: !!symbol,
  })

  // 수급 데이터 조회 (최근 1년)
  const { data: flowData = [] } = useQuery({
    queryKey: ['flowHistory', symbol],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setFullYear(startDate.getFullYear() - 1)
      return getFlowHistory(symbol, startDate.toISOString().slice(0, 10), endDate)
    },
    enabled: !!symbol,
  })
}
```

**장점**:
- TanStack Query 자동 캐싱
- 로딩/에러 상태 자동 관리
- 재시도 로직 내장

---

## 🎯 구현 완료 (2026-01-17)

### Phase 2a: DB 마이그레이션 ✅

1. **DB 스키마 확인** ✅
   - `data.daily_prices` 파티션 테이블
   - `data.investor_flow` 파티션 테이블

2. **권한 설정** ✅
   - `GRANT USAGE ON SCHEMA data TO aegis_v14`
   - `GRANT SELECT ON ALL TABLES IN SCHEMA data TO aegis_v14`

### Phase 2b: Backend API 구현 ✅

1. **Handler 레이어** ✅
   - `internal/api/handlers/chart_handler.go`
     - `GetPriceHistory`: daily_prices 조회
     - `GetFlowHistory`: investor_flow + daily_prices LEFT JOIN

2. **라우팅 등록** ✅
   - `internal/api/routes/chart_routes.go`
     - `GET /api/v1/fetcher/prices/{code}/history`
     - `GET /api/v1/fetcher/flows/{code}/history`

3. **메인 서버 등록** ✅
   - `cmd/api/main.go`에 RegisterChartRoutes 추가

### Phase 2c: Frontend 차트 구현 ✅

1. **타입 정의** ✅
   - `types.ts`에 DailyPrice, InvestorFlow 추가

2. **API 함수** ✅
   - `lib/api.ts`에 getPriceHistory, getFlowHistory 추가

3. **차트 컴포넌트** ✅
   - `tabs/chart-tab.tsx` 통합 구현
     - `PriceChart`: Candlestick + 거래량 + Crosshair + 평단가
     - `InvestorTradingChart`: LineChart + 데이터 테이블

4. **StockDetailSheet 탭 추가** ✅
   - Chart 탭 등록 및 TabsList grid-cols-5로 확장

---

## 🧪 테스트 전략

### Backend 테스트

- [ ] Repository: `SELECT` 쿼리 테스트
- [ ] Service: 날짜 범위 필터링 테스트
- [ ] Handler: API 응답 포맷 테스트

### Frontend 테스트

- [ ] useDailyPrices: 데이터 로딩 테스트
- [ ] useInvestorTrading: 데이터 로딩 테스트
- [ ] PriceChart: Candlestick 렌더링 테스트
- [ ] InvestorTradingChart: 라인 차트 렌더링 테스트
- [ ] Chart 탭: 탭 전환 테스트

---

## 📦 의존성

### Frontend

| 패키지 | 용도 |
|--------|------|
| `recharts` | 차트 라이브러리 |
| `lucide-react` | 아이콘 |

### Backend

| 패키지 | 용도 |
|--------|------|
| `pgx/v5` | PostgreSQL 드라이버 |

---

## 🔗 관련 문서

- [docs/modules/stock-detail-sheet.md](../modules/stock-detail-sheet.md)
- [docs/database/schema.md](../database/schema.md)
- [docs/ui/pages.md](./pages.md)
- [CLAUDE.md](../../CLAUDE.md)

---

**작성일**: 2026-01-17
**업데이트**: 2026-01-17
**Phase**: Phase 2 완료 ✅ (v10 스타일 포팅 완료)
**구현 파일**:
- Backend: `chart_handler.go`, `chart_routes.go`
- Frontend: `chart-tab.tsx`, `types.ts`, `api.ts`
