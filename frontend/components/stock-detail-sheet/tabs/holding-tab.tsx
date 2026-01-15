'use client'

import { Package } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'

interface HoldingTabProps {
  symbol: string
  symbolName: string
  holding: any | null
  totalEvaluation: number
  onExitModeToggle: (enabled: boolean) => void
}

/**
 * Holding 탭 - 보유 정보 + Exit Engine 스위치
 *
 * 기존 Sheet 기능 통합
 */
export function HoldingTab({
  symbol,
  symbolName,
  holding,
  totalEvaluation,
  onExitModeToggle,
}: HoldingTabProps) {
  if (!holding) {
    return (
      <div className="p-6 text-center text-muted-foreground">
        보유 정보가 없습니다.
      </div>
    )
  }

  const evaluateAmount = holding.raw?.evaluate_amount || (holding.qty * holding.current_price).toString()
  const purchaseAmount = holding.raw?.purchase_amount || (holding.qty * holding.avg_price).toString()
  const weight = totalEvaluation > 0 ? (parseInt(evaluateAmount) / totalEvaluation) * 100 : 0
  const pnl = typeof holding.pnl === 'string' ? parseFloat(holding.pnl) : holding.pnl
  const currentPrice = typeof holding.current_price === 'string' ? parseFloat(holding.current_price) : holding.current_price
  const avgPrice = typeof holding.avg_price === 'string' ? parseFloat(holding.avg_price) : holding.avg_price
  const priceColor = pnl >= 0 ? '#EA5455' : '#2196F3'

  const formatNumber = (n: number | null | undefined, decimals = 0) => {
    if (n == null || n === 0) return '-'
    return decimals > 0 ? n.toFixed(decimals) : Math.round(n).toLocaleString()
  }

  const formatPnL = (n: number | null | undefined) => {
    if (n == null) return '-'
    const sign = n >= 0 ? '+' : ''
    return `${sign}${Math.round(n).toLocaleString()}`
  }

  const formatPercent = (n: number | null | undefined) => {
    if (n == null) return '-'
    const sign = n >= 0 ? '+' : ''
    return `${sign}${n.toFixed(2)}%`
  }

  return (
    <div className="space-y-6 p-4">
      {/* 종목명 */}
      <div className="text-lg font-semibold text-muted-foreground">
        {symbolName}
      </div>

      {/* 현재가 (큰 숫자) */}
      <div className="space-y-2">
        <div className="text-4xl font-bold" style={{ color: priceColor }}>
          {formatNumber(currentPrice, 0)}원
        </div>
        <div className="flex items-center gap-4">
          <span style={{ color: priceColor }} className="text-xl font-semibold">
            {formatPnL(pnl)}
          </span>
          <span style={{ color: priceColor }} className="text-xl font-semibold">
            {formatPercent(holding.pnl_pct)}
          </span>
        </div>
      </div>

      {/* 구분선 */}
      <div className="border-t border-border"></div>

      {/* Exit Engine 스위치 */}
      <div className="flex items-center justify-between rounded-lg border p-4">
        <div className="space-y-0.5">
          <Label htmlFor="exit-mode" className="text-base font-semibold">
            Exit Engine
          </Label>
          <div className="text-sm text-muted-foreground">
            {holding.exit_mode === 'ENABLED' ? '활성화됨' : '비활성화됨'}
          </div>
        </div>
        <Switch
          id="exit-mode"
          checked={holding.exit_mode === 'ENABLED'}
          onCheckedChange={onExitModeToggle}
        />
      </div>

      {/* 구분선 */}
      <div className="border-t border-border"></div>

      {/* 보유 정보 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">보유수량</span>
          <span className="font-mono font-semibold">{formatNumber(holding.qty)}주</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">매도가능</span>
          <span className="font-mono font-semibold">{formatNumber(holding.qty)}주</span>
        </div>
      </div>

      {/* 구분선 */}
      <div className="border-t border-border"></div>

      {/* 손익 정보 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">평가손익</span>
          <span className="font-mono font-semibold" style={{ color: priceColor }}>
            {formatPnL(pnl)}원
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">수익률</span>
          <span className="font-mono font-semibold" style={{ color: priceColor }}>
            {formatPercent(holding.pnl_pct)}
          </span>
        </div>
      </div>

      {/* 구분선 */}
      <div className="border-t border-border"></div>

      {/* 가격 정보 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">매입단가</span>
          <span className="font-mono font-semibold">{formatNumber(avgPrice, 0)}원</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">현재가</span>
          <span className="font-mono font-semibold" style={{ color: priceColor }}>
            {formatNumber(currentPrice, 0)}원
          </span>
        </div>
      </div>

      {/* 구분선 */}
      <div className="border-t border-border"></div>

      {/* 금액 정보 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">평가금액</span>
          <span className="font-mono font-semibold">{formatNumber(parseInt(evaluateAmount), 0)}원</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">매입금액</span>
          <span className="font-mono font-semibold">{formatNumber(parseInt(purchaseAmount), 0)}원</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground">비중</span>
          <span className="font-mono font-semibold">{weight.toFixed(1)}%</span>
        </div>
      </div>
    </div>
  )
}
