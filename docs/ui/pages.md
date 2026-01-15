# 페이지 설계 (Pages Design)

v14 시스템의 페이지 구조 및 기능 설계

---

## 📋 페이지 목록

### 1. 대시보드 (Dashboard) - `/`

**목적**: 실시간 트레이딩 엔진 모니터링

**주요 기능**:
- Portfolio (보유종목) 실시간 표시
- Exit Engine 청산 대상 모니터링
- KIS Orders Execution 승인 대기
- KIS 미체결/체결 주문 표시

**컴포넌트**:
- `Card` (shadcn/ui)
- `Table` (shadcn/ui)
- `StockSymbol` (custom)
- `StockDetailSheet` (custom)

---

## 📊 Portfolio 섹션 설계

### 목적
보유종목 현황 및 실시간 가격 동기화 표시

### 테이블 구조

| 컬럼명 | 타입 | 정렬 가능 | 설명 |
|--------|------|-----------|------|
| 종목명 | string | ✅ | 종목명 + 로고 (StockSymbol) |
| 보유수량 | number | ✅ | 보유 주식 수량 |
| 매도가능 | number | ❌ | 매도 가능 수량 |
| 평가손익 | number | ✅ | 평가손익 금액 |
| 수익률 | number | ✅ | 평가손익률 (%) |
| 매입단가 | number | ✅ | 평균 매입 단가 |
| 현재가 | number | ✅ | 실시간 현재가 |
| 평가금액 | number | ✅ | 현재 평가 금액 |
| 매입금액 | number | ✅ | 총 매입 금액 |
| 비중 | number | ✅ | 포트폴리오 내 비중 (%) |

### 정렬 정책

**기본 정렬**: 평가금액 내림차순 (높은 순)

**사유**:
- 포트폴리오에서 가장 중요한 지표는 평가금액
- 큰 포지션부터 확인하는 것이 리스크 관리에 유리
- 사용자가 가장 자주 확인하는 정렬 순서

**구현**:
```tsx
const [sortField, setSortField] = useState<SortField>('eval_amount')
const [sortOrder, setSortOrder] = useState<SortOrder>('desc')
```

### 상태 표시

**보유 상태**:
- 보유종목: 종목명 뒤 녹색점 (●)
- Exit Engine 활성화: 종목명 뒤 빨간점 (●)

**시장 정보**:
- 종목코드 뒤 KOSPI/KOSDAQ 표시

### 상호작용

**클릭 동작**:
- 종목명 클릭 → StockDetailSheet 열림
- 컬럼 헤더 클릭 → 해당 컬럼 정렬 토글

**자동 갱신**:
- 10초마다 자동 새로고침
- 수동 새로고침 버튼 제공

---

## 🎯 Exit Engine 섹션 설계

### 목적
Exit 규칙 평가 및 청산 주문 의도 표시

### 테이블 구조

| 컬럼명 | 타입 | 정렬 가능 | 설명 |
|--------|------|-----------|------|
| 종목명 | string | ✅ | 종목명 + 로고 |
| 현재가 | number | ✅ | 실시간 현재가 |
| 전일대비 | number | ❌ | 전일 대비 등락률 |
| 매입단가 | number | ❌ | 평균 매입 단가 |
| 주문가격 | number | ✅ | Exit Intent 주문가격 |
| 괴리율 | number | ✅ | 현재가 vs 주문가격 괴리 |
| 타입 | string | ❌ | EXIT_PARTIAL, EXIT_FULL |
| 수량 | number | ✅ | 주문 수량 |
| 주문유형 | string | ❌ | MKT, LMT |
| 사유 | string | ❌ | SL1, SL2, TP1, TP2, TRAILING |
| 상태 | string | ❌ | PENDING_APPROVAL, NEW, ACK, FILLED |
| 생성시각 | timestamp | ✅ | Intent 생성 시각 |

### 정렬 정책

**기본 정렬**: 생성시각 내림차순 (최신 순)

**사유**:
- 가장 최근 생성된 Intent가 우선 확인 대상
- 시간순 처리가 직관적

---

## 📤 KIS Orders Execution 섹션 설계

### 목적
승인 대기 중인 Exit Intent 관리

### 액션 버튼
- "주문 실행" (녹색): Intent 승인 → KIS 주문 제출
- "삭제" (빨간색): Intent 거부

### 상태별 표시
- `PENDING_APPROVAL`: 액션 버튼 표시
- `NEW`: "주문 대기 중" 뱃지
- `SUBMITTED`: "주문 완료" 뱃지

---

## ⏳ KIS 미체결 주문 섹션 설계

### 목적
KIS에 제출되었으나 아직 체결되지 않은 주문 표시

### 통계 표시
- 총 건수
- 매수/매도 건수
- 총 금액

---

## ✅ KIS 체결 주문 섹션 설계

### 목적
KIS에서 체결 완료된 주문 표시

### 통계 표시
- 총 건수
- 매수/매도 건수 및 금액

---

## 🎨 공통 UI 패턴

### 테이블 정렬
- 정렬 가능 컬럼: 헤더 호버 시 배경색 변경
- 현재 정렬 컬럼: 화살표 표시 (↑ 오름차순, ↓ 내림차순)
- 클릭 동작: 같은 컬럼 클릭 시 정렬 순서 토글

### 숫자 표시
- 폰트: `font-mono tabular-nums`
- 정렬: `text-right`
- 천단위 구분: `toLocaleString('ko-KR')`

### 색상 코드
- 상승/이익: `#EA5455` (빨간색)
- 하락/손실: `#2196F3` (파란색)
- 중립: `text-muted-foreground`

---

## 🔄 상태 관리

### 로컬 상태
```tsx
const [holdings, setHoldings] = useState<Holding[]>([])
const [intents, setIntents] = useState<OrderIntent[]>([])
const [sortField, setSortField] = useState<SortField>('eval_amount') // 기본 정렬
const [sortOrder, setSortOrder] = useState<SortOrder>('desc')
```

### API 폴링
- 간격: 10초
- 에러 처리: 일부 API 실패 시 에러 메시지 표시, 나머지는 정상 표시

---

## 📱 반응형

현재는 데스크톱 우선 (모바일 미지원)

---

## 🔗 관련 문서

- [CLAUDE.md](../../CLAUDE.md)
- [docs/ui/README.md](./README.md)
- [docs/api/runtime-api.md](../api/runtime-api.md)
