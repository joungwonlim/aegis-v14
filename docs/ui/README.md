# UI (UI 설계)

이 폴더는 v14 시스템의 UI/UX 설계 문서를 포함합니다.

---

## 📋 문서 목록

### 1. pages.md
- **목적**: 페이지 구조 설계
- **내용**:
  - 전체 페이지 목록
  - 페이지 라우팅
  - 페이지별 기능
  - 네비게이션 구조

### 2. components.md
- **목적**: 컴포넌트 계층 설계
- **내용**:
  - UI 컴포넌트 목록 (shadcn/ui)
  - 도메인 컴포넌트 목록
  - 컴포넌트 재사용 전략
  - Props 인터페이스 설계

### 3. state-management.md
- **목적**: 상태 관리 전략
- **내용**:
  - 전역 상태 vs 로컬 상태
  - 상태 관리 도구 선택 (Zustand, Context, etc.)
  - 서버 상태 관리 (React Query, SWR)
  - 상태 구조 설계

### 4. api-integration.md
- **목적**: API 연동 방안
- **내용**:
  - API 클라이언트 설계
  - 에러 처리 전략
  - 로딩 상태 관리
  - 캐싱 전략

---

## 🎯 UI 설계 원칙

### 1. shadcn/ui 우선 사용
```tsx
// ✅ shadcn/ui 컴포넌트 사용
import { Button } from '@/shared/components/ui/button'
import { Card } from '@/shared/components/ui/card'

// ❌ 직접 스타일링 금지
<button className="px-4 py-2 bg-blue-500">
```

### 2. 컴포넌트 독립성
- 각 컴포넌트는 독립적으로 작동
- Props를 통한 데이터 전달
- Side-effect 최소화

### 3. 일관된 디자인 시스템
- Tailwind CSS 유틸리티 사용
- 디자인 토큰 활용
- 하드코딩 금지

### 4. 접근성 (Accessibility)
- ARIA 속성 사용
- 키보드 내비게이션 지원
- 색상 대비 고려

---

## 📐 페이지 구조 예시

```
app/
├── (auth)/
│   ├── login/
│   └── register/
├── (dashboard)/
│   ├── layout.tsx          # 대시보드 레이아웃
│   ├── page.tsx            # 대시보드 홈
│   ├── stocks/
│   │   ├── page.tsx        # 종목 목록
│   │   └── [code]/
│   │       └── page.tsx    # 종목 상세
│   ├── portfolio/
│   │   └── page.tsx        # 포트폴리오
│   ├── orders/
│   │   └── page.tsx        # 주문 내역
│   └── performance/
│       └── page.tsx        # 성과 분석
└── api/                    # API Routes (BFF)
```

---

## 🧩 컴포넌트 계층 구조

```
src/
├── shared/
│   ├── components/
│   │   ├── ui/            # shadcn/ui 컴포넌트
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   └── ...
│   │   └── layout/        # 레이아웃 컴포넌트
│   │       ├── header.tsx
│   │       ├── sidebar.tsx
│   │       └── footer.tsx
│   └── hooks/             # 공용 훅
│       ├── use-auth.ts
│       └── use-api.ts
└── modules/
    ├── stocks/
    │   ├── components/    # 도메인 컴포넌트
    │   │   ├── stock-card.tsx
    │   │   ├── stock-chart.tsx
    │   │   └── stock-table.tsx
    │   ├── hooks/
    │   │   └── use-stocks.ts
    │   ├── api.ts         # API 호출
    │   └── types.ts       # 타입 정의
    ├── portfolio/
    └── orders/
```

---

## 🎨 디자인 시스템

### 색상 팔레트 (shadcn/ui 기반)

```css
/* Primary */
--primary: 222.2 47.4% 11.2%;
--primary-foreground: 210 40% 98%;

/* Secondary */
--secondary: 210 40% 96.1%;
--secondary-foreground: 222.2 47.4% 11.2%;

/* Accent */
--accent: 210 40% 96.1%;
--accent-foreground: 222.2 47.4% 11.2%;

/* Destructive */
--destructive: 0 84.2% 60.2%;
--destructive-foreground: 210 40% 98%;

/* Custom (Trading) */
--positive: 142 76% 36%;  /* 상승: 녹색 */
--negative: 0 84% 60%;    /* 하락: 빨간색 */
```

### 타이포그래피

```tsx
// 숫자는 font-mono + tabular-nums 필수
<span className="font-mono tabular-nums">72,300원</span>
<span className="font-mono tabular-nums">+3.25%</span>
```

---

## 🔄 상태 관리 전략

### 1. 서버 상태 (React Query 권장)

```tsx
// 종목 조회
const { data, isLoading, error } = useQuery({
  queryKey: ['stocks', { market: 'KOSPI' }],
  queryFn: () => stocksApi.getList({ market: 'KOSPI' })
})
```

### 2. 전역 상태 (필요 시)

```tsx
// Zustand 예시
const useAuthStore = create((set) => ({
  user: null,
  setUser: (user) => set({ user }),
  logout: () => set({ user: null })
}))
```

### 3. 로컬 상태 (useState)

```tsx
// 컴포넌트 내부 상태
const [isOpen, setIsOpen] = useState(false)
```

---

## 🌐 API 연동 패턴

### API 클라이언트 설계

```typescript
// modules/stocks/api.ts
export const stocksApi = {
  getList: async (params: GetStocksParams) => {
    const response = await apiClient.get('/api/stocks', { params })
    return response.data
  },

  getDetail: async (code: string) => {
    const response = await apiClient.get(`/api/stocks/${code}`)
    return response.data
  }
}
```

### 에러 처리

```tsx
const { data, error } = useQuery({
  queryKey: ['stocks'],
  queryFn: stocksApi.getList,
  retry: 3,
  onError: (error) => {
    toast.error(error.message)
  }
})

if (error) {
  return <ErrorBoundary error={error} />
}
```

---

## 📱 반응형 디자인

### Breakpoints (Tailwind 기본)

```
sm: 640px
md: 768px
lg: 1024px
xl: 1280px
2xl: 1536px
```

### 반응형 컴포넌트 예시

```tsx
<div className="
  grid
  grid-cols-1
  sm:grid-cols-2
  lg:grid-cols-3
  xl:grid-cols-4
  gap-4
">
  {stocks.map(stock => <StockCard key={stock.code} stock={stock} />)}
</div>
```

---

## 🧪 테스트 전략

### 컴포넌트 테스트 (Vitest + Testing Library)

```tsx
import { render, screen } from '@testing-library/react'
import { StockCard } from './stock-card'

describe('StockCard', () => {
  it('renders stock information', () => {
    const stock = {
      code: '005930',
      name: '삼성전자',
      price: 72300
    }

    render(<StockCard stock={stock} />)

    expect(screen.getByText('005930')).toBeInTheDocument()
    expect(screen.getByText('삼성전자')).toBeInTheDocument()
    expect(screen.getByText('72,300원')).toBeInTheDocument()
  })
})
```

---

## ✅ 설계 검증 체크리스트

UI 설계 완료 시:

- [ ] 모든 페이지 정의
- [ ] 컴포넌트 계층 구조 설계
- [ ] 상태 관리 전략 정의
- [ ] API 연동 방안 정의
- [ ] 에러 처리 전략 정의
- [ ] 디자인 시스템 정의
- [ ] 반응형 설계 고려
- [ ] 접근성 고려

---

## 🔗 참고

- [CLAUDE.md](../../CLAUDE.md) - UI 설계 원칙
- [api/](../api/) - API 스펙
- [shadcn/ui Documentation](https://ui.shadcn.com/)
- [Next.js App Router](https://nextjs.org/docs/app)
- [Tailwind CSS](https://tailwindcss.com/)
- [React Query](https://tanstack.com/query/latest)
