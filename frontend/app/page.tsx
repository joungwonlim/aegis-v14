'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { getHoldings, getOrderIntents, getOrders, getFills, type Holding, type OrderIntent, type Order, type Fill } from '@/lib/api'

export default function RuntimeDashboard() {
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [intents, setIntents] = useState<OrderIntent[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [fills, setFills] = useState<Fill[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)

      const [holdingsData, intentsData, ordersData, fillsData] = await Promise.all([
        getHoldings(),
        getOrderIntents(),
        getOrders(),
        getFills(),
      ])

      setHoldings(holdingsData)
      setIntents(intentsData)
      setOrders(ordersData)
      setFills(fillsData)
      setLastUpdate(new Date())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()

    // Auto-refresh every 5 seconds
    const interval = setInterval(loadData, 5000)
    return () => clearInterval(interval)
  }, [])

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
      NEW: 'secondary',
      SUBMITTED: 'default',
      FILLED: 'default',
      PARTIAL: 'secondary',
      FAILED: 'destructive',
      REJECTED: 'destructive',
      CANCELLED: 'outline',
      DUPLICATE: 'outline',
    }

    return <Badge variant={variants[status] || 'default'}>{status}</Badge>
  }

  const formatNumber = (value: number | undefined, decimals = 0) => {
    return value?.toLocaleString('ko-KR', { minimumFractionDigits: decimals, maximumFractionDigits: decimals }) ?? '-'
  }

  const formatPercent = (value: number | undefined) => {
    if (value === undefined) return '-'
    const formatted = value.toFixed(2)
    const color = value >= 0 ? 'text-green-600' : 'text-red-600'
    return <span className={color}>{formatted}%</span>
  }

  const formatTimestamp = (ts: string) => {
    return new Date(ts).toLocaleString('ko-KR', {
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
          <h1 className="text-3xl font-bold">Aegis v14 Runtime</h1>
          <p className="text-muted-foreground">Core Trading Engine Monitor</p>
        </div>
        <div className="flex items-center gap-4">
          {lastUpdate && (
            <span className="text-sm text-muted-foreground">
              Last updated: {lastUpdate.toLocaleTimeString('ko-KR')}
            </span>
          )}
          <Button onClick={loadData} disabled={loading}>
            {loading ? '새로고침 중...' : '새로고침'}
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

      {/* Holdings */}
      <Card>
        <CardHeader>
          <CardTitle>보유종목 (Holdings)</CardTitle>
          <CardDescription>
            현재 보유 중인 포지션 ({holdings.length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>종목코드</TableHead>
                <TableHead>종목명</TableHead>
                <TableHead className="text-right">수량</TableHead>
                <TableHead className="text-right">평균단가</TableHead>
                <TableHead className="text-right">현재가</TableHead>
                <TableHead className="text-right">평가손익</TableHead>
                <TableHead className="text-right">수익률</TableHead>
                <TableHead>갱신시각</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {holdings.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    보유종목이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                holdings.map((holding) => (
                  <TableRow key={`${holding.account_id}-${holding.symbol}`}>
                    <TableCell className="font-mono">{holding.symbol}</TableCell>
                    <TableCell>{holding.name || '-'}</TableCell>
                    <TableCell className="text-right">{formatNumber(holding.qty)}</TableCell>
                    <TableCell className="text-right">{formatNumber(holding.avg_price, 0)}</TableCell>
                    <TableCell className="text-right">{formatNumber(holding.current_price, 0)}</TableCell>
                    <TableCell className="text-right">{formatNumber(holding.pnl, 0)}</TableCell>
                    <TableCell className="text-right">{formatPercent(holding.pnl_pct)}</TableCell>
                    <TableCell className="text-sm">{formatTimestamp(holding.updated_ts)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Order Intents */}
      <Card>
        <CardHeader>
          <CardTitle>Order Intents (Exit 평가 결과)</CardTitle>
          <CardDescription>
            Exit Engine이 생성한 주문 의도 ({intents.length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>종목코드</TableHead>
                <TableHead>타입</TableHead>
                <TableHead className="text-right">수량</TableHead>
                <TableHead>주문유형</TableHead>
                <TableHead>사유</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>생성시각</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {intents.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground">
                    Order Intent가 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                intents.map((intent) => (
                  <TableRow key={intent.intent_id}>
                    <TableCell className="font-mono">{intent.symbol}</TableCell>
                    <TableCell>{intent.intent_type}</TableCell>
                    <TableCell className="text-right">{formatNumber(intent.qty)}</TableCell>
                    <TableCell>{intent.order_type}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{intent.reason_code}</Badge>
                    </TableCell>
                    <TableCell>{getStatusBadge(intent.status)}</TableCell>
                    <TableCell className="text-sm">{formatTimestamp(intent.created_ts)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Orders */}
      <Card>
        <CardHeader>
          <CardTitle>주문 (Orders)</CardTitle>
          <CardDescription>
            KIS에 제출된 주문 ({orders.length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>주문번호</TableHead>
                <TableHead>종목코드</TableHead>
                <TableHead className="text-right">주문수량</TableHead>
                <TableHead className="text-right">미체결</TableHead>
                <TableHead className="text-right">체결</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>브로커상태</TableHead>
                <TableHead>제출시각</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    주문이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                orders.map((order) => (
                  <TableRow key={order.order_id}>
                    <TableCell className="font-mono text-sm">{order.order_id.slice(0, 8)}...</TableCell>
                    <TableCell className="font-mono">{order.symbol || '-'}</TableCell>
                    <TableCell className="text-right">{formatNumber(order.qty)}</TableCell>
                    <TableCell className="text-right">{formatNumber(order.open_qty)}</TableCell>
                    <TableCell className="text-right">{formatNumber(order.filled_qty)}</TableCell>
                    <TableCell>{getStatusBadge(order.status)}</TableCell>
                    <TableCell>{order.broker_status}</TableCell>
                    <TableCell className="text-sm">{formatTimestamp(order.submitted_ts)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Fills */}
      <Card>
        <CardHeader>
          <CardTitle>체결 (Fills)</CardTitle>
          <CardDescription>
            주문 체결 내역 ({fills.length}개)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>체결번호</TableHead>
                <TableHead>주문번호</TableHead>
                <TableHead>종목코드</TableHead>
                <TableHead className="text-right">수량</TableHead>
                <TableHead className="text-right">가격</TableHead>
                <TableHead className="text-right">수수료</TableHead>
                <TableHead className="text-right">세금</TableHead>
                <TableHead>체결시각</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {fills.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    체결 내역이 없습니다
                  </TableCell>
                </TableRow>
              ) : (
                fills.map((fill) => (
                  <TableRow key={fill.exec_id}>
                    <TableCell className="font-mono text-sm">{fill.exec_id.slice(0, 8)}...</TableCell>
                    <TableCell className="font-mono text-sm">{fill.order_id.slice(0, 8)}...</TableCell>
                    <TableCell className="font-mono">{fill.symbol || '-'}</TableCell>
                    <TableCell className="text-right">{formatNumber(fill.qty)}</TableCell>
                    <TableCell className="text-right">{formatNumber(fill.price, 0)}</TableCell>
                    <TableCell className="text-right">{formatNumber(fill.fee, 0)}</TableCell>
                    <TableCell className="text-right">{formatNumber(fill.tax, 0)}</TableCell>
                    <TableCell className="text-sm">{formatTimestamp(fill.timestamp)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
