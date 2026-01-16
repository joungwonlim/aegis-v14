'use client'

import { useMemo } from 'react'
import type { PriceInfo } from '../types'

/**
 * 종목 가격 정보 조회 훅
 *
 * Phase 1: Holdings 데이터 활용
 * TODO Phase 2: 일봉 데이터 추가
 */
export function useStockPrice(symbol: string, holdings: any[]) {
  const priceInfo = useMemo<PriceInfo | null>(() => {
    // Holdings 데이터에서 해당 종목 찾기
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

  return {
    data: priceInfo,
    isLoading: false, // Holdings는 이미 로드된 상태
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
