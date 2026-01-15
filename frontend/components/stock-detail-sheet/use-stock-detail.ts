'use client'

import { useState, useCallback } from 'react'
import type { StockInfo } from './types'

/**
 * 종목 상세 시트 상태 관리 훅
 *
 * v10 useStockDetail과 동일한 인터페이스 제공
 */
export function useStockDetail() {
  const [selectedStock, setSelectedStock] = useState<StockInfo | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  const openStockDetail = useCallback((stock: StockInfo) => {
    setSelectedStock(stock)
    setIsOpen(true)
  }, [])

  const closeStockDetail = useCallback(() => {
    setIsOpen(false)
    // 애니메이션 후 상태 초기화
    setTimeout(() => setSelectedStock(null), 300)
  }, [])

  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) {
      closeStockDetail()
    }
  }, [closeStockDetail])

  return {
    selectedStock,
    isOpen,
    openStockDetail,
    closeStockDetail,
    handleOpenChange,
  }
}
