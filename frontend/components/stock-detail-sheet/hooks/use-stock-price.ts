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
    const holding = holdings.find((h) => h.Symbol === symbol)
    if (!holding) return null

    const currentPrice = holding.CurrentPrice || 0
    const prevClose = holding.Raw?.prpr || currentPrice // 전일종가 (Raw 데이터)
    const changePrice = currentPrice - prevClose
    const changeRate = prevClose > 0 ? (changePrice / prevClose) * 100 : 0

    return {
      currentPrice,
      changePrice,
      changeRate,
      prevClose,
      // TODO: Holdings에서 제공하는 추가 필드가 있으면 매핑
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
