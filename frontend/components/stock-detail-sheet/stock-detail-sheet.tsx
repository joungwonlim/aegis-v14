'use client'

import { useState } from 'react'
import { X, TrendingUp, TrendingDown, ShoppingCart, DollarSign, Target, Brain, Package, Settings, Star, Bell, BarChart3, ExternalLink, LogOut } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { StockSymbol } from '@/components/stock-symbol'
import type { StockInfo, StockDetailTab } from './types'
import { useStockPrice } from './hooks/use-stock-price'
import { HoldingTab } from './tabs/holding-tab'
import { PriceTab } from './tabs/price-tab'
import { OrderTab } from './tabs/order-tab'
import { ExitTab } from './tabs/exit-tab'

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
  const [exitRuleDialogOpen, setExitRuleDialogOpen] = useState(false)

  // 가격 정보 조회
  const { data: priceInfo } = useStockPrice(stock?.symbol || '', holdings)

  // 해당 종목의 보유 정보 찾기
  const holding = holdings.find((h) => h.symbol === stock?.symbol)

  if (!stock) return null

  const tabs = [
    { key: 'holding', label: '보유', icon: Package },
    { key: 'price', label: '시세', icon: TrendingUp },
    { key: 'order', label: '주문', icon: ShoppingCart },
    { key: 'exit', label: 'Exit', icon: LogOut },
    // Phase 2에서 활성화
    // { key: 'investment', label: '투자지표', icon: DollarSign },
    // { key: 'consensus', label: '컨센서스', icon: Target },
    // { key: 'ai', label: 'AI분석', icon: Brain },
  ] as const

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-4xl overflow-y-auto px-6 sm:px-8">
        <SheetHeader className="sticky top-0 z-10 border-b pb-4 bg-background -mx-6 sm:-mx-8 px-6 sm:px-8">
          <SheetTitle className="sr-only">{stock.symbolName || stock.symbol} 종목 상세</SheetTitle>
          <div className="flex items-start gap-3 justify-between">

            <div className="flex items-start gap-3">
              {/* Stock Logo */}
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 overflow-hidden">
                <img
                  src={`https://ssl.pstatic.net/imgstock/fn/real/logo/stock/Stock${stock.symbol}.svg`}
                  alt={stock.symbolName || stock.symbol}
                  className="w-full h-full object-cover"
                  onError={(e) => {
                    // Fallback to first letter if image fails to load
                    e.currentTarget.style.display = 'none'
                    const parent = e.currentTarget.parentElement
                    if (parent) {
                      const fallback = document.createElement('span')
                      fallback.className = 'text-lg font-bold text-primary'
                      fallback.textContent = (stock.symbolName || stock.symbol).charAt(0)
                      parent.appendChild(fallback)
                    }
                  }}
                />
              </div>

              {/* 종목정보 + 가격 */}
              <div>
                <div className="font-semibold text-lg">{stock.symbolName || stock.symbol}</div>
                <div className="text-sm text-muted-foreground">{stock.symbol}</div>
                {/* 현재가 + 전일대비 */}
                {priceInfo ? (
                  <div className="flex items-baseline gap-2 mt-1">
                    <span className="text-xl font-bold">
                      {Math.floor(priceInfo.currentPrice || 0).toLocaleString()}원
                    </span>
                    <span
                      className="text-sm font-medium"
                      style={{
                        color: (priceInfo.changeRate || 0) >= 0 ? '#EA5455' : '#2196F3'
                      }}
                    >
                      {(priceInfo.changeRate || 0) >= 0 ? '▲' : '▼'}
                      {Math.floor(Math.abs(priceInfo.changePrice || 0)).toLocaleString()}
                      {' '}
                      ({(priceInfo.changeRate || 0) >= 0 ? '+' : ''}{(priceInfo.changeRate || 0).toFixed(2)}%)
                    </span>
                  </div>
                ) : (
                  <div className="text-sm text-muted-foreground mt-1">가격 정보 로딩 중...</div>
                )}
              </div>
            </div>

            {/* Exit Engine 스위치 (오른쪽 정렬) */}
            {holding && (
              <div className="flex items-center gap-2">
                <Label htmlFor="exit-engine-header" className="text-sm font-medium">
                  Exit Engine
                </Label>
                <Switch
                  id="exit-engine-header"
                  checked={holding.exit_mode === 'ENABLED'}
                  onCheckedChange={(enabled) => {
                    onExitModeToggle(holding, enabled)
                  }}
                />
                <span className="text-xs text-muted-foreground">
                  {holding.exit_mode === 'ENABLED' ? '활성화됨' : '비활성화됨'}
                </span>
              </div>
            )}
          </div>

          {/* 보유 정보 요약 */}
          {holding && (() => {
            const evalAmount = parseInt(holding.raw?.evaluate_amount || '0')
            const purchaseAmount = parseInt(holding.raw?.purchase_amount || '0')
            const pnl = holding.pnl || 0
            const pnlPct = holding.pnl_pct || 0
            const avgPrice = Math.floor(holding.avg_price || 0)
            const qty = holding.qty || 0
            const weight = totalEvaluation > 0 ? (evalAmount / totalEvaluation) * 100 : 0

            return (
              <div className="mt-4 grid grid-cols-4 gap-x-4 gap-y-2 text-sm border rounded-lg bg-muted/30 p-4">
                <div>
                  <div className="text-muted-foreground text-xs">보유수량</div>
                  <div className="font-medium">{qty.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">매도가능</div>
                  <div className="font-medium">{qty.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">평가손익</div>
                  <div className={`font-medium ${pnl >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                    {pnl >= 0 ? '+' : ''}{Math.floor(pnl).toLocaleString()}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">수익률</div>
                  <div className={`font-medium ${pnlPct >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                    {pnlPct >= 0 ? '+' : ''}{pnlPct.toFixed(2)}%
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">매입단가</div>
                  <div className="font-medium font-mono">{avgPrice.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">평가금액</div>
                  <div className="font-medium font-mono">{evalAmount.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">매입금액</div>
                  <div className="font-medium font-mono">{purchaseAmount.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">비중</div>
                  <div className="font-medium">{weight.toFixed(1)}%</div>
                </div>
              </div>
            )
          })()}
        </SheetHeader>

        <Tabs value={activeTab} onValueChange={(v: string) => setActiveTab(v as StockDetailTab)} className="mt-4">
          <TabsList className="grid w-full grid-cols-4">
            {tabs.map((tab) => (
              <TabsTrigger key={tab.key} value={tab.key} className="gap-2">
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>

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

          <TabsContent value="exit" className="mt-4">
            <ExitTab
              symbol={stock.symbol}
              symbolName={stock.symbolName}
              holding={holding}
              onExitModeToggle={(enabled) => holding && onExitModeToggle(holding, enabled)}
            />
          </TabsContent>

          {/* Phase 2: 추가 탭들 */}
          {/* <TabsContent value="investment">...</TabsContent> */}
          {/* <TabsContent value="consensus">...</TabsContent> */}
          {/* <TabsContent value="ai">...</TabsContent> */}
        </Tabs>
      </SheetContent>

      {/* Exit Rule 다이얼로그 */}
      <Dialog open={exitRuleDialogOpen} onOpenChange={setExitRuleDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Exit Rule 설정</DialogTitle>
            <DialogDescription>
              {stock.symbolName} ({stock.symbol})의 Exit Engine 설정
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6 py-4">
            {/* Exit Engine 활성화/비활성화 */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="exit-engine-dialog" className="text-base font-semibold">
                  Exit Engine
                </Label>
                <div className="text-sm text-muted-foreground">
                  자동 손절/익절 시스템 활성화
                </div>
              </div>
              <Switch
                id="exit-engine-dialog"
                checked={holding?.exit_mode === 'ENABLED'}
                onCheckedChange={(enabled) => {
                  if (holding) {
                    onExitModeToggle(holding, enabled)
                  }
                }}
              />
            </div>

            {/* 추가 설정 (Phase 2) */}
            {holding?.exit_mode === 'ENABLED' && (
              <div className="rounded-lg border bg-muted/50 p-4">
                <div className="text-sm text-muted-foreground">
                  🚧 상세 Exit Rule 설정은 Phase 2에서 추가됩니다
                  <br />
                  (손절률, 익절률, Trailing Stop 등)
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </Sheet>
  )
}
