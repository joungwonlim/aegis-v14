'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getStockInfo, refreshStockInfo, type StockInfoDetail } from '@/lib/api'

/**
 * 종목 기업 개요 조회 hook
 * DB에 없으면 네이버증권에서 자동으로 가져옴
 */
export function useStockInfo(symbol: string) {
  return useQuery({
    queryKey: ['stock-info', symbol],
    queryFn: () => getStockInfo(symbol),
    enabled: !!symbol,
    staleTime: 5 * 60 * 1000, // 5분간 캐시
    gcTime: 30 * 60 * 1000, // 30분간 가비지 컬렉션 방지
  })
}

/**
 * 종목 기업 개요 강제 새로고침 mutation
 */
export function useRefreshStockInfo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (symbol: string) => refreshStockInfo(symbol),
    onSuccess: (data, symbol) => {
      // 캐시 업데이트
      queryClient.setQueryData(['stock-info', symbol], data)
    },
  })
}
