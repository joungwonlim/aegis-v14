'use client'

import { useState, useRef } from 'react'
import { ShoppingCart, CheckCircle, Clock, TrendingUp, TrendingDown } from 'lucide-react'
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { placeKISOrder } from '@/lib/api'

interface OrderTabProps {
  symbol: string
  symbolName: string
  currentPrice: number
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

      {/* 빠른 주문 폼 */}
      <div className="rounded-lg border p-6">
        <div className="mb-4 flex items-center gap-2">
          <ShoppingCart className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold">빠른 주문</h3>
        </div>

        <div className="space-y-4">
          {/* 매수/매도 선택 */}
          <div className="space-y-2">
            <Label>구분</Label>
            <RadioGroup value={orderSide} onValueChange={(v) => setOrderSide(v as 'buy' | 'sell')}>
              <div className="flex gap-4">
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="buy" id="buy" />
                  <Label htmlFor="buy" className="cursor-pointer font-medium text-red-500">
                    <TrendingUp className="inline h-4 w-4 mr-1" />
                    매수
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="sell" id="sell" />
                  <Label htmlFor="sell" className="cursor-pointer font-medium text-blue-500">
                    <TrendingDown className="inline h-4 w-4 mr-1" />
                    매도
                  </Label>
                </div>
              </div>
            </RadioGroup>
          </div>

          {/* 주문 유형 선택 */}
          <div className="space-y-2">
            <Label>주문 유형</Label>
            <RadioGroup value={orderType} onValueChange={(v) => setOrderType(v as 'limit' | 'market')}>
              <div className="flex gap-4">
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="limit" id="limit" />
                  <Label htmlFor="limit" className="cursor-pointer">지정가</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="market" id="market" />
                  <Label htmlFor="market" className="cursor-pointer">시장가</Label>
                </div>
              </div>
            </RadioGroup>
          </div>

          {/* 주문 수량 */}
          <div className="space-y-2">
            <Label htmlFor="quantity">주문수량</Label>
            <Input
              id="quantity"
              type="number"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              placeholder="주문수량 입력"
              min="1"
            />
          </div>

          {/* 주문 가격 (지정가일 때만) */}
          {orderType === 'limit' && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="price">주문가격</Label>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handlePriceSync}
                  className="h-6 text-xs"
                >
                  현재가 동기화
                </Button>
              </div>
              <Input
                id="price"
                type="number"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                placeholder="주문가격 입력"
                min="1"
              />
            </div>
          )}

          {/* 주문 총액 */}
          <div className="rounded-lg bg-muted/50 p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">주문 총액</span>
              <span className="text-lg font-bold">
                {totalAmount > 0 ? `${totalAmount.toLocaleString()}원` : '-'}
              </span>
            </div>
          </div>

          {/* 주문 버튼 */}
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting}
            className={`w-full ${orderSide === 'buy' ? 'bg-red-500 hover:bg-red-600' : 'bg-blue-500 hover:bg-blue-600'}`}
          >
            {isSubmitting ? '주문 중...' : `${orderSide === 'buy' ? '매수' : '매도'} 주문`}
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
