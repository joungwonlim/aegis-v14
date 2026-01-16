'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { StockSymbol } from '@/components/stock-symbol'
import { StockDetailSheet, useStockDetail, type StockInfo } from '@/components/stock-detail-sheet'
import { ChangeIndicator } from '@/components/ui/change-indicator'
import { approveIntent, rejectIntent, updateExitMode, cancelKISOrder, type Holding, type OrderIntent, type Order, type Fill, type KISUnfilledOrder, type KISFill } from '@/lib/api'
import { useHoldings, useOrderIntents, useOrders, useFills, useKISUnfilledOrders, useKISFilledOrders } from '@/hooks/useRuntimeData'
import { toast } from 'sonner'

type SortField = 'symbol' | 'qty' | 'pnl' | 'pnl_pct' | 'avg_price' | 'current_price' | 'eval_amount' | 'purchase_amount' | 'weight'
type IntentSortField = 'symbol' | 'current_price' | 'order_price' | 'deviation' | 'qty' | 'created_ts'
type SortOrder = 'asc' | 'desc'

export default function RuntimeDashboard() {
  // React Query 훅으로 데이터 조회 (1초마다 자동 갱신)
  const { data: holdings = [], isLoading: holdingsLoading, error: holdingsError, refetch: refetchHoldings } = useHoldings()
  const { data: intents = [], isLoading: intentsLoading, refetch: refetchIntents } = useOrderIntents()
  const { data: orders = [], isLoading: ordersLoading } = useOrders()
  const { data: fills = [], isLoading: fillsLoading } = useFills()
  const { data: kisUnfilledOrders = [], isLoading: kisUnfilledLoading, refetch: refetchKISUnfilledOrders } = useKISUnfilledOrders()
  const { data: kisFilledOrders = [], isLoading: kisFilledLoading } = useKISFilledOrders()

  const loading = holdingsLoading || intentsLoading || ordersLoading || fillsLoading || kisUnfilledLoading || kisFilledLoading
  const error = holdingsError ? (holdingsError as Error).message : null
  const [rulesDialogOpen, setRulesDialogOpen] = useState(false)
  const [sortField, setSortField] = useState<SortField>('eval_amount') // 기본 정렬: 평가금액
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc') // 내림차순 (높은 순)
  const [intentSortField, setIntentSortField] = useState<IntentSortField | null>(null)
  const [intentSortOrder, setIntentSortOrder] = useState<SortOrder>('desc')

  // StockDetailSheet 훅
  const { selectedStock, isOpen: isStockDetailOpen, openStockDetail, handleOpenChange: handleStockDetailOpenChange } = useStockDetail()

  // 총 평가금액 계산 (비중 계산용)
  const totalEvaluation = holdings.reduce((sum, h) => {
    const evalAmount = parseInt(h.raw?.evaluate_amount || '0')
    return sum + evalAmount
  }, 0)

  // 정렬 핸들러
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      // 같은 필드 클릭 시 정렬 순서 변경
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      // 다른 필드 클릭 시 해당 필드로 내림차순 정렬
      setSortField(field)
      setSortOrder('desc')
    }
  }

  // 정렬된 holdings
  const sortedHoldings = [...holdings].sort((a, b) => {
    if (!sortField) return 0

    let aValue: number | string = 0
    let bValue: number | string = 0

    const aEvalAmount = parseInt(a.raw?.evaluate_amount || '0')
    const aPurchaseAmount = parseInt(a.raw?.purchase_amount || '0')
    const aWeight = totalEvaluation > 0 ? (aEvalAmount / totalEvaluation) * 100 : 0

    const bEvalAmount = parseInt(b.raw?.evaluate_amount || '0')
    const bPurchaseAmount = parseInt(b.raw?.purchase_amount || '0')
    const bWeight = totalEvaluation > 0 ? (bEvalAmount / totalEvaluation) * 100 : 0

    switch (sortField) {
      case 'symbol':
        aValue = a.raw?.symbol_name || a.symbol
        bValue = b.raw?.symbol_name || b.symbol
        break
      case 'qty':
        aValue = a.qty
        bValue = b.qty
        break
      case 'pnl':
        aValue = typeof a.pnl === 'string' ? parseFloat(a.pnl) : a.pnl
        bValue = typeof b.pnl === 'string' ? parseFloat(b.pnl) : b.pnl
        break
      case 'pnl_pct':
        aValue = a.pnl_pct
        bValue = b.pnl_pct
        break
      case 'avg_price':
        aValue = typeof a.avg_price === 'string' ? parseFloat(a.avg_price) : a.avg_price
        bValue = typeof b.avg_price === 'string' ? parseFloat(b.avg_price) : b.avg_price
        break
      case 'current_price':
        aValue = typeof a.current_price === 'string' ? parseFloat(a.current_price) : a.current_price
        bValue = typeof b.current_price === 'string' ? parseFloat(b.current_price) : b.current_price
        break
      case 'eval_amount':
        aValue = aEvalAmount
        bValue = bEvalAmount
        break
      case 'purchase_amount':
        aValue = aPurchaseAmount
        bValue = bPurchaseAmount
        break
      case 'weight':
        aValue = aWeight
        bValue = bWeight
        break
    }

    if (typeof aValue === 'string' && typeof bValue === 'string') {
      return sortOrder === 'asc' ? aValue.localeCompare(bValue) : bValue.localeCompare(aValue)
    }

    return sortOrder === 'asc' ? (aValue as number) - (bValue as number) : (bValue as number) - (aValue as number)
  })

  // Intent 정렬 핸들러
  const handleIntentSort = (field: IntentSortField) => {
    if (intentSortField === field) {
      setIntentSortOrder(intentSortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setIntentSortField(field)
      setIntentSortOrder('desc')
    }
  }

  // 정렬된 intents
  const sortedIntents = [...intents].sort((a, b) => {
    if (!intentSortField) return 0

    const aHolding = holdings.find(h => h.symbol === a.symbol)
    const bHolding = holdings.find(h => h.symbol === b.symbol)
    const aCurrentPrice = typeof aHolding?.current_price === 'string'
      ? parseFloat(aHolding.current_price)
      : (aHolding?.current_price || 0)
    const bCurrentPrice = typeof bHolding?.current_price === 'string'
      ? parseFloat(bHolding.current_price)
      : (bHolding?.current_price || 0)
    const aOrderPrice = a.limit_price || aCurrentPrice
    const bOrderPrice = b.limit_price || bCurrentPrice
    const aDeviation = aOrderPrice > 0 ? ((aCurrentPrice - aOrderPrice) / aOrderPrice) * 100 : 0
    const bDeviation = bOrderPrice > 0 ? ((bCurrentPrice - bOrderPrice) / bOrderPrice) * 100 : 0

    let aValue: number | string = 0
    let bValue: number | string = 0

    switch (intentSortField) {
      case 'symbol':
        aValue = a.symbol_name || a.symbol
        bValue = b.symbol_name || b.symbol
        break
      case 'current_price':
        aValue = aCurrentPrice
        bValue = bCurrentPrice
        break
      case 'order_price':
        aValue = aOrderPrice
        bValue = bOrderPrice
        break
      case 'deviation':
        aValue = aDeviation
        bValue = bDeviation
        break
      case 'qty':
        aValue = a.qty
        bValue = b.qty
        break
      case 'created_ts':
        aValue = new Date(a.created_ts).getTime()
        bValue = new Date(b.created_ts).getTime()
        break
    }

    if (typeof aValue === 'string' && typeof bValue === 'string') {
      return intentSortOrder === 'asc' ? aValue.localeCompare(bValue) : bValue.localeCompare(aValue)
    }

    return intentSortOrder === 'asc' ? (aValue as number) - (bValue as number) : (bValue as number) - (aValue as number)
  })

  // 합계 계산
  const totals = holdings.reduce((acc, h) => {
    const pnl = typeof h.pnl === 'string' ? parseFloat(h.pnl) : h.pnl
    const evalAmount = parseInt(h.raw?.evaluate_amount || '0')
    const purchaseAmount = parseInt(h.raw?.purchase_amount || '0')

    return {
      qty: acc.qty + h.qty,
      pnl: acc.pnl + pnl,
      evalAmount: acc.evalAmount + evalAmount,
      purchaseAmount: acc.purchaseAmount + purchaseAmount,
    }
  }, { qty: 0, pnl: 0, evalAmount: 0, purchaseAmount: 0 })

  const totalPnlPct = totals.purchaseAmount > 0 ? (totals.pnl / totals.purchaseAmount) * 100 : 0

  const handleApprove = async (intentId: string) => {
    try {
      const result = await approveIntent(intentId)
      await refetchIntents() // Refresh intents after approval

      // 주문 승인 성공 toast
      toast.success('주문 승인 완료', {
        description: '주문이 실행 대기열에 추가되었습니다.',
        duration: 10000,
        style: {
          background: '#10b981',
          color: '#ffffff',
          border: '1px solid #059669',
        },
      })
    } catch (err) {
      console.error('Failed to approve intent:', err)

      // 에러 메시지 파싱
      const errorMessage = err instanceof Error ? err.message : '알 수 없는 오류'

      // 장중 체크 에러인지 확인
      if (errorMessage.includes('market') || errorMessage.includes('시간') || errorMessage.includes('장중')) {
        toast.error('주문 실행 실패', {
          description: '장중이 아니라서 주문을 실행할 수 없습니다.',
          duration: 10000,
          style: {
            background: '#ef4444',
            color: '#ffffff',
            border: '1px solid #dc2626',
          },
        })
      } else {
        toast.error('주문 실행 실패', {
          description: errorMessage,
          duration: 10000,
          style: {
            background: '#ef4444',
            color: '#ffffff',
            border: '1px solid #dc2626',
          },
        })
      }
    }
  }

  const handleReject = async (intentId: string) => {
    try {
      await rejectIntent(intentId)
      await refetchIntents() // Refresh intents after rejection

      // 주문 취소 성공 toast
      toast.info('주문 취소됨', {
        description: 'Exit Intent가 취소되었습니다.',
        duration: 10000,
        style: {
          background: '#3b82f6',
          color: '#ffffff',
          border: '1px solid #2563eb',
        },
      })
    } catch (err) {
      console.error('Failed to reject intent:', err)

      const errorMessage = err instanceof Error ? err.message : '알 수 없는 오류'
      toast.error('주문 취소 실패', {
        description: errorMessage,
        duration: 10000,
        style: {
          background: '#ef4444',
          color: '#ffffff',
          border: '1px solid #dc2626',
        },
      })
    }
  }

  const handleCancelOrder = async (orderNo: string, stockName?: string) => {
    const displayName = stockName || orderNo
    if (!confirm(`${displayName} 주문을 취소하시겠습니까?`)) {
      return
    }

    try {
      const result = await cancelKISOrder(orderNo)
      if (result.success) {
        console.log(`Order ${orderNo} cancelled successfully. Cancel No: ${result.cancel_no}`)
        await refetchKISUnfilledOrders() // Refresh unfilled orders after cancellation
      } else {
        console.error(`Failed to cancel order ${orderNo}:`, result.error)
        alert(`주문 취소 실패: ${result.error}`)
      }
    } catch (err) {
      console.error('Failed to cancel order:', err)
      alert(`주문 취소 중 오류 발생: ${err instanceof Error ? err.message : '알 수 없는 오류'}`)
    }
  }

  const handleHoldingClick = (holding: Holding) => {
    // StockDetailSheet 열기
    openStockDetail({
      symbol: holding.symbol,
      symbolName: holding.raw?.symbol_name || holding.symbol,
    })
  }

  const handleExitModeToggle = async (holding: Holding, enabled: boolean) => {
    try {
      const exitMode = enabled ? 'ENABLED' : 'DISABLED'
      console.log('Updating exit mode:', { account_id: holding.account_id, symbol: holding.symbol, exitMode })

      const result = await updateExitMode(holding.account_id, holding.symbol, exitMode)
      console.log('Update result:', result)

      // Refresh holdings after update
      await refetchHoldings()
    } catch (err) {
      console.error('Failed to update exit mode:', err)
      // Revert by refetching
      await refetchHoldings()
    }
  }

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
      PENDING_APPROVAL: 'outline',
      NEW: 'secondary',
      ACK: 'default',
      SUBMITTED: 'default',
      FILLED: 'default',
      PARTIAL: 'secondary',
      FAILED: 'destructive',
      REJECTED: 'destructive',
      CANCELLED: 'outline',
      DUPLICATE: 'outline',
    }

    // Status 한글 변환
    const statusLabels: Record<string, string> = {
      PENDING_APPROVAL: '승인대기',
      NEW: '주문대기',
      SUBMITTED: '주문완료',
      ACK: '처리중',
      FILLED: '체결완료',
      PARTIAL: '부분체결',
      FAILED: '실패',
      REJECTED: '거부',
      CANCELLED: '취소',
      DUPLICATE: '중복',
    }

    return <Badge variant={variants[status] || 'default'}>{statusLabels[status] || status}</Badge>
  }

  const formatNumber = (value: number | undefined, decimals = 0) => {
    return value?.toLocaleString('ko-KR', { minimumFractionDigits: decimals, maximumFractionDigits: decimals }) ?? '-'
  }

  const formatPercent = (value: number | undefined) => {
    if (value === undefined) return '-'
    const formatted = value.toFixed(2)
    const color = value >= 0 ? '#EA5455' : '#2196F3'
    const sign = value > 0 ? '+' : ''
    return <span style={{ color }}>{sign}{formatted}%</span>
  }

  const formatPnL = (value: number | undefined) => {
    if (value === undefined) return '-'
    const color = value >= 0 ? '#EA5455' : '#2196F3'
    const sign = value > 0 ? '+' : ''
    return <span style={{ color }}>{sign}{formatNumber(value, 0)}</span>
  }

  const formatTimestamp = (ts: string) => {
    // DB timestamp는 KST이지만 timezone 정보가 없어서 UTC로 해석됨
    // +09:00을 추가하여 KST로 명시
    const kstTimestamp = ts.includes('+') || ts.includes('Z') ? ts : `${ts}+09:00`
    return new Date(kstTimestamp).toLocaleString('ko-KR', {
      timeZone: 'Asia/Seoul',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Aegis v14 Runtime Monitor</h1>
          <p className="text-muted-foreground">실시간 트레이딩 엔진 모니터링 (1초 자동 갱신)</p>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-sm text-muted-foreground">
            {loading ? '갱신 중...' : '✅ 실시간 연결'}
          </span>
          <Button
            onClick={() => {
              refetchHoldings()
              refetchIntents()
            }}
            disabled={loading}
          >
            수동 새로고침
          </Button>
        </div>
      </div>

      {error && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Error</CardTitle>
          </CardHeader>
          <CardContent>
            <p>{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Portfolio - PriceSync */}
      <Card>
        <CardHeader>
          <CardTitle>📊 Portfolio (PriceSync 되어야함)</CardTitle>
          <CardDescription>
            현재 보유 포지션 및 실시간 가격 동기화 ({holdings.length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('symbol')}
                >
                  <div className="flex items-center gap-1">
                    종목명
                    {sortField === 'symbol' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('current_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    현재가
                    {sortField === 'current_price' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('pnl_pct')}
                >
                  <div className="flex items-center justify-end gap-1">
                    전일대비
                    {sortField === 'pnl_pct' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('qty')}
                >
                  <div className="flex items-center justify-end gap-1">
                    보유수량
                    {sortField === 'qty' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-right">매도가능</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('pnl')}
                >
                  <div className="flex items-center justify-end gap-1">
                    평가손익
                    {sortField === 'pnl' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('pnl_pct')}
                >
                  <div className="flex items-center justify-end gap-1">
                    수익률
                    {sortField === 'pnl_pct' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('avg_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    매입단가
                    {sortField === 'avg_price' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('purchase_amount')}
                >
                  <div className="flex items-center justify-end gap-1">
                    매입금액
                    {sortField === 'purchase_amount' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('eval_amount')}
                >
                  <div className="flex items-center justify-end gap-1">
                    평가금액
                    {sortField === 'eval_amount' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleSort('weight')}
                >
                  <div className="flex items-center justify-end gap-1">
                    비중
                    {sortField === 'weight' && (
                      <span className="text-xs">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {holdings.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={11} className="text-center text-muted-foreground">
                    보유종목이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                <>
                  {sortedHoldings.map((holding) => {
                    const symbolName = holding.raw?.symbol_name || holding.symbol
                    const evaluateAmount = holding.raw?.evaluate_amount || (holding.qty * holding.current_price).toString()
                    const purchaseAmount = holding.raw?.purchase_amount || (holding.qty * holding.avg_price).toString()
                    const weight = totalEvaluation > 0 ? (parseInt(evaluateAmount) / totalEvaluation) * 100 : 0

                    // 문자열을 숫자로 변환
                    const pnl = typeof holding.pnl === 'string' ? parseFloat(holding.pnl) : holding.pnl
                    const currentPrice = typeof holding.current_price === 'string' ? parseFloat(holding.current_price) : holding.current_price
                    const avgPrice = typeof holding.avg_price === 'string' ? parseFloat(holding.avg_price) : holding.avg_price

                    return (
                      <TableRow key={`${holding.account_id}-${holding.symbol}`}>
                        <TableCell
                          className="cursor-pointer hover:opacity-80"
                          onClick={() => handleHoldingClick(holding)}
                        >
                          <StockSymbol
                            symbol={holding.symbol}
                            symbolName={symbolName}
                            size="sm"
                            isHolding={true}
                            isExitEnabled={holding.exit_mode === 'ENABLED'}
                            market={holding.raw?.market}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">
                          <ChangeIndicator
                            changePrice={holding.change_price}
                            changeRate={holding.change_rate}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(holding.qty)}</TableCell>
                        <TableCell className="text-right font-mono text-muted-foreground">{formatNumber(holding.qty)}</TableCell>
                        <TableCell className="text-right font-mono">{formatPnL(pnl)}</TableCell>
                        <TableCell className="text-right font-mono">{formatPercent(holding.pnl_pct)}</TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(avgPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(parseInt(purchaseAmount), 0)}</TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(parseInt(evaluateAmount), 0)}</TableCell>
                        <TableCell className="text-right font-mono text-muted-foreground">{weight.toFixed(1)}%</TableCell>
                      </TableRow>
                    )
                  })}
                  {/* 합계 행 */}
                  <TableRow className="font-semibold bg-muted/30">
                    <TableCell className="font-bold">합계</TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">-</TableCell>
                    <TableCell className="text-right font-mono">{formatPercent(totalPnlPct)}</TableCell>
                    <TableCell className="text-right font-mono">{formatNumber(totals.qty)}</TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">{formatNumber(totals.qty)}</TableCell>
                    <TableCell className="text-right font-mono">{formatPnL(totals.pnl)}</TableCell>
                    <TableCell className="text-right font-mono">{formatPercent(totalPnlPct)}</TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">-</TableCell>
                    <TableCell className="text-right font-mono">{formatNumber(totals.purchaseAmount, 0)}</TableCell>
                    <TableCell className="text-right font-mono">{formatNumber(totals.evalAmount, 0)}</TableCell>
                    <TableCell className="text-right font-mono">100.0%</TableCell>
                  </TableRow>
                </>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Exit Engine - 청산 대상 종목 모니터링 */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-start">
            <div className="space-y-1.5">
              <CardTitle>🎯 Exit Engine - 청산 대상 종목 모니터링</CardTitle>
              <CardDescription>
                Exit 규칙 평가 및 청산 주문 의도 ({intents.filter(i => holdings.some(h => h.symbol === i.symbol && h.qty > 0)).length}개)
              </CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={() => setRulesDialogOpen(true)}>
              규칙 관리
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('symbol')}
                >
                  <div className="flex items-center gap-1">
                    종목명
                    {intentSortField === 'symbol' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('current_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    현재가
                    {intentSortField === 'current_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="text-right">매입단가</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('order_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    주문가격
                    {intentSortField === 'order_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('deviation')}
                >
                  <div className="flex items-center justify-end gap-1">
                    괴리율
                    {intentSortField === 'deviation' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>타입</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('qty')}
                >
                  <div className="flex items-center justify-end gap-1">
                    수량
                    {intentSortField === 'qty' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>주문유형</TableHead>
                <TableHead>사유</TableHead>
                <TableHead>상태</TableHead>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('created_ts')}
                >
                  <div className="flex items-center gap-1">
                    생성시각
                    {intentSortField === 'created_ts' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(() => {
                // 보유종목이 있는 intent만 표시 (매도 완료된 종목 제외)
                const activeIntents = sortedIntents.filter(intent => {
                  const holding = holdings.find(h => h.symbol === intent.symbol)
                  return holding && holding.qty > 0
                })

                if (activeIntents.length === 0) {
                  return (
                    <TableRow>
                      <TableCell colSpan={11} className="text-center text-muted-foreground">
                        Order Intent가 없습니다
                      </TableCell>
                    </TableRow>
                  )
                }

                return activeIntents.map((intent) => {
                  // holdings에서 현재가 정보 가져오기
                  const holding = holdings.find(h => h.symbol === intent.symbol)
                  const currentPrice = typeof holding?.current_price === 'string'
                    ? parseFloat(holding.current_price)
                    : (holding?.current_price || 0)
                  const pnlPct = holding?.pnl_pct || 0
                  const avgPrice = holding
                    ? (typeof holding.avg_price === 'string' ? parseFloat(holding.avg_price) : holding.avg_price)
                    : 0

                  // 주문가격 (limit_price 또는 현재가)
                  const orderPrice = intent.limit_price || currentPrice

                  // 괴리율 계산: (현재가 - 주문가격) / 주문가격 * 100
                  const deviationPct = orderPrice > 0 ? ((currentPrice - orderPrice) / orderPrice) * 100 : 0

                  return (
                    <TableRow key={intent.intent_id}>
                      <TableCell>
                        <StockSymbol
                          symbol={intent.symbol}
                          symbolName={intent.symbol_name}
                          size="sm"
                          isHolding={!!holding}
                          isExitEnabled={holding?.exit_mode === 'ENABLED'}
                          market={holding?.raw?.market}
                        />
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">
                        <ChangeIndicator
                          changePrice={holding?.change_price}
                          changeRate={holding?.change_rate}
                        />
                      </TableCell>
                      <TableCell className="text-right font-mono text-muted-foreground">
                        {holding ? formatNumber(avgPrice, 0) : '-'}
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(orderPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">{formatPercent(deviationPct)}</TableCell>
                      <TableCell>{intent.intent_type}</TableCell>
                      <TableCell className="text-right">{formatNumber(intent.qty)}</TableCell>
                      <TableCell>{intent.order_type}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{intent.reason_code}</Badge>
                      </TableCell>
                      <TableCell>{getStatusBadge(intent.status)}</TableCell>
                      <TableCell className="text-sm">{formatTimestamp(intent.created_ts)}</TableCell>
                    </TableRow>
                  )
                })
              })()}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* KIS 주문 대기 - PENDING_APPROVAL */}
      <Card>
        <CardHeader>
          <CardTitle>🕐 KIS 주문 대기</CardTitle>
          <CardDescription>
            수동 승인 필요 ({sortedIntents.filter(i => i.status === 'PENDING_APPROVAL' && holdings.some(h => h.symbol === i.symbol && h.qty > 0)).length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('symbol')}
                >
                  <div className="flex items-center gap-1">
                    종목명
                    {intentSortField === 'symbol' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('current_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    현재가
                    {intentSortField === 'current_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="text-right">매입단가</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('order_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    주문가격
                    {intentSortField === 'order_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('deviation')}
                >
                  <div className="flex items-center justify-end gap-1">
                    괴리율
                    {intentSortField === 'deviation' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>타입</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('qty')}
                >
                  <div className="flex items-center justify-end gap-1">
                    수량
                    {intentSortField === 'qty' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>주문유형</TableHead>
                <TableHead>사유</TableHead>
                <TableHead>상태</TableHead>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('created_ts')}
                >
                  <div className="flex items-center gap-1">
                    생성시각
                    {intentSortField === 'created_ts' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-center">작업</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(() => {
                // PENDING_APPROVAL 상태의 intent만 표시
                const pendingApprovalIntents = sortedIntents.filter(i =>
                  i.status === 'PENDING_APPROVAL' &&
                  holdings.some(h => h.symbol === i.symbol && h.qty > 0)
                )

                if (pendingApprovalIntents.length === 0) {
                  return (
                    <TableRow>
                      <TableCell colSpan={12} className="text-center text-muted-foreground">
                        승인 대기 중인 Intent가 없습니다
                      </TableCell>
                    </TableRow>
                  )
                }

                return pendingApprovalIntents.map((intent) => {
                    // holdings에서 현재가 정보 가져오기
                    const holding = holdings.find(h => h.symbol === intent.symbol)
                    const currentPrice = typeof holding?.current_price === 'string'
                      ? parseFloat(holding.current_price)
                      : (holding?.current_price || 0)
                    const pnlPct = holding?.pnl_pct || 0
                    const avgPrice = holding
                      ? (typeof holding.avg_price === 'string' ? parseFloat(holding.avg_price) : holding.avg_price)
                      : 0

                    // 주문가격 (limit_price 또는 현재가)
                    const orderPrice = intent.limit_price || currentPrice

                    // 괴리율 계산: (현재가 - 주문가격) / 주문가격 * 100
                    const deviationPct = orderPrice > 0 ? ((currentPrice - orderPrice) / orderPrice) * 100 : 0

                    return (
                      <TableRow key={intent.intent_id}>
                        <TableCell>
                          <StockSymbol
                            symbol={intent.symbol}
                            symbolName={intent.symbol_name}
                            size="sm"
                            isHolding={!!holding}
                            isExitEnabled={holding?.exit_mode === 'ENABLED'}
                            market={holding?.raw?.market}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">
                          <ChangeIndicator
                            changePrice={holding?.change_price}
                            changeRate={holding?.change_rate}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono text-muted-foreground">
                          {holding ? formatNumber(avgPrice, 0) : '-'}
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(orderPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">{formatPercent(deviationPct)}</TableCell>
                        <TableCell>{intent.intent_type}</TableCell>
                        <TableCell className="text-right">{formatNumber(intent.qty)}</TableCell>
                        <TableCell>{intent.order_type}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{intent.reason_code}</Badge>
                        </TableCell>
                        <TableCell>{getStatusBadge(intent.status)}</TableCell>
                        <TableCell className="text-sm">{formatTimestamp(intent.created_ts)}</TableCell>
                        <TableCell className="text-center">
                          <div className="flex gap-2 justify-center">
                            <Button
                              size="sm"
                              onClick={() => handleApprove(intent.intent_id)}
                              className="bg-green-600 hover:bg-green-700"
                            >
                              주문 실행
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleReject(intent.intent_id)}
                            >
                              삭제
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
              })()}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* KIS Orders Execution */}
      <Card>
        <CardHeader>
          <CardTitle>📤 KIS Orders Execution</CardTitle>
          <CardDescription>
            승인 완료 Exit Intent ({sortedIntents.filter(i => (i.status === 'NEW' || i.status === 'SUBMITTED') && holdings.some(h => h.symbol === i.symbol && h.qty > 0)).length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('symbol')}
                >
                  <div className="flex items-center gap-1">
                    종목명
                    {intentSortField === 'symbol' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('current_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    현재가
                    {intentSortField === 'current_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="text-right">매입단가</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('order_price')}
                >
                  <div className="flex items-center justify-end gap-1">
                    주문가격
                    {intentSortField === 'order_price' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('deviation')}
                >
                  <div className="flex items-center justify-end gap-1">
                    괴리율
                    {intentSortField === 'deviation' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>타입</TableHead>
                <TableHead
                  className="text-right cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('qty')}
                >
                  <div className="flex items-center justify-end gap-1">
                    수량
                    {intentSortField === 'qty' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>주문유형</TableHead>
                <TableHead>사유</TableHead>
                <TableHead>상태</TableHead>
                <TableHead
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleIntentSort('created_ts')}
                >
                  <div className="flex items-center gap-1">
                    생성시각
                    {intentSortField === 'created_ts' && (
                      <span className="text-xs">{intentSortOrder === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-center">상태</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(() => {
                // 승인 완료된 intent만 표시 (NEW, SUBMITTED)
                const executingIntents = sortedIntents.filter(i =>
                  (i.status === 'NEW' || i.status === 'SUBMITTED') &&
                  holdings.some(h => h.symbol === i.symbol && h.qty > 0)
                )

                if (executingIntents.length === 0) {
                  return (
                    <TableRow>
                      <TableCell colSpan={12} className="text-center text-muted-foreground">
                        승인 완료된 Intent가 없습니다
                      </TableCell>
                    </TableRow>
                  )
                }

                return executingIntents.map((intent) => {
                    // holdings에서 현재가 정보 가져오기
                    const holding = holdings.find(h => h.symbol === intent.symbol)
                    const currentPrice = typeof holding?.current_price === 'string'
                      ? parseFloat(holding.current_price)
                      : (holding?.current_price || 0)
                    const pnlPct = holding?.pnl_pct || 0
                    const avgPrice = holding
                      ? (typeof holding.avg_price === 'string' ? parseFloat(holding.avg_price) : holding.avg_price)
                      : 0

                    // 주문가격 (limit_price 또는 현재가)
                    const orderPrice = intent.limit_price || currentPrice

                    // 괴리율 계산: (현재가 - 주문가격) / 주문가격 * 100
                    const deviationPct = orderPrice > 0 ? ((currentPrice - orderPrice) / orderPrice) * 100 : 0

                    return (
                      <TableRow key={intent.intent_id}>
                        <TableCell>
                          <StockSymbol
                            symbol={intent.symbol}
                            symbolName={intent.symbol_name}
                            size="sm"
                            isHolding={!!holding}
                            isExitEnabled={holding?.exit_mode === 'ENABLED'}
                            market={holding?.raw?.market}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">
                          <ChangeIndicator
                            changePrice={holding?.change_price}
                            changeRate={holding?.change_rate}
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono text-muted-foreground">
                          {holding ? formatNumber(avgPrice, 0) : '-'}
                        </TableCell>
                        <TableCell className="text-right font-mono">{formatNumber(orderPrice, 0)}</TableCell>
                        <TableCell className="text-right font-mono">{formatPercent(deviationPct)}</TableCell>
                        <TableCell>{intent.intent_type}</TableCell>
                        <TableCell className="text-right">{formatNumber(intent.qty)}</TableCell>
                        <TableCell>{intent.order_type}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{intent.reason_code}</Badge>
                        </TableCell>
                        <TableCell>{getStatusBadge(intent.status)}</TableCell>
                        <TableCell className="text-sm">{formatTimestamp(intent.created_ts)}</TableCell>
                        <TableCell className="text-center">
                          <Badge variant="secondary">{intent.status === 'NEW' ? '주문 대기 중' : '주문 완료'}</Badge>
                        </TableCell>
                      </TableRow>
                    )
                  })
              })()}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* KIS 미체결 주문 */}
      <Card>
        <CardHeader>
          <CardTitle>⏳ KIS 미체결 주문</CardTitle>
          <CardDescription>
            {(() => {
              const buyOrders = kisUnfilledOrders.filter(o => o.Raw?.order_side !== '01')
              const sellOrders = kisUnfilledOrders.filter(o => o.Raw?.order_side === '01')
              const totalAmount = kisUnfilledOrders.reduce((sum, o) => {
                const price = parseFloat(o.Raw?.order_price || '0')
                return sum + (price * o.OpenQty)
              }, 0)

              return (
                <>
                  {kisUnfilledOrders.length}건
                  {buyOrders.length > 0 && `, 매수 ${buyOrders.length}건`}
                  {sellOrders.length > 0 && `, 매도 ${sellOrders.length}건`}
                  {totalAmount > 0 && `, ${formatNumber(totalAmount, 0)}원`}
                </>
              )
            })()}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">순번</TableHead>
                <TableHead>종목명</TableHead>
                <TableHead className="text-right">현재가</TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="text-center">구분</TableHead>
                <TableHead className="text-right">주문가격</TableHead>
                <TableHead className="text-right">괴리율</TableHead>
                <TableHead className="text-right">주문수량</TableHead>
                <TableHead className="text-right">미체결</TableHead>
                <TableHead>주문시간</TableHead>
                <TableHead className="text-center">액션</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {kisUnfilledOrders.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={11} className="text-center text-muted-foreground">
                    미체결 주문이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                kisUnfilledOrders.map((order, index) => {
                  const isBuy = order.Raw?.order_side !== '01'
                  const orderPrice = parseFloat(order.Raw?.order_price || '0')

                  // holdings에서 현재가 정보 가져오기
                  const holding = holdings.find(h => h.symbol === order.Symbol)
                  const currentPrice = typeof holding?.current_price === 'string'
                    ? parseFloat(holding.current_price)
                    : (holding?.current_price || 0)
                  const pnl = holding?.pnl || 0
                  const pnlPct = holding?.pnl_pct || 0

                  // 괴리율 계산: (현재가 - 주문가격) / 주문가격 * 100
                  const deviationPct = orderPrice > 0 ? ((currentPrice - orderPrice) / orderPrice) * 100 : 0

                  return (
                    <TableRow key={order.OrderID}>
                      <TableCell className="text-center text-muted-foreground">{index + 1}</TableCell>
                      <TableCell>
                        <StockSymbol
                          symbol={order.Symbol}
                          symbolName={order.Raw?.stock_name}
                          size="sm"
                          isHolding={!!holding}
                          isExitEnabled={holding?.exit_mode === 'ENABLED'}
                          market={holding?.raw?.market}
                        />
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">
                        <ChangeIndicator
                          changePrice={holding?.change_price}
                          changeRate={holding?.change_rate}
                        />
                      </TableCell>
                      <TableCell className="text-center">
                        <Badge variant={isBuy ? 'default' : 'destructive'}>
                          {isBuy ? '매수' : '매도'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(orderPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">{formatPercent(deviationPct)}</TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(order.Qty)}</TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(order.OpenQty)}</TableCell>
                      <TableCell className="text-sm font-mono">{order.Raw?.order_time || '-'}</TableCell>
                      <TableCell className="text-center">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleCancelOrder(order.OrderID, order.Raw?.stock_name)}
                        >
                          삭제
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* KIS 체결 주문 */}
      <Card>
        <CardHeader>
          <CardTitle>✅ KIS 체결 주문</CardTitle>
          <CardDescription>
            {(() => {
              const buyFills = kisFilledOrders.filter(f => f.Raw?.order_side !== '01')
              const sellFills = kisFilledOrders.filter(f => f.Raw?.order_side === '01')
              const buyAmount = buyFills.reduce((sum, f) => sum + (parseFloat(f.Price) * f.Qty), 0)
              const sellAmount = sellFills.reduce((sum, f) => sum + (parseFloat(f.Price) * f.Qty), 0)

              return (
                <>
                  {kisFilledOrders.length}건
                  {buyFills.length > 0 && `, 매수 ${buyFills.length}건, ${formatNumber(buyAmount, 0)}원`}
                  {sellFills.length > 0 && `, 매도 ${sellFills.length}건, ${formatNumber(sellAmount, 0)}원`}
                </>
              )
            })()}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">순번</TableHead>
                <TableHead>종목명</TableHead>
                <TableHead className="text-right">현재가</TableHead>
                <TableHead className="text-right">전일대비</TableHead>
                <TableHead className="text-center">구분</TableHead>
                <TableHead className="text-right">체결가</TableHead>
                <TableHead className="text-right">체결수량</TableHead>
                <TableHead className="text-right">체결금액</TableHead>
                <TableHead>체결시간</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {kisFilledOrders.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground">
                    체결 내역이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                kisFilledOrders.map((fill, index) => {
                  const fillPrice = parseFloat(fill.Price)
                  const fillQty = fill.Qty
                  const fillAmount = fillPrice * fillQty
                  const isBuy = fill.Raw?.order_side !== '01'

                  // holdings에서 현재가 정보 가져오기
                  const holding = holdings.find(h => h.symbol === fill.Symbol)
                  const currentPrice = typeof holding?.current_price === 'string'
                    ? parseFloat(holding.current_price)
                    : (holding?.current_price || 0)
                  const pnl = holding?.pnl || 0
                  const pnlPct = holding?.pnl_pct || 0

                  return (
                    <TableRow key={fill.ExecID}>
                      <TableCell className="text-center text-muted-foreground">{index + 1}</TableCell>
                      <TableCell>
                        <StockSymbol
                          symbol={fill.Symbol}
                          symbolName={fill.Raw?.stock_name}
                          size="sm"
                          isHolding={!!holding}
                          isExitEnabled={holding?.exit_mode === 'ENABLED'}
                          market={holding?.raw?.market}
                        />
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(currentPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">
                        <ChangeIndicator
                          changePrice={holding?.change_price}
                          changeRate={holding?.change_rate}
                        />
                      </TableCell>
                      <TableCell className="text-center">
                        <Badge variant={isBuy ? 'default' : 'destructive'}>
                          {isBuy ? '매수' : '매도'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(fillPrice, 0)}</TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(fillQty)}</TableCell>
                      <TableCell className="text-right font-mono">{formatNumber(fillAmount, 0)}</TableCell>
                      <TableCell className="text-sm font-mono">{formatTimestamp(fill.Timestamp)}</TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Exit 규칙 관리 다이얼로그 */}
      <Dialog open={rulesDialogOpen} onOpenChange={setRulesDialogOpen}>
        <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Exit 규칙 요약 (v14 원칙)</DialogTitle>
            <DialogDescription>
              실시간 Exit Engine 규칙 및 운영 원칙
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {/* 손절 규칙 */}
            <div className="space-y-2">
              <div className="font-semibold text-base flex items-center gap-2">
                <span style={{ color: '#2196F3' }}>▼ 손절 (Stop Loss)</span>
              </div>
              <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                <div>• SL1 (-3%): 잔량의 50% 청산</div>
                <div>• SL2 (-5%): 잔량의 100% 강제 청산</div>
              </div>
            </div>

            {/* 익절 규칙 */}
            <div className="space-y-2">
              <div className="font-semibold text-base flex items-center gap-2">
                <span style={{ color: '#EA5455' }}>▲ 익절 (Take Profit)</span>
              </div>
              <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                <div>• TP1 (+7%): 원본의 10% 청산 → <span className="font-semibold text-foreground">Stop Floor 활성화</span></div>
                <div>• TP2 (+10%): 원본의 20% 청산 → 부분 트레일링 활성화</div>
                <div>• TP3 (+15%): 원본의 30% 청산 → 잔량 트레일링 활성화</div>
                <div className="text-xs mt-2 p-2 bg-muted rounded">
                  ※ TP 합계 60%, 잔량 40%는 Stop Floor 및 Trailing으로 관리
                </div>
              </div>
            </div>

            {/* Stop Floor */}
            <div className="space-y-2 border-l-4 border-yellow-500 pl-4 bg-yellow-50 dark:bg-yellow-950/20 p-3 rounded-r">
              <div className="font-semibold text-base flex items-center gap-2">
                <span className="text-yellow-600 dark:text-yellow-500">🛡️ Stop Floor (본전+0.6%)</span>
              </div>
              <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                <div>• <span className="font-semibold text-foreground">TP1 체결 시 즉시 활성화</span> (v14 핵심 안전장치)</div>
                <div>• stop_floor_price = 평단가 × 1.006</div>
                <div>• 가격이 Stop Floor 이하로 내려가면 → 잔량 전량 청산</div>
                <div className="text-xs mt-2 p-2 bg-muted rounded">
                  ※ TP1 이후 수익을 보호하고 손실 전환을 구조적으로 차단
                </div>
              </div>
            </div>

            {/* 트레일링 */}
            <div className="space-y-2">
              <div className="font-semibold text-base flex items-center gap-2">
                <span>🎯 트레일링 (Trailing)</span>
              </div>
              <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                <div>• <span className="font-semibold text-foreground">트레일링 거리</span>: HWM 대비 -3%</div>
                <div>• <span className="font-semibold text-foreground">TP2 이후</span>: HWM 대비 -3% 도달 시 → 원본의 20% 청산 (부분 트레일링)</div>
                <div>• <span className="font-semibold text-foreground">TP3 이후</span>: HWM 대비 -3% 도달 시 → 잔량 전량 청산 (잔량 트레일링)</div>
              </div>
            </div>

            {/* 운영 원칙 */}
            <div className="space-y-2 pt-4 border-t border-border">
              <div className="font-semibold text-sm text-muted-foreground">운영 원칙</div>
              <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                <div>• 3초마다 OPEN 포지션 평가 (10초 초과 가격 데이터 사용 금지)</div>
                <div>• 우선순위: <span className="font-semibold text-red-600 dark:text-red-400">HARDSTOP (0번)</span> → SL2 → Stop Floor → SL1 → TP3 → TP2 → TP1 → Trailing</div>
              </div>
            </div>

            {/* v14 개선사항 */}
            <div className="space-y-3 pt-4 border-t border-border">
              <div className="font-semibold text-base flex items-center gap-2">
                <span className="text-blue-600 dark:text-blue-400">🚀 v14 핵심 개선사항</span>
              </div>

              {/* HARDSTOP */}
              <div className="space-y-2 border-l-4 border-red-500 pl-4 bg-red-50 dark:bg-red-950/20 p-3 rounded-r">
                <div className="font-semibold text-sm flex items-center gap-2">
                  <span className="text-red-600 dark:text-red-400">🚨 HARDSTOP (비상 손절)</span>
                </div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• <span className="font-semibold text-foreground">우선순위 0번</span> - 모든 트리거보다 먼저 평가</div>
                  <div>• <span className="font-semibold text-red-600 dark:text-red-400">PAUSE_ALL 모드에서도 작동</span> (제어 모드 우회)</div>
                  <div>• 기본값: -10% (설정 가능)</div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ 시스템 전체가 일시정지 상태여도 비상 손절은 계속 작동하여 큰 손실 방지
                  </div>
                </div>
              </div>

              {/* action_key Phase 포함 */}
              <div className="space-y-2">
                <div className="font-semibold text-sm">📋 action_key Phase 포함</div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• 형식: <code className="bg-muted px-1 rounded text-xs">{'{'}position_id{'}'}:{'{'}phase{'}'}:{'{'}reason_code{'}'}</code></div>
                  <div>• 예시: <code className="bg-muted px-1 rounded text-xs">abc-123:OPEN:TP1</code></div>
                  <div>• <span className="font-semibold text-foreground">평단가 리셋 후 재발동 가능</span></div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ 추가매수로 Phase=OPEN 리셋 시 동일 트리거 재평가 가능 (TP1 → 추가매수 → TP1 재발동)
                  </div>
                </div>
              </div>

              {/* breach_ticks 분리 */}
              <div className="space-y-2">
                <div className="font-semibold text-sm">🎯 breach_ticks 독립화</div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• StopFloor 전용 카운터: <code className="bg-muted px-1 rounded text-xs">stop_floor_breach_ticks</code></div>
                  <div>• Trailing 전용 카운터: <code className="bg-muted px-1 rounded text-xs">trailing_breach_ticks</code></div>
                  <div>• <span className="font-semibold text-foreground">연속 조건 오염 방지</span></div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ 각 트리거가 독립 카운트하여 오작동 방지 (StopFloor 2틱 + Trailing 1틱 = 3틱 X)
                  </div>
                </div>
              </div>

              {/* 평단가 리셋 로직 */}
              <div className="space-y-2">
                <div className="font-semibold text-sm">💰 평단가 리셋 로직 개선</div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• <span className="font-semibold text-foreground">추가매수 (≥2%)</span>: Phase=OPEN 리셋, 모든 트리거 재평가</div>
                  <div>• <span className="font-semibold text-foreground">부분체결 (0.5~2%)</span>: Phase 유지, State 보호</div>
                  <div>• &lt;0.5%: 무시</div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ TP1 체결 후 일부 매도 시 StopFloor 유지 (기존: 손실 → v14: 유지 ✅)
                  </div>
                </div>
              </div>

              {/* ProfileResolver */}
              <div className="space-y-2">
                <div className="font-semibold text-sm">⚙️ ProfileResolver 3단계 우선순위</div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• <span className="font-semibold text-foreground">1순위: Position 설정</span> (positions.exit_profile_id)</div>
                  <div>• <span className="font-semibold text-foreground">2순위: Symbol 설정</span> (symbol_exit_overrides)</div>
                  <div>• <span className="font-semibold text-foreground">3순위: Default Profile</span></div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ 각 포지션/종목별 맞춤 Exit 규칙 적용 가능
                  </div>
                </div>
              </div>

              {/* Intent 상태 통일 */}
              <div className="space-y-2">
                <div className="font-semibold text-sm">🔄 Intent 상태 정의 통일</div>
                <div className="ml-4 space-y-1 text-sm text-muted-foreground">
                  <div>• 활성 상태: <code className="bg-muted px-1 rounded text-xs">NEW</code>, <code className="bg-muted px-1 rounded text-xs">PENDING_APPROVAL</code>, <code className="bg-muted px-1 rounded text-xs">ACK</code></div>
                  <div>• <span className="font-semibold text-foreground">중복 검사 일관성</span> (Evaluator ↔ Reconciliation)</div>
                  <div className="text-xs mt-2 p-2 bg-muted rounded">
                    ※ 두 모듈에서 동일한 ActiveIntentStatuses 사용
                  </div>
                </div>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* StockDetailSheet - v10 스타일 종목 상세 */}
      <StockDetailSheet
        stock={selectedStock}
        open={isStockDetailOpen}
        onOpenChange={handleStockDetailOpenChange}
        holdings={holdings}
        unfilledOrders={kisUnfilledOrders}
        executedOrders={kisFilledOrders}
        totalEvaluation={totalEvaluation}
        onExitModeToggle={handleExitModeToggle}
      />
    </div>
  )
}
