'use client'

import { useState } from 'react'
import { X, TrendingUp, ShoppingCart, DollarSign, Target, Brain } from 'lucide-react'
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
import { PriceTab } from './tabs/price-tab'
import { OrderTab } from './tabs/order-tab'

interface StockDetailSheetProps {
  stock: StockInfo | null
  open: boolean
  onOpenChange: (open: boolean) => void
  holdings: any[]           // Holdings 데이터
  unfilledOrders: any[]     // 미체결 주문
  executedOrders: any[]     // 체결 주문
}

/**
 * 종목 상세 Sheet (v10 스타일)
 *
 * Phase 1: Price, Order 탭
 * Phase 2: Investment, Consensus, AI 탭 추가
 */
export function StockDetailSheet({
  stock,
  open,
  onOpenChange,
  holdings,
  unfilledOrders,
  executedOrders,
}: StockDetailSheetProps) {
  const [activeTab, setActiveTab] = useState<StockDetailTab>('price')

  // 가격 정보 조회
  const { data: priceInfo } = useStockPrice(stock?.symbol || '', holdings)

  if (!stock) return null

  const tabs = [
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
          <div className="flex items-start justify-between">
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
          </div>
        </SheetHeader>

        <Tabs value={activeTab} onValueChange={(v: string) => setActiveTab(v as StockDetailTab)} className="mt-4">
          <TabsList className="grid w-full grid-cols-2">
            {tabs.map((tab) => (
              <TabsTrigger key={tab.key} value={tab.key} className="gap-2">
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>

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
