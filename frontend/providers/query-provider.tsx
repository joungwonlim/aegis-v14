'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 1000, // 1초
            refetchInterval: 3000, // 3초마다 자동 갱신
            refetchOnWindowFocus: false, // 윈도우 포커스 시 자동 갱신 비활성화
            retry: 1, // 실패 시 1회만 재시도
          },
        },
      })
  )

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}
