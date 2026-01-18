# Frontend 구현 계획

v14 Frontend 구현 작업 리스트 (Stocks 페이지 + StockDetailSheet 차트)

---

## 📋 작업 개요

**목적**: Stocks 페이지 및 StockDetailSheet 차트 기능 구현

**우선순위**:
1. **Phase 1**: Stocks 페이지 (Watchlist 재사용)
2. **Phase 2**: StockDetailSheet 차트 (v10 포팅)

---

## Phase 1: Stocks 페이지 구현

### 1-1. Backend API 구현

#### 1-1-1. Repository 레이어
**파일**: `backend/internal/repository/stocks.go`

**기능**:
- `ListStocks(ctx, filters, pagination)` - 전체 종목 목록 조회
- `CountStocks(ctx, filters)` - 총 종목 수 조회
- `SearchStocks(ctx, query)` - 종목 검색 (코드/이름)

**쿼리**:
```go
// ListStocks 예시
SELECT
  s.symbol,
  s.symbol_name,
  s.market,
  s.sector,
  s.market_cap,
  pb.current_price,
  pb.change_rate,
  pb.volume
FROM market.stocks s
LEFT JOIN market.prices_best pb ON s.symbol = pb.symbol
WHERE
  ($1::text IS NULL OR s.market = $1)  -- KOSPI/KOSDAQ 필터
  AND ($2::text IS NULL OR s.sector = $2)  -- 업종 필터
  AND ($3::text IS NULL OR s.symbol LIKE $3 OR s.symbol_name LIKE $3)  -- 검색
ORDER BY s.symbol ASC
LIMIT $4 OFFSET $5;  -- 페이징
```

#### 1-1-2. Service 레이어
**파일**: `backend/internal/service/stocks.go`

**기능**:
- `GetStockList(ctx, req)` - 종목 목록 조회 (필터링 + 페이징)

#### 1-1-3. Handler 레이어
**파일**: `backend/internal/api/handlers/stocks.go`

**엔드포인트**:
```
GET /api/stocks?page=1&limit=50&market=KOSPI&sort=symbol&order=asc&search=삼성
```

**응답**:
```json
{
  "success": true,
  "data": {
    "stocks": [...],
    "pagination": {
      "current_page": 1,
      "total_pages": 50,
      "total_count": 2500,
      "limit": 50
    }
  }
}
```

#### 1-1-4. 라우팅 등록
**파일**: `backend/internal/api/routes.go`

---

### 1-2. Frontend 페이지 구현

#### 1-2-1. Stocks 페이지
**파일**: `frontend/app/stocks/page.tsx`

**기능**:
- StockTable 컴포넌트 재사용
- 페이징 처리 (서버 사이드)
- 필터링 (시장/업종)
- 검색 (종목코드/이름)
- StockDetailSheet 통합

**컴포넌트 구조**:
```tsx
'use client'

import { useState } from 'react'
import { StockTable } from '@/components/stock-table'
import { StockDetailSheet } from '@/components/stock-detail-sheet'
import { useStockDetail } from '@/components/stock-detail-sheet/use-stock-detail'

export default function StocksPage() {
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({ market: 'ALL', sector: 'ALL' })
  const [search, setSearch] = useState('')

  const { data, isLoading } = useStocks({ page, filters, search })
  const { selectedStock, isOpen, openStockDetail, handleOpenChange } = useStockDetail()

  return (
    <div>
      {/* Filters Row */}
      <FiltersRow filters={filters} onFiltersChange={setFilters} />

      {/* Search */}
      <SearchBar search={search} onSearchChange={setSearch} />

      {/* Stock Table */}
      <StockTable
        stocks={data?.stocks}
        mode="all"
        showWatchlistActions={false}
        pagination={true}
        currentPage={page}
        totalPages={data?.pagination.total_pages}
        onPageChange={setPage}
        onStockClick={openStockDetail}
      />

      {/* StockDetailSheet */}
      <StockDetailSheet
        stock={selectedStock}
        open={isOpen}
        onOpenChange={handleOpenChange}
      />
    </div>
  )
}
```

#### 1-2-2. StockTable 컴포넌트 확장
**파일**: `frontend/components/stock-table.tsx` (기존 수정)

**Props 추가**:
```tsx
interface StockTableProps {
  // 기존 props...
  mode: 'watchlist' | 'all'
  showWatchlistActions?: boolean
  pagination?: boolean
  currentPage?: number
  totalPages?: number
  onPageChange?: (page: number) => void
  onStockClick?: (stock: StockInfo) => void
}
```

#### 1-2-3. useStocks 훅
**파일**: `frontend/hooks/use-stocks.ts`

**기능**:
- 종목 목록 API 호출
- 페이징 상태 관리
- 필터링 상태 관리

```tsx
export function useStocks(options: UseStocksOptions) {
  const [data, setData] = useState<StocksResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchStocks = async () => {
      const params = new URLSearchParams({
        page: String(options.page),
        limit: String(options.limit ?? 50),
        market: options.filters.market,
        sector: options.filters.sector,
        search: options.search,
      })
      const res = await fetch(`/api/stocks?${params}`)
      const json = await res.json()
      setData(json.data)
      setIsLoading(false)
    }

    fetchStocks()
  }, [options])

  return { data, isLoading }
}
```

---

## Phase 2: StockDetailSheet 차트 구현

### 2-1. DB 마이그레이션 (선행 필수)

**참조**: [docs/database/migration-charts.md](../database/migration-charts.md)

#### 2-1-1. v10 스키마 분석
- v10 DB 접속
- `market.daily_prices` 스키마 확인
- `market.investor_trading` 스키마 확인
- 샘플 데이터 조회

#### 2-1-2. v14 테이블 생성
**스크립트**: `scripts/db/06_create_chart_tables.sql`

```sql
CREATE TABLE market.daily_prices (...);
CREATE TABLE market.investor_trading (...);
-- 인덱스, 권한 설정
```

#### 2-1-3. 데이터 검증
- 레코드 수 확인
- 종목별 최신 데이터 확인
- NULL 체크

---

### 2-2. Backend API 구현

#### 2-2-1. Repository 레이어
**파일**: `backend/internal/repository/chart_data.go`

**기능**:
- `GetDailyPrices(ctx, symbol, days)` - 일봉 데이터 조회
- `GetInvestorTrading(ctx, symbol, days)` - 수급 데이터 조회

#### 2-2-2. Service 레이어
**파일**: `backend/internal/service/chart_data.go`

**기능**:
- `GetStockDailyPrices(ctx, symbol, days)` - 일봉 조회
- `GetStockInvestorTrading(ctx, symbol, days)` - 수급 조회

#### 2-2-3. Handler 레이어
**파일**: `backend/internal/api/handlers/chart_data.go`

**엔드포인트**:
```
GET /api/stocks/:symbol/daily-prices?days=90
GET /api/stocks/:symbol/investor-trading?days=90
```

#### 2-2-4. 라우팅 등록
**파일**: `backend/internal/api/routes.go`

---

### 2-3. Frontend 차트 구현

#### 2-3-1. recharts 설치
```bash
cd frontend
npm install recharts
```

#### 2-3-2. 타입 정의
**파일**: `frontend/components/stock-detail-sheet/components/charts/types.ts`

```tsx
export interface DailyPrice {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface InvestorTrading {
  date: string
  foreign_net: number
  inst_net: number
  indiv_net: number
  close_price: number
  price_change: number
  change_rate: number
  volume: number
}
```

#### 2-3-3. 훅 구현
**파일**: `frontend/components/stock-detail-sheet/hooks/use-daily-prices.ts`
**파일**: `frontend/components/stock-detail-sheet/hooks/use-investor-trading.ts`

(설계 문서 참조: [docs/ui/charts.md](../ui/charts.md))

#### 2-3-4. PriceChart 컴포넌트
**파일**: `frontend/components/stock-detail-sheet/components/charts/price-chart.tsx`

**기능**:
- v10 PriceChart.tsx 포팅
- Candlestick 렌더링
- 거래량 차트
- 기간 필터 (1M, 3M, 6M, 1Y)
- 평단가 선 (보유 종목만)
- Crosshair

#### 2-3-5. InvestorTradingChart 컴포넌트
**파일**: `frontend/components/stock-detail-sheet/components/charts/investor-trading-chart.tsx`

**기능**:
- v10 InvestorTradingChart.tsx 포팅
- 3개 라인 (외국인/기관/개인)
- 기간 필터 (1M, 3M, 6M, 1Y)
- 0 기준선
- 데이터 테이블 (최근 10일)

#### 2-3-6. Chart 탭
**파일**: `frontend/components/stock-detail-sheet/tabs/chart-tab.tsx`

```tsx
'use client'

import { PriceChart } from '../components/charts/price-chart'
import { InvestorTradingChart } from '../components/charts/investor-trading-chart'
import { useDailyPrices } from '../hooks/use-daily-prices'
import { useInvestorTrading } from '../hooks/use-investor-trading'

interface ChartTabProps {
  symbol: string
  avgBuyPrice?: number  // 보유 종목인 경우
}

export function ChartTab({ symbol, avgBuyPrice }: ChartTabProps) {
  const { data: dailyPrices, isLoading: pricesLoading } = useDailyPrices(symbol, { days: 90 })
  const { data: investorTrading, isLoading: tradingLoading } = useInvestorTrading(symbol, { days: 90 })

  return (
    <div className="space-y-6">
      {/* 일봉 차트 */}
      <PriceChart
        data={dailyPrices}
        isLoading={pricesLoading}
        avgBuyPrice={avgBuyPrice}
      />

      {/* 수급 차트 */}
      <InvestorTradingChart
        data={investorTrading}
        isLoading={tradingLoading}
      />
    </div>
  )
}
```

#### 2-3-7. StockDetailSheet 탭 추가
**파일**: `frontend/components/stock-detail-sheet/stock-detail-sheet.tsx` (기존 수정)

```tsx
// Tabs에 Chart 탭 추가
<Tabs defaultValue="holding">
  <TabsList>
    <TabsTrigger value="holding">보유</TabsTrigger>
    <TabsTrigger value="price">가격</TabsTrigger>
    <TabsTrigger value="chart">차트</TabsTrigger>  {/* NEW */}
    <TabsTrigger value="order">주문</TabsTrigger>
  </TabsList>

  <TabsContent value="holding">
    <HoldingTab {...} />
  </TabsContent>

  <TabsContent value="price">
    <PriceTab {...} />
  </TabsContent>

  <TabsContent value="chart">
    <ChartTab symbol={stock.symbol} avgBuyPrice={holding?.avg_buy_price} />
  </TabsContent>

  <TabsContent value="order">
    <OrderTab {...} />
  </TabsContent>
</Tabs>
```

---

## 테스트 계획

### Backend 테스트

- [ ] `/api/stocks` - 종목 목록 조회 (페이징)
- [ ] `/api/stocks` - 필터링 (시장/업종)
- [ ] `/api/stocks` - 검색 (종목코드/이름)
- [ ] `/api/stocks/:symbol/daily-prices` - 일봉 조회
- [ ] `/api/stocks/:symbol/investor-trading` - 수급 조회

### Frontend 테스트

- [ ] Stocks 페이지 렌더링
- [ ] 페이징 버튼 클릭
- [ ] 필터 변경 (시장/업종)
- [ ] 검색 입력
- [ ] 종목명 클릭 시 StockDetailSheet 열림
- [ ] Chart 탭 렌더링
- [ ] PriceChart Candlestick 렌더링
- [ ] InvestorTradingChart 라인 렌더링
- [ ] 기간 필터 변경 (1M, 3M, 6M, 1Y)

---

## 의존성

### Frontend

| 패키지 | 버전 | 용도 |
|--------|------|------|
| `recharts` | `^3.6.0` | 차트 라이브러리 |
| `lucide-react` | (기존) | 아이콘 |

### Backend

| 패키지 | 버전 | 용도 |
|--------|------|------|
| `pgx/v5` | (기존) | PostgreSQL 드라이버 |

---

## 우선순위 요약

### Phase 1 (높음)
1. Backend API: `/api/stocks` 구현
2. Frontend: Stocks 페이지 구현
3. StockTable 컴포넌트 확장 (mode, pagination)
4. 통합 테스트 (Stocks 페이지)

### Phase 2 (중간)
1. v10 DB 스키마 분석
2. v14 차트 테이블 마이그레이션
3. Backend API: 차트 엔드포인트 구현
4. Frontend: 차트 컴포넌트 포팅 (v10 → v14)
5. Chart 탭 통합
6. 통합 테스트 (차트 기능)

---

## 완료 조건 (DoD)

- [ ] Backend: 모든 API 엔드포인트 구현 완료
- [ ] Frontend: Stocks 페이지 렌더링 정상
- [ ] Frontend: StockDetailSheet 차트 탭 렌더링 정상
- [ ] 테스트: 모든 테스트 통과
- [ ] 문서: API 문서 업데이트
- [ ] Git: 커밋 완료 (`feat(frontend): Stocks 페이지 및 차트 구현`)

---

**작성일**: 2026-01-17
**Phase**: 구현 대기 (PHASE=IMPLEMENT 전환 필요)
