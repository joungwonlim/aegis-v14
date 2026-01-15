'use client'

import { useState, useRef } from 'react'
import { ShoppingCart, CheckCircle, Clock, TrendingUp, TrendingDown, ChevronUp, ChevronDown } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { placeKISOrder } from '@/lib/api'

interface OrderTabProps {
  symbol: string
  symbolName: string
  currentPrice: number
  holding: any | null  // 보유 정보 (보유수량, 평가손익 등)
  unfilledOrders: any[]
  executedOrders: any[]
}

/**
 * Order 탭 - 주문 내역 표시 + 빠른 주문
 *
 * Phase 1: 미체결/체결 주문 목록 ✅
 * Phase 2: 빠른 주문 기능 ✅
 *   - 매수/매도, 지정가/시장가 선택
 *   - 주문수량/가격 입력
 *   - 현재가 동기화
 *   - KIS API 주문 제출
 *   - 중복 실행 방지
 */
export function OrderTab({
  symbol,
  symbolName,
  currentPrice,
  holding,
  unfilledOrders,
  executedOrders,
}: OrderTabProps) {
  // 주문 폼 상태
  const [orderSide, setOrderSide] = useState<'buy' | 'sell'>('buy')
  const [orderType, setOrderType] = useState<'limit' | 'market'>('limit')
  const [quantity, setQuantity] = useState<string>('')
  const [price, setPrice] = useState<string>(currentPrice > 0 ? Math.floor(currentPrice).toString() : '')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null)
  const isExecutingRef = useRef(false)

  // 해당 종목의 미체결 주문 필터링
  const symbolUnfilled = unfilledOrders.filter((o) => o.Symbol === symbol)
  const symbolExecuted = executedOrders.filter((o) => o.Symbol === symbol)

  const formatNumber = (n: number | null | undefined) => {
    if (n == null || n === 0) return '-'
    return n.toLocaleString()
  }

  const formatTime = (ts: string | null | undefined) => {
    if (!ts) return '-'
    // DB timestamp는 KST이지만 timezone 정보가 없어서 UTC로 해석됨
    // +09:00을 추가하여 KST로 명시
    const kstTimestamp = ts.includes('+') || ts.includes('Z') ? ts : `${ts}+09:00`
    const date = new Date(kstTimestamp)
    return date.toLocaleTimeString('ko-KR', {
      timeZone: 'Asia/Seoul',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // 현재가 동기화
  const handlePriceSync = () => {
    if (currentPrice > 0) {
      setPrice(Math.floor(currentPrice).toString())
    }
  }

  // 주문가격 증감
  const handlePriceIncrement = () => {
    const currentPriceValue = parseInt(price) || 0
    setPrice((currentPriceValue + 1).toString())
  }

  const handlePriceDecrement = () => {
    const currentPriceValue = parseInt(price) || 0
    if (currentPriceValue > 1) {
      setPrice((currentPriceValue - 1).toString())
    }
  }

  // 주문수량 빠른 조절
  const handleQuantityAdjust = (delta: number) => {
    const currentQty = parseInt(quantity) || 0
    const newQty = Math.max(0, currentQty + delta)
    setQuantity(newQty.toString())
  }

  // 주문 총액 계산
  const totalAmount = (() => {
    const qty = parseInt(quantity) || 0
    const prc = orderType === 'market' ? currentPrice : (parseInt(price) || 0)
    return qty * prc
  })()

  // 주문 제출
  const handleSubmit = async () => {
    // 중복 실행 방지
    if (isExecutingRef.current) {
      console.warn('[OrderTab] Submit already in progress')
      return
    }

    if (!quantity || parseInt(quantity) <= 0) {
      setResult({ success: false, message: '주문수량을 입력해주세요' })
      return
    }

    if (orderType === 'limit' && (!price || parseInt(price) <= 0)) {
      setResult({ success: false, message: '주문가격을 입력해주세요' })
      return
    }

    isExecutingRef.current = true
    setIsSubmitting(true)
    setResult(null)

    try {
      const response = await placeKISOrder({
        symbol,
        side: orderSide,
        order_type: orderType,
        qty: parseInt(quantity),
        price: orderType === 'market' ? 0 : parseInt(price),
      })

      if (response.success) {
        setResult({
          success: true,
          message: `주문 완료! 주문번호: ${response.order_id}`,
        })
        setQuantity('')
      } else {
        setResult({
          success: false,
          message: response.error || '주문 실패',
        })
      }
    } catch (error) {
      setResult({
        success: false,
        message: error instanceof Error ? error.message : '서버 연결 오류',
      })
    } finally {
      isExecutingRef.current = false
      setIsSubmitting(false)
    }
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

      {/* 빠른 주문 폼 (v10 스타일) */}
      <div className="rounded-lg border p-6">
        <div className="mb-4 flex items-center gap-2">
          <ShoppingCart className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">빠른 주문</h3>
        </div>

        <div className="space-y-4">
          {/* 보유 정보 (holding이 있을 때만) */}
          {holding && (
            <div className="rounded-lg border border-green-500/30 bg-green-500/10 p-4">
              <div className="mb-2 text-sm font-medium text-green-600 dark:text-green-400">보유 정보</div>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <div className="text-muted-foreground">보유수량</div>
                  <div className="font-semibold">{holding.qty?.toLocaleString()}주</div>
                </div>
                <div>
                  <div className="text-muted-foreground">평균매수단가</div>
                  <div className="font-mono font-semibold">{Math.floor(holding.avg_price || 0).toLocaleString()}원</div>
                </div>
                <div>
                  <div className="text-muted-foreground">평가손익</div>
                  <div className={`font-semibold ${holding.pnl >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                    {holding.pnl >= 0 ? '+' : ''}{holding.pnl?.toLocaleString()}원
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">손익률</div>
                  <div className={`font-semibold ${holding.pnl_pct >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                    {holding.pnl_pct >= 0 ? '+' : ''}{holding.pnl_pct?.toFixed(2)}%
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* 매수/매도 큰 버튼 */}
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={orderSide === 'buy' ? 'default' : 'outline'}
              className={`h-12 text-base font-semibold ${
                orderSide === 'buy'
                  ? 'bg-red-500 hover:bg-red-600 text-white'
                  : 'hover:bg-muted'
              }`}
              onClick={() => setOrderSide('buy')}
            >
              매수
            </Button>
            <Button
              variant={orderSide === 'sell' ? 'default' : 'outline'}
              className={`h-12 text-base font-semibold ${
                orderSide === 'sell'
                  ? 'bg-blue-500 hover:bg-blue-600 text-white'
                  : 'hover:bg-muted'
              }`}
              onClick={() => setOrderSide('sell')}
            >
              매도
            </Button>
          </div>

          {/* 지정가/시장가 큰 버튼 */}
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={orderType === 'limit' ? 'default' : 'outline'}
              className={`h-10 ${
                orderType === 'limit'
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-muted'
              }`}
              onClick={() => setOrderType('limit')}
            >
              지정가
            </Button>
            <Button
              variant={orderType === 'market' ? 'default' : 'outline'}
              className={`h-10 ${
                orderType === 'market'
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-muted'
              }`}
              onClick={() => setOrderType('market')}
            >
              시장가
            </Button>
          </div>

          {/* 주문가격 (지정가일 때만) */}
          {orderType === 'limit' && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>주문가격</Label>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handlePriceSync}
                  className="h-6 text-xs text-primary"
                >
                  현재가 적용
                </Button>
              </div>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  placeholder="주문가격 입력"
                  className="font-mono text-right"
                  min="1"
                />
                <div className="flex flex-col gap-1">
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-6 w-8"
                    onClick={handlePriceIncrement}
                  >
                    <ChevronUp className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-6 w-8"
                    onClick={handlePriceDecrement}
                  >
                    <ChevronDown className="h-3 w-3" />
                  </Button>
                </div>
              </div>
            </div>
          )}

          {/* 주문수량 */}
          <div className="space-y-2">
            <Label>주문수량</Label>
            <Input
              type="number"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              placeholder="주문수량 입력"
              className="text-right"
              min="1"
            />
            {/* 빠른 수량 조절 버튼 */}
            <div className="grid grid-cols-4 gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleQuantityAdjust(10)}
              >
                +10
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleQuantityAdjust(50)}
              >
                +50
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleQuantityAdjust(100)}
              >
                +100
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleQuantityAdjust(1000)}
              >
                +1000
              </Button>
            </div>
          </div>

          {/* 총 주문금액 */}
          <div className="rounded-lg bg-muted/50 p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">총 주문금액</span>
              <span className="text-lg font-bold font-mono">
                {totalAmount > 0 ? `${totalAmount.toLocaleString()}원` : '-'}
              </span>
            </div>
          </div>

          {/* 주문 실행 버튼 (큰 버튼) */}
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting}
            className={`w-full h-12 text-lg font-semibold ${
              orderSide === 'buy'
                ? 'bg-red-500 hover:bg-red-600'
                : 'bg-blue-500 hover:bg-blue-600'
            }`}
          >
            {isSubmitting ? '주문 중...' : `🛒 ${orderSide === 'buy' ? '한할 매수하기' : '한할 매도하기'}`}
          </Button>

          {/* 결과 메시지 */}
          {result && (
            <div
              className={`rounded-lg p-3 text-sm ${
                result.success
                  ? 'bg-green-500/10 text-green-500 border border-green-500/20'
                  : 'bg-red-500/10 text-red-500 border border-red-500/20'
              }`}
            >
              {result.message}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
