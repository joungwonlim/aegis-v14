# StockDetailSheet 모듈 설계

## 개요

v10의 StockDetailSheet 기능을 v14에 모듈 단위로 포팅한 독립 모듈

- **책임**: 종목 상세 정보 표시 (Sheet UI)
- **위치**: `frontend/components/stock-detail-sheet/`
- **Phase**: Phase 1 완료 (Holding, Price, Order 탭 + 빠른 주문 기능)

---

## 모듈 구조

```
frontend/components/stock-detail-sheet/
├── stock-detail-sheet.tsx         # 메인 Sheet 컴포넌트
├── use-stock-detail.ts            # 상태 관리 훅
├── types.ts                       # 타입 정의
├── index.ts                       # Export
├── tabs/
│   ├── holding-tab.tsx            # ✅ Phase 1: 보유 정보 탭
│   ├── price-tab.tsx              # ✅ Phase 1: 가격 정보 탭
│   └── order-tab.tsx              # ✅ Phase 1: 주문 탭 (주문 목록 + 빠른 주문)
└── hooks/
    └── use-stock-price.ts         # ✅ Phase 1: 가격 정보 조회 훅
```

---

## Phase 1: 구현 완료 기능

### 1. Holding 탭 ✅

**기능**:
- 보유수량, 매도가능 수량 표시
- 평가손익, 수익률 표시
- 매입단가 표시
- 평가금액, 매입금액 표시
- 포트폴리오 비중 표시

**참고**: Exit Engine 스위치는 상단 아이콘 버튼(Settings)으로 이동

**데이터 소스**:
- v14 기존 `Holdings` 데이터
- Exit Engine 상태 업데이트 API

**구현 파일**:
- `tabs/holding-tab.tsx`

### 2. Price 탭 ✅

**기능**:
- 현재가, 전일대비, 등락률 표시
- 시가/고가/저가 표시
- 거래량/거래대금 표시
- 52주 최고/최저 표시 (Placeholder)

**데이터 소스**:
- v14 기존 `Holdings` 데이터
- `useStockPrice` 훅으로 조회

**구현 파일**:
- `tabs/price-tab.tsx`
- `hooks/use-stock-price.ts`

### 3. Order 탭 ✅

**기능**:
- 미체결 주문 목록 (해당 종목만 필터링)
- 체결 주문 목록 (해당 종목만 필터링)
- 매수/매도 구분 표시
- 주문가, 체결가, 수량 표시
- **빠른 주문 기능 (v10 스타일)** ✅
  - 보유 정보 표시 (보유수량, 평가손익, 평균매수단가, 손익률)
  - 매수/매도 큰 토글 버튼
  - 지정가/시장가 큰 토글 버튼
  - 주문가격 입력 + 증감 버튼 (↑↓)
  - 주문수량 입력 + 빠른 조절 버튼 (+10, +50, +100, +1000)
  - 현재가 적용 버튼
  - 총 주문금액 자동 계산
  - 한할 매수/매도하기 버튼 (큰 버튼)
  - 실시간 주문 제출 (KIS API 연동)
  - 성공/실패 결과 메시지 표시
  - 중복 실행 방지

**데이터 소스**:
- v14 기존 `KisUnfilledOrders`
- v14 기존 `KisFilledOrders`
- v14 기존 `KisAdapter.SubmitOrder` (주문 제출)

**API 엔드포인트**:
- `POST /api/kis/orders` - KIS 주문 제출

**구현 파일**:
- `tabs/order-tab.tsx`
- `lib/api.ts` (placeKISOrder 함수)
- Backend: `internal/api/handlers/kis_orders.go` (PlaceOrder 핸들러)

### 4. 기본 컴포넌트 ✅

**StockDetailSheet**:
- shadcn/ui Sheet 기반
- Tabs로 탭 전환
- StockSymbol 컴포넌트로 종목 표시
- v10과 동일한 UI/UX
- **SheetHeader 구조**:
  - 종목 로고 + 종목명 + 마켓/섹터 정보
  - 종목명 (text-lg, 회색)
  - 현재가 (text-4xl, 큰 숫자, 빨강/파랑)
  - 전일대비 + 등락률 (text-xl, 빨강/파랑)
  - Exit Engine 스위치 (왼쪽)
  - 아이콘 버튼 그룹 (오른쪽):
    - Settings (h-10 w-10): Exit Rule 상세 설정
    - BarChart3 (h-10 w-10): 차트 보기 (Phase 2)
    - ExternalLink (h-10 w-10): 외부 링크 (Phase 2)
- **Exit Rule 다이얼로그**:
  - Settings 버튼 클릭 시 열림
  - Exit Engine 스위치 (중복, 빠른 접근용)
  - 상세 설정은 Phase 2 예정

**useStockDetail 훅**:
- Sheet 열기/닫기 상태 관리
- v10과 동일한 인터페이스 제공

---

## Phase 2: 예정 기능 (v10 DB 필요)

### Investment 탭 (투자 지표)

**필요 데이터**:
- `market.fundamentals` 테이블 (v10 마이그레이션)
  - PER, PBR, ROE, ROA
  - 배당수익률, 주당배당금
  - 시가총액, 상장주식수

### Consensus 탭 (컨센서스)

**필요 데이터**:
- `market.consensus` 테이블 (v10 마이그레이션)
  - 목표주가
  - 컨센서스 점수
  - 매수/보유/매도 의견 수

### AI Analysis 탭

**필요 데이터**:
- `stock_ai_analysis` 테이블 (v10에 이미 존재)
  - ChatGPT, DeepSeek 분석 히스토리
  - 매수/매도/보유 추천
  - 목표가, 신뢰도 점수

---

## 타입 정의

### StockInfo

```typescript
export interface StockInfo {
  symbol: string      // 종목코드 (6자리)
  symbolName: string  // 종목명
  market?: string     // KOSPI/KOSDAQ
  sector?: string     // 업종
}
```

### PriceInfo

```typescript
export interface PriceInfo {
  currentPrice: number     // 현재가
  changePrice: number      // 전일대비 (가격)
  changeRate: number       // 전일대비 (%)
  openPrice?: number       // 시가
  highPrice?: number       // 고가
  lowPrice?: number        // 저가
  volume?: number          // 거래량
  value?: number           // 거래대금
  high52w?: number         // 52주 최고가
  low52w?: number          // 52주 최저가
  prevClose?: number       // 전일종가
}
```

---

## 사용 방법

### 1. useStockDetail 훅 사용

```tsx
import { useStockDetail } from '@/components/stock-detail-sheet'

export default function Page() {
  const { selectedStock, isOpen, openStockDetail, handleOpenChange } = useStockDetail()

  const handleClick = (symbol: string, symbolName: string) => {
    openStockDetail({ symbol, symbolName })
  }

  return (
    <>
      <button onClick={() => handleClick('005930', '삼성전자')}>
        삼성전자 상세 보기
      </button>

      <StockDetailSheet
        stock={selectedStock}
        open={isOpen}
        onOpenChange={handleOpenChange}
        holdings={holdings}
        unfilledOrders={unfilledOrders}
        executedOrders={executedOrders}
      />
    </>
  )
}
```

### 2. StockSymbol 클릭 통합

```tsx
<StockSymbol
  symbol={holding.symbol}
  symbolName={holding.raw?.symbol_name}
  size="sm"
  onClick={() => openStockDetail({
    symbol: holding.symbol,
    symbolName: holding.raw?.symbol_name || holding.symbol,
  })}
/>
```

---

## 의존성

### Frontend 의존성

| 패키지 | 용도 |
|--------|------|
| `shadcn/ui` | Sheet, Tabs, Table UI |
| `@/components/stock-symbol` | 종목 로고/이름 표시 |
| `@/components/ui/table` | 주문 테이블 |
| `lucide-react` | 아이콘 |

### 데이터 의존성

| 데이터 | 소스 | Phase |
|--------|------|-------|
| Holdings | v14 기존 API | Phase 1 ✅ |
| KisUnfilledOrders | v14 기존 API | Phase 1 ✅ |
| KisFilledOrders | v14 기존 API | Phase 1 ✅ |
| Fundamentals | v10 DB (미구현) | Phase 2 ⚠️ |
| Consensus | v10 DB (미구현) | Phase 2 ⚠️ |
| AI Analysis | v10 DB (미구현) | Phase 2 ⚠️ |

---

## 모듈 독립성

### ✅ 기존 코드와 완전 분리

- 새로운 디렉토리 `components/stock-detail-sheet/` 생성
- 기존 코드 수정 최소화 (page.tsx에 import/통합만)
- 독립적으로 테스트 가능

### ✅ v10 호환 인터페이스

- `useStockDetail` 훅은 v10과 동일한 API 제공
- 향후 v10 코드 마이그레이션 시 최소 수정

---

## 성능 고려사항

### 최적화

- `useMemo`로 Price 계산 캐싱
- 종목별 주문 필터링 최적화
- Sheet lazy loading (필요 시)

### Phase 2 고려사항

- 일봉 데이터: 차트 렌더링 시 대량 데이터 처리
- AI 분석: 마크다운 파싱 성능

---

## 테스트 전략

### Phase 1 테스트 완료

- [x] StockDetailSheet 렌더링 테스트
- [x] Price 탭 데이터 표시 테스트
- [x] Order 탭 필터링 테스트
- [x] useStockDetail 훅 상태 관리 테스트
- [x] 빌드 테스트 (TypeScript, Next.js)

### Phase 2 테스트 (TODO)

- [ ] Investment 탭 데이터 연동
- [ ] Consensus 탭 데이터 연동
- [ ] AI 탭 데이터 연동

---

## 마이그레이션 계획 (Phase 2)

### 1단계: v10 DB 테이블 마이그레이션

```sql
-- v10 → v14 마이그레이션
CREATE TABLE market.fundamentals (...)
CREATE TABLE market.consensus (...)
CREATE TABLE market.ai_analysis (...)
```

### 2단계: Backend API 구현

```
GET /api/stocks/:symbol/fundamentals
GET /api/stocks/:symbol/consensus
GET /api/stocks/:symbol/ai-analysis
```

### 3단계: Frontend 탭 구현

```
tabs/investment-tab.tsx
tabs/consensus-tab.tsx
tabs/ai-tab.tsx
```

---

## 버전 히스토리

| 버전 | 날짜 | 내용 |
|------|------|------|
| v14.1.0-phase1 | 2026-01-15 | Phase 1 완료 (Holding, Price, Order 탭) |
| v14.2.0-phase2 | TBD | Phase 2 (Investment, Consensus, AI 탭) |

---

**작성일**: 2026-01-15
**Phase**: Phase 1 완료
**다음 단계**: v10 DB 마이그레이션 (Phase 2)
