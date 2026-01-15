'use client'

import { useState } from 'react'
import { X, TrendingUp, ShoppingCart, DollarSign, Target, Brain, Package } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StockSymbol } from '@/components/stock-symbol'
import type { StockInfo, StockDetailTab } from './types'
import { useStockPrice } from './hooks/use-stock-price'
import { HoldingTab } from './tabs/holding-tab'
import { PriceTab } from './tabs/price-tab'
import { OrderTab } from './tabs/order-tab'

interface StockDetailSheetProps {
  stock: StockInfo | null
  open: boolean
  onOpenChange: (open: boolean) => void
  holdings: any[]           // Holdings 데이터
  unfilledOrders: any[]     // 미체결 주문
  executedOrders: any[]     // 체결 주문
  totalEvaluation: number   // 총 평가금액
  onExitModeToggle: (holding: any, enabled: boolean) => void  // Exit Engine 토글
}

/**
 * 종목 상세 Sheet (v10 스타일)
 *
 * Phase 1: Holding, Price, Order 탭
 * Phase 2: Investment, Consensus, AI 탭 추가
 */
export function StockDetailSheet({
  stock,
  open,
  onOpenChange,
  holdings,
  unfilledOrders,
  executedOrders,
  totalEvaluation,
  onExitModeToggle,
}: StockDetailSheetProps) {
  const [activeTab, setActiveTab] = useState<StockDetailTab>('holding')

  // 가격 정보 조회
  const { data: priceInfo } = useStockPrice(stock?.symbol || '', holdings)

  // 해당 종목의 보유 정보 찾기
  const holding = holdings.find((h) => h.symbol === stock?.symbol)

  if (!stock) return null

  const tabs = [
    { key: 'holding', label: '보유', icon: Package },
    { key: 'price', label: '시세', icon: TrendingUp },
    { key: 'order', label: '주문', icon: ShoppingCart },
    // Phase 2에서 활성화
    // { key: 'investment', label: '투자지표', icon: DollarSign },
    // { key: 'consensus', label: '컨센서스', icon: Target },
    // { key: 'ai', label: 'AI분석', icon: Brain },
  ] as const

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto">
        <SheetHeader className="border-b pb-4">
          <SheetTitle className="flex items-start justify-between">
            <div className="flex-1">
              <StockSymbol
                symbol={stock.symbol}
                symbolName={stock.symbolName}
                size="lg"
              />
              <div className="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
                {stock.market && (
                  <span className="rounded bg-muted px-2 py-0.5">
                    {stock.market}
                  </span>
                )}
                {stock.sector && <span>{stock.sector}</span>}
              </div>
            </div>
          </SheetTitle>
        </SheetHeader>

        <Tabs value={activeTab} onValueChange={(v: string) => setActiveTab(v as StockDetailTab)} className="mt-4">
          <TabsList className="grid w-full grid-cols-3">
            {tabs.map((tab) => (
              <TabsTrigger key={tab.key} value={tab.key} className="gap-2">
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>

          {/* 종목명 + 현재가 (모든 탭 공통) */}
          <div className="mt-4 space-y-3 px-4">
            {/* 종목명 */}
            <div className="text-lg font-semibold text-muted-foreground">
              {stock.symbolName}
            </div>

            {/* 현재가 */}
            {priceInfo && (
              <div className="space-y-2">
                <div
                  className="text-4xl font-bold"
                  style={{
                    color: (priceInfo.changeRate || 0) >= 0 ? '#EA5455' : '#2196F3'
                  }}
                >
                  {Math.floor(priceInfo.currentPrice || 0).toLocaleString()}원
                </div>
                <div className="flex items-center gap-4">
                  <span
                    className="text-xl font-semibold"
                    style={{
                      color: (priceInfo.changeRate || 0) >= 0 ? '#EA5455' : '#2196F3'
                    }}
                  >
                    {(priceInfo.changeRate || 0) >= 0 ? '+' : ''}
                    {Math.floor(priceInfo.changePrice || 0).toLocaleString()}
                  </span>
                  <span
                    className="text-xl font-semibold"
                    style={{
                      color: (priceInfo.changeRate || 0) >= 0 ? '#EA5455' : '#2196F3'
                    }}
                  >
                    {(priceInfo.changeRate || 0) >= 0 ? '+' : ''}
                    {(priceInfo.changeRate || 0).toFixed(2)}%
                  </span>
                </div>
              </div>
            )}
          </div>

          <TabsContent value="holding" className="mt-4">
            <HoldingTab
              symbol={stock.symbol}
              symbolName={stock.symbolName}
              holding={holding}
              totalEvaluation={totalEvaluation}
              onExitModeToggle={(enabled) => holding && onExitModeToggle(holding, enabled)}
            />
          </TabsContent>

          <TabsContent value="price" className="mt-4">
            <PriceTab
              symbol={stock.symbol}
              symbolName={stock.symbolName}
              priceInfo={priceInfo}
            />
          </TabsContent>

          <TabsContent value="order" className="mt-4">
            <OrderTab
              symbol={stock.symbol}
              symbolName={stock.symbolName}
              currentPrice={priceInfo?.currentPrice || 0}
              holding={holding}
              unfilledOrders={unfilledOrders}
              executedOrders={executedOrders}
            />
          </TabsContent>

          {/* Phase 2: 추가 탭들 */}
          {/* <TabsContent value="investment">...</TabsContent> */}
          {/* <TabsContent value="consensus">...</TabsContent> */}
          {/* <TabsContent value="ai">...</TabsContent> */}
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}
