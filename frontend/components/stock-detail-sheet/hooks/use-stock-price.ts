'use client'

import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getStockData } from '@/lib/api'
import type { PriceInfo } from '../types'

/**
 * 종목 가격 정보 조회 훅
 *
 * 1. Holdings 데이터에서 먼저 확인
 * 2. Holdings에 없으면 API에서 조회
 */
export function useStockPrice(symbol: string, holdings: any[]) {
  // Holdings 데이터에서 해당 종목 찾기
  const holdingPriceInfo = useMemo<PriceInfo | null>(() => {
    const holding = holdings.find((h) => h.symbol === symbol)
    if (!holding) return null

    const currentPrice = typeof holding.current_price === 'string'
      ? parseFloat(holding.current_price)
      : (holding.current_price || 0)

    // Holdings API에서 전일대비 정보 사용
    const changePrice = holding.change_price || 0
    const changeRate = holding.change_rate || 0

    return {
      currentPrice,
      changePrice,
      changeRate,
      prevClose: currentPrice - changePrice,
    }
  }, [symbol, holdings])

  // Holdings에 없는 경우 API에서 조회
  const { data: apiData, isLoading } = useQuery({
    queryKey: ['stock-price', symbol],
    queryFn: async () => {
      const result = await getStockData(symbol)
      return result
    },
    enabled: !!symbol && !holdingPriceInfo, // Holdings에 없을 때만 API 호출
    staleTime: 10000, // 10초
    refetchInterval: 30000, // 30초마다 갱신
  })

  // API 데이터에서 가격 정보 추출
  const apiPriceInfo = useMemo<PriceInfo | null>(() => {
    // latestFlow에서 가격 정보 가져오기 (더 상세한 정보)
    if (apiData?.latestFlow) {
      const flow = apiData.latestFlow
      return {
        currentPrice: flow.close_price || 0,
        changePrice: flow.price_change || 0,
        changeRate: flow.change_rate || 0,
        prevClose: (flow.close_price || 0) - (flow.price_change || 0),
      }
    }

    // latestPrice에서 기본 가격만 가져오기 (전일대비 없음)
    if (apiData?.latestPrice) {
      return {
        currentPrice: apiData.latestPrice.close || 0,
        changePrice: 0,
        changeRate: 0,
        prevClose: apiData.latestPrice.close || 0,
      }
    }

    return null
  }, [apiData])

  return {
    data: holdingPriceInfo || apiPriceInfo,
    isLoading: !holdingPriceInfo && isLoading,
  }
}

/**
 * 일봉 데이터 조회 훅 (Placeholder)
 *
 * Phase 2: Backend API 구현 후 활성화
 */
export function useStockDailyPrices(symbol: string) {
  // TODO: Implement
  return {
    data: null,
    isLoading: false,
  }
}
