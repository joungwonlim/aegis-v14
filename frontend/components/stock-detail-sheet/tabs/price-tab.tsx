'use client'

import { TrendingUp, TrendingDown } from 'lucide-react'
import type { PriceInfo } from '../types'

interface PriceTabProps {
  symbol: string
  symbolName: string
  priceInfo: PriceInfo | null
}

/**
 * Price 탭 - 가격 정보 표시
 *
 * Phase 1: 현재가, 전일대비 표시
 * TODO Phase 2: 차트, 52주 고저, 이동평균선
 */
export function PriceTab({ symbol, symbolName, priceInfo }: PriceTabProps) {
  if (!priceInfo) {
    return (
      <div className="p-6 text-center text-muted-foreground">
        가격 정보를 불러올 수 없습니다.
      </div>
    )
  }

  const { currentPrice, changePrice, changeRate, prevClose } = priceInfo
  const isPositive = changePrice >= 0

  return (
    <div className="space-y-4 p-4">
      {/* 현재가 헤더 */}
      <div className="border-b pb-4">
        <div className="flex items-baseline gap-2">
          <span className="text-3xl font-bold">
            {currentPrice.toLocaleString()}
          </span>
          <span className="text-sm text-muted-foreground">원</span>
        </div>
        <div className={`flex items-center gap-1 text-sm ${isPositive ? 'text-red-500' : 'text-blue-500'}`}>
          {isPositive ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
          <span className="font-medium">
            {isPositive ? '+' : ''}{changePrice.toLocaleString()}
          </span>
          <span>({isPositive ? '+' : ''}{changeRate.toFixed(2)}%)</span>
        </div>
      </div>

      {/* 가격 정보 */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <div className="text-xs text-muted-foreground">전일종가</div>
          <div className="text-lg font-medium">
            {prevClose?.toLocaleString() || '-'}
          </div>
        </div>

        <div>
          <div className="text-xs text-muted-foreground">시가</div>
          <div className="text-lg font-medium">
            {priceInfo.openPrice?.toLocaleString() || '-'}
          </div>
        </div>

        <div>
          <div className="text-xs text-muted-foreground">고가</div>
          <div className="text-lg font-medium text-red-500">
            {priceInfo.highPrice?.toLocaleString() || '-'}
          </div>
        </div>

        <div>
          <div className="text-xs text-muted-foreground">저가</div>
          <div className="text-lg font-medium text-blue-500">
            {priceInfo.lowPrice?.toLocaleString() || '-'}
          </div>
        </div>

        <div>
          <div className="text-xs text-muted-foreground">거래량</div>
          <div className="text-lg font-medium">
            {priceInfo.volume?.toLocaleString() || '-'}
          </div>
        </div>

        <div>
          <div className="text-xs text-muted-foreground">거래대금</div>
          <div className="text-lg font-medium">
            {priceInfo.value ? `${(priceInfo.value / 100000000).toFixed(0)}억` : '-'}
          </div>
        </div>
      </div>

      {/* TODO Phase 2: 차트 추가 */}
      <div className="rounded-lg border bg-muted/50 p-6 text-center text-sm text-muted-foreground">
        📊 차트는 Phase 2에서 추가됩니다
      </div>

      {/* TODO Phase 2: 52주 고저 */}
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-lg border p-3">
          <div className="text-xs text-muted-foreground">52주 최고</div>
          <div className="text-lg font-medium text-red-500">
            {priceInfo.high52w?.toLocaleString() || '-'}
          </div>
        </div>

        <div className="rounded-lg border p-3">
          <div className="text-xs text-muted-foreground">52주 최저</div>
          <div className="text-lg font-medium text-blue-500">
            {priceInfo.low52w?.toLocaleString() || '-'}
          </div>
        </div>
      </div>
    </div>
  )
}
