'use client'

import { ShoppingCart, CheckCircle, Clock } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface OrderTabProps {
  symbol: string
  symbolName: string
  unfilledOrders: any[]
  executedOrders: any[]
}

/**
 * Order 탭 - 주문 내역 표시
 *
 * Phase 1: 미체결/체결 주문 목록
 */
export function OrderTab({
  symbol,
  symbolName,
  unfilledOrders,
  executedOrders,
}: OrderTabProps) {
  // 해당 종목의 미체결 주문 필터링
  const symbolUnfilled = unfilledOrders.filter((o) => o.Symbol === symbol)
  const symbolExecuted = executedOrders.filter((o) => o.Symbol === symbol)

  const formatNumber = (n: number | null | undefined) => {
    if (n == null || n === 0) return '-'
    return n.toLocaleString()
  }

  const formatTime = (ts: string | null | undefined) => {
    if (!ts) return '-'
    const date = new Date(ts)
    return date.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="space-y-6 p-4">
      {/* 미체결 주문 */}
      <div>
        <div className="mb-3 flex items-center gap-2">
          <Clock className="h-5 w-5 text-orange-500" />
          <h3 className="text-lg font-semibold">미체결 주문</h3>
          <span className="rounded-full bg-orange-500/10 px-2 py-0.5 text-xs font-medium text-orange-500">
            {symbolUnfilled.length}건
          </span>
        </div>

        {symbolUnfilled.length === 0 ? (
          <div className="rounded-lg border bg-muted/50 p-6 text-center text-sm text-muted-foreground">
            미체결 주문이 없습니다
          </div>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>구분</TableHead>
                  <TableHead className="text-right">주문가</TableHead>
                  <TableHead className="text-right">주문수량</TableHead>
                  <TableHead className="text-right">미체결</TableHead>
                  <TableHead>주문시간</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {symbolUnfilled.map((order, idx) => {
                  const isBuy = order.Raw?.order_side !== '01'
                  return (
                    <TableRow key={idx}>
                      <TableCell>
                        <span className={`font-medium ${isBuy ? 'text-red-500' : 'text-blue-500'}`}>
                          {isBuy ? '매수' : '매도'}
                        </span>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatNumber(parseFloat(order.Raw?.order_price || '0'))}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatNumber(order.OrderQty)}
                      </TableCell>
                      <TableCell className="text-right font-medium text-orange-500">
                        {formatNumber(order.OpenQty)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatTime(order.OrderedTS)}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {/* 체결 주문 */}
      <div>
        <div className="mb-3 flex items-center gap-2">
          <CheckCircle className="h-5 w-5 text-green-500" />
          <h3 className="text-lg font-semibold">체결 주문</h3>
          <span className="rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-500">
            {symbolExecuted.length}건
          </span>
        </div>

        {symbolExecuted.length === 0 ? (
          <div className="rounded-lg border bg-muted/50 p-6 text-center text-sm text-muted-foreground">
            체결 주문이 없습니다
          </div>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>구분</TableHead>
                  <TableHead className="text-right">체결가</TableHead>
                  <TableHead className="text-right">체결수량</TableHead>
                  <TableHead className="text-right">체결금액</TableHead>
                  <TableHead>체결시간</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {symbolExecuted.slice(0, 10).map((order, idx) => {
                  const isBuy = order.Raw?.order_side !== '01'
                  const fillPrice = parseFloat(order.Raw?.fill_price || '0')
                  const fillQty = order.FilledQty || 0
                  const fillAmount = fillPrice * fillQty

                  return (
                    <TableRow key={idx}>
                      <TableCell>
                        <span className={`font-medium ${isBuy ? 'text-red-500' : 'text-blue-500'}`}>
                          {isBuy ? '매수' : '매도'}
                        </span>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatNumber(fillPrice)}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatNumber(fillQty)}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatNumber(fillAmount)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatTime(order.FilledTS)}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {/* TODO Phase 2: 빠른 주문 버튼 */}
      <div className="rounded-lg border bg-muted/50 p-6 text-center text-sm text-muted-foreground">
        🚀 빠른 주문 기능은 Phase 2에서 추가됩니다
      </div>
    </div>
  )
}
