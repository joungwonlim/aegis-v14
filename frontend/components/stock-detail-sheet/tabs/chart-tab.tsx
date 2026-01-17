'use client'

import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getPriceHistory, getFlowHistory } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  ReferenceLine,
  ComposedChart,
  Cell,
} from 'recharts'
import type { DailyPrice, InvestorFlow } from '../types'

interface ChartTabProps {
  symbol: string
  symbolName: string
  avgBuyPrice?: number  // 평균매입단가 (보유 종목인 경우)
}

type PeriodType = '1M' | '3M' | '6M' | '1Y'

const PERIODS: { label: string; value: PeriodType; days: number }[] = [
  { label: '1개월', value: '1M', days: 30 },
  { label: '3개월', value: '3M', days: 90 },
  { label: '6개월', value: '6M', days: 180 },
  { label: '1년', value: '1Y', days: 365 },
]

const COLORS = {
  foreign: '#F04452',  // 외국인 - 빨강
  inst: '#7B61FF',     // 기관 - 보라
  retail: '#F2A93B',   // 개인 - 노랑
}

function formatKRW(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  if (Math.abs(value) >= 100000000) {
    return `${(value / 100000000).toFixed(1)}억`
  }
  if (Math.abs(value) >= 10000) {
    return `${(value / 10000).toFixed(0)}만`
  }
  return value.toLocaleString()
}

function formatAxisValue(value: number): string {
  const absValue = Math.abs(value)
  if (absValue >= 100000000) {
    return (value / 100000000).toFixed(1) + '억'
  } else if (absValue >= 10000) {
    return (value / 10000).toFixed(0) + '만'
  } else if (absValue >= 1000) {
    return (value / 1000).toFixed(1) + '천'
  }
  return value.toLocaleString()
}

function formatVolume(volume: number): string {
  const absVolume = Math.abs(volume)
  const sign = volume < 0 ? '-' : ''

  if (absVolume >= 100000000) {
    return sign + (absVolume / 100000000).toFixed(2) + '억'
  } else if (absVolume >= 10000) {
    return sign + Math.round(absVolume / 10000).toLocaleString() + '만'
  } else if (absVolume >= 1000) {
    return sign + (absVolume / 1000).toFixed(1) + '천'
  }
  return volume.toLocaleString()
}

function formatVolumeShort(volume: number): string {
  if (volume >= 100000000) {
    return (volume / 100000000).toFixed(1) + '억'
  } else if (volume >= 10000) {
    return Math.round(volume / 10000).toLocaleString() + '만'
  }
  return volume.toLocaleString()
}

// Candlestick shape interface
interface CandleShapeProps {
  x: number
  y: number
  width: number
  height: number
  payload: {
    close: number
    open: number
    high: number
    low: number
  }
  index: number
}

// 일봉 차트 (Candlestick with crosshair)
function PriceChart({ data, avgBuyPrice }: { data: DailyPrice[]; avgBuyPrice?: number }) {
  const [period, setPeriod] = useState<PeriodType>('3M')
  const [mouseY, setMouseY] = useState<number | null>(null)

  // Filter data by period
  const filteredData = useMemo(() => {
    const periodConfig = PERIODS.find(p => p.value === period)
    const days = periodConfig?.days ?? 90
    return data.slice(-days)
  }, [data, period])

  // Calculate Y axis domain and prepare chart data
  const chartData = useMemo(() => {
    if (filteredData.length === 0) return []

    return filteredData.map(d => ({
      date: d.date.slice(5), // MM-DD format
      fullDate: d.date,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
      volume: d.volume,
    }))
  }, [filteredData])

  // Get price range for Y axis
  const [yMin, yMax] = useMemo(() => {
    if (chartData.length === 0) return [0, 100]
    const allPrices = chartData.flatMap(d => [d.high, d.low])
    const min = Math.min(...allPrices)
    const max = Math.max(...allPrices)
    const padding = (max - min) * 0.1
    return [min - padding, max + padding]
  }, [chartData])

  // Convert mouse Y coordinate to price value
  const mousePrice = useMemo(() => {
    if (mouseY === null) return null
    const chartHeight = 350 - 20 // height - (top + bottom margins)
    const marginTop = 10
    const priceAtY = yMax - ((mouseY - marginTop) / chartHeight) * (yMax - yMin)
    return priceAtY
  }, [mouseY, yMin, yMax])

  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[400px] text-muted-foreground">
        일봉 데이터가 없습니다
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header with Period Selector */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">일봉 차트</h3>
        <div className="flex gap-1 bg-muted/30 rounded-lg p-0.5">
          {PERIODS.map(p => (
            <Button
              key={p.value}
              variant={period === p.value ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setPeriod(p.value)}
              className="h-7 px-3 text-xs"
            >
              {p.label}
            </Button>
          ))}
        </div>
      </div>

      {/* 가격 차트 */}
      <ResponsiveContainer width="100%" height={350}>
        <ComposedChart
          data={chartData}
          margin={{ top: 10, right: 80, left: 10, bottom: 0 }}
          onMouseMove={(e: any) => {
            if (e && e.activeCoordinate) {
              setMouseY(e.activeCoordinate.y)
            }
          }}
          onMouseLeave={() => setMouseY(null)}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
            tickLine={{ stroke: 'hsl(var(--border))' }}
            axisLine={{ stroke: 'hsl(var(--border))' }}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={[yMin, yMax]}
            tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
            tickLine={{ stroke: 'hsl(var(--border))' }}
            axisLine={{ stroke: 'hsl(var(--border))' }}
            tickFormatter={(v) => formatKRW(v)}
            width={70}
          />
          <Tooltip
            content={({ active, payload }) => {
              if (!active || !payload || !payload.length) return null
              const data = payload[0].payload
              const isUp = data.close >= data.open
              return (
                <div className="bg-popover border rounded-lg p-3 shadow-lg">
                  <p className="text-xs font-medium mb-2">{data.fullDate}</p>
                  <div className="space-y-1 text-xs">
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">고가:</span>
                      <span className="font-semibold text-red-500">{data.high.toLocaleString()}원</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">저가:</span>
                      <span className="font-semibold text-blue-500">{data.low.toLocaleString()}원</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">종가:</span>
                      <span className={`font-semibold ${isUp ? 'text-red-500' : 'text-blue-500'}`}>
                        {data.close.toLocaleString()}원
                      </span>
                    </div>
                  </div>
                </div>
              )
            }}
            cursor={{
              stroke: '#6b7280',
              strokeWidth: 1,
              strokeDasharray: '4 4',
            }}
          />

          {/* Horizontal crosshair line with price label */}
          {mousePrice !== null && (
            <ReferenceLine
              y={mousePrice}
              stroke="#3b82f6"
              strokeWidth={1.5}
              strokeDasharray="5 3"
              ifOverflow="extendDomain"
              label={{
                value: Math.round(mousePrice).toLocaleString(),
                position: 'left',
                fill: '#60a5fa',
                fontSize: 12,
                fontWeight: 'bold',
                offset: 5,
              }}
            />
          )}

          {/* Candlesticks */}
          <Bar
            dataKey="close"
            shape={(rawProps: unknown) => {
              const props = rawProps as CandleShapeProps
              const { x, width, payload, index } = props
              const yScale = (350 - 20) / (yMax - yMin)

              const getY = (price: number) => 10 + (yMax - price) * yScale

              const isUp = payload.close >= payload.open
              const color = isUp ? '#EA5455' : '#2196F3'

              const wickX = x + width / 2
              const highY = getY(payload.high)
              const lowY = getY(payload.low)
              const openY = getY(payload.open)
              const closeY = getY(payload.close)

              const bodyTop = Math.min(openY, closeY)
              const bodyHeight = Math.abs(closeY - openY) || 1

              return (
                <g key={`candle-${index}`}>
                  {/* Wick */}
                  <line
                    x1={wickX}
                    x2={wickX}
                    y1={highY}
                    y2={lowY}
                    stroke={color}
                    strokeWidth={1}
                  />
                  {/* Body */}
                  <rect
                    x={x + 1}
                    y={bodyTop}
                    width={Math.max(width - 2, 2)}
                    height={bodyHeight}
                    fill={isUp ? 'transparent' : color}
                    stroke={color}
                    strokeWidth={1}
                  />
                </g>
              )
            }}
          />

          {/* 평단가 선 (보유 종목만) */}
          {avgBuyPrice && avgBuyPrice >= yMin && avgBuyPrice <= yMax && (
            <ReferenceLine
              y={avgBuyPrice}
              stroke="#fbbf24"
              strokeWidth={2}
              strokeDasharray="5 5"
              label={{
                value: avgBuyPrice.toLocaleString(),
                position: 'right',
                fill: '#fbbf24',
                fontSize: 14,
                fontWeight: 'bold',
              }}
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>

      {/* 거래량 차트 */}
      <ResponsiveContainer width="100%" height={100}>
        <ComposedChart data={chartData} margin={{ top: 0, right: 80, left: 10, bottom: 0 }}>
          <XAxis dataKey="date" hide />
          <YAxis
            tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
            tickFormatter={(v) => (v / 1000000).toFixed(0) + 'M'}
            width={70}
          />
          <Bar dataKey="volume">
            {chartData.map((entry, index) => (
              <Cell
                key={`cell-${index}`}
                fill={entry.close >= entry.open ? '#EA5455' : '#2196F3'}
                fillOpacity={0.5}
              />
            ))}
          </Bar>
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
}

// 수급 차트 (투자자별 매매동향 - v10 스타일)
function InvestorTradingChart({ data }: { data: InvestorFlow[] }) {
  const [period, setPeriod] = useState<PeriodType>('1M')

  // Filter data by period
  const filteredData = useMemo(() => {
    const periodConfig = PERIODS.find(p => p.value === period)
    const days = periodConfig?.days ?? 30
    return data.slice(-days)
  }, [data, period])

  // Format data for chart
  const chartData = useMemo(() => {
    return filteredData.map(d => ({
      ...d,
      displayDate: d.date.substring(5).replace('-', '. ') + '.',
      foreign: d.foreign_net,
      inst: d.inst_net,
      retail: d.retail_net,
    }))
  }, [filteredData])

  // Calculate date range
  const dateRange = useMemo(() => {
    if (chartData.length === 0) return ''
    const first = chartData[0]?.date ?? ''
    const last = chartData[chartData.length - 1]?.date ?? ''
    const formatDate = (d: string) => d.replace(/-/g, '. ') + '.'
    return `${formatDate(first)} - ${formatDate(last)} 기준`
  }, [chartData])

  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        수급 데이터가 없습니다
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header with Period Selector */}
      <div className="flex items-center justify-between">
        <div className="text-xs text-muted-foreground">{dateRange}</div>
        <div className="flex gap-1 bg-muted/30 rounded-lg p-0.5">
          {PERIODS.map(p => (
            <Button
              key={p.value}
              variant={period === p.value ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setPeriod(p.value)}
              className="h-7 px-3 text-xs"
            >
              {p.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center justify-end gap-4">
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.foreign }} />
          <span className="text-xs text-muted-foreground">외국인</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.inst }} />
          <span className="text-xs text-muted-foreground">기관</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.retail }} />
          <span className="text-xs text-muted-foreground">개인</span>
        </div>
        <span className="text-xs text-muted-foreground">(주)</span>
      </div>

      {/* Line Chart */}
      <ResponsiveContainer width="100%" height={200}>
        <LineChart data={chartData} margin={{ top: 5, right: 50, left: 10, bottom: 5 }}>
          <CartesianGrid strokeDasharray="1 1" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="displayDate"
            tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
            tickLine={false}
            axisLine={{ stroke: 'hsl(var(--border))' }}
            interval="preserveStartEnd"
            minTickGap={50}
          />
          <YAxis
            tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
            tickLine={false}
            axisLine={false}
            tickFormatter={(v) => formatAxisValue(v)}
            orientation="right"
            width={55}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--popover))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
              fontSize: '12px',
            }}
            labelStyle={{ color: 'hsl(var(--foreground))', marginBottom: '4px' }}
            formatter={(value, name) => {
              const labels: Record<string, string> = {
                foreign: '외국인',
                inst: '기관',
                retail: '개인',
              }
              const numValue = Number(value) || 0
              const strName = String(name || '')
              const color = numValue >= 0 ? '#EA5455' : '#2196F3'
              return [
                <span key={strName} style={{ color }}>{formatVolume(numValue)}주</span>,
                labels[strName] || strName
              ]
            }}
          />
          <ReferenceLine y={0} stroke="hsl(var(--border))" strokeWidth={1} />
          <Line
            type="monotone"
            dataKey="foreign"
            stroke={COLORS.foreign}
            strokeWidth={1.5}
            dot={false}
            activeDot={{ r: 4, fill: COLORS.foreign }}
          />
          <Line
            type="monotone"
            dataKey="inst"
            stroke={COLORS.inst}
            strokeWidth={1.5}
            dot={false}
            activeDot={{ r: 4, fill: COLORS.inst }}
          />
          <Line
            type="monotone"
            dataKey="retail"
            stroke={COLORS.retail}
            strokeWidth={1.5}
            dot={false}
            activeDot={{ r: 4, fill: COLORS.retail }}
          />
        </LineChart>
      </ResponsiveContainer>

      {/* Data Table */}
      <div className="border-t overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b bg-muted/30">
              <th className="py-2.5 px-3 text-left font-medium text-muted-foreground">날짜</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">종가</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">전일대비</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">등락률</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">거래량</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">외국인</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">기관</th>
              <th className="py-2.5 px-3 text-right font-medium text-muted-foreground">개인</th>
            </tr>
          </thead>
          <tbody>
            {chartData.slice().reverse().slice(0, 10).map((row, idx) => (
              <tr key={idx} className="border-b last:border-b-0 hover:bg-muted/30">
                <td className="py-2.5 px-3">
                  {row.date.replace(/-/g, '. ') + '.'}
                </td>
                <td className="py-2.5 px-3 text-right">
                  {row.close_price > 0 ? row.close_price.toLocaleString() : '-'}
                </td>
                <td className={`py-2.5 px-3 text-right ${row.price_change >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                  {row.price_change !== 0 ? (
                    <>
                      {row.price_change > 0 ? '▲' : '▼'} {Math.abs(row.price_change).toLocaleString()}
                    </>
                  ) : '-'}
                </td>
                <td className={`py-2.5 px-3 text-right ${row.change_rate >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                  {row.change_rate !== 0 ? `${row.change_rate > 0 ? '+' : ''}${row.change_rate.toFixed(2)}%` : '-'}
                </td>
                <td className="py-2.5 px-3 text-right">
                  {row.volume > 0 ? formatVolumeShort(row.volume) + '주' : '-'}
                </td>
                <td className={`py-2.5 px-3 text-right ${row.foreign >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                  {row.foreign >= 0 ? '+' : ''}{formatVolume(row.foreign)}주
                </td>
                <td className={`py-2.5 px-3 text-right ${row.inst >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                  {row.inst >= 0 ? '+' : ''}{formatVolume(row.inst)}주
                </td>
                <td className={`py-2.5 px-3 text-right ${row.retail >= 0 ? 'text-red-500' : 'text-blue-500'}`}>
                  {row.retail >= 0 ? '+' : ''}{formatVolume(row.retail)}주
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export function ChartTab({ symbol, symbolName, avgBuyPrice }: ChartTabProps) {
  // 일봉 데이터 조회 (최근 1년 - 기간 선택 가능하므로)
  const { data: priceData = [] } = useQuery({
    queryKey: ['priceHistory', symbol],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setFullYear(startDate.getFullYear() - 1)
      return getPriceHistory(symbol, startDate.toISOString().slice(0, 10), endDate)
    },
    enabled: !!symbol,
  })

  // 수급 데이터 조회 (최근 1년 - 기간 선택 가능하므로)
  const { data: flowData = [] } = useQuery({
    queryKey: ['flowHistory', symbol],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setFullYear(startDate.getFullYear() - 1)
      return getFlowHistory(symbol, startDate.toISOString().slice(0, 10), endDate)
    },
    enabled: !!symbol,
  })

  return (
    <div className="space-y-6">
      {/* 일봉 차트 */}
      <Card>
        <CardHeader>
          <CardTitle>일봉 차트</CardTitle>
        </CardHeader>
        <CardContent>
          <PriceChart data={priceData} avgBuyPrice={avgBuyPrice} />
        </CardContent>
      </Card>

      {/* 수급 차트 */}
      <Card>
        <CardHeader>
          <CardTitle>투자자별 매매동향</CardTitle>
        </CardHeader>
        <CardContent>
          <InvestorTradingChart data={flowData} />
        </CardContent>
      </Card>
    </div>
  )
}
