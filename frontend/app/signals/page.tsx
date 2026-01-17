'use client'

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  getLatestSignalSnapshot,
  getBuySignals,
  getSellSignals,
  getFactors,
  type Signal,
  type FactorScore,
} from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  TrendingUp,
  TrendingDown,
  Activity,
  Zap,
  Filter,
  BarChart3,
  Search,
} from 'lucide-react'
import {
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  Radar,
  ResponsiveContainer,
  Legend,
} from 'recharts'

type FactorType = 'momentum' | 'technical' | 'value' | 'quality' | 'flow' | 'event'

const FACTORS: { key: FactorType; label: string; color: string }[] = [
  { key: 'momentum', label: 'Momentum', color: '#3b82f6' },
  { key: 'technical', label: 'Technical', color: '#10b981' },
  { key: 'value', label: 'Value', color: '#f59e0b' },
  { key: 'quality', label: 'Quality', color: '#8b5cf6' },
  { key: 'flow', label: 'Flow', color: '#ec4899' },
  { key: 'event', label: 'Event', color: '#6366f1' },
]

function formatScore(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  return value.toFixed(3)
}

function formatPercent(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  const sign = value >= 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

function formatKRW(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  return value.toLocaleString()
}

function ScoreBadge({ score, label }: { score: number; label: string }) {
  const getColor = (s: number) => {
    if (s >= 0.7) return 'bg-green-100 text-green-800 border-green-300'
    if (s >= 0.3) return 'bg-blue-100 text-blue-800 border-blue-300'
    if (s >= -0.3) return 'bg-gray-100 text-gray-800 border-gray-300'
    if (s >= -0.7) return 'bg-orange-100 text-orange-800 border-orange-300'
    return 'bg-red-100 text-red-800 border-red-300'
  }

  return (
    <Badge variant="outline" className={`${getColor(score)} text-xs`}>
      {label}: {score.toFixed(2)}
    </Badge>
  )
}

function FactorRadarChart({ signal }: { signal: Signal }) {
  const data = FACTORS.map((f) => ({
    factor: f.label,
    score: ((signal[f.key] || 0) + 1) / 2 * 100, // -1~1 -> 0~100
  }))

  return (
    <ResponsiveContainer width="100%" height={250}>
      <RadarChart data={data}>
        <PolarGrid />
        <PolarAngleAxis dataKey="factor" tick={{ fontSize: 11 }} />
        <PolarRadiusAxis domain={[0, 100]} tick={{ fontSize: 10 }} />
        <Radar
          name="Factor Score"
          dataKey="score"
          stroke="#3b82f6"
          fill="#3b82f6"
          fillOpacity={0.3}
        />
      </RadarChart>
    </ResponsiveContainer>
  )
}

function SignalTable({
  signals,
  type,
  onSelect,
  selectedSymbol,
  filterFactor,
}: {
  signals: Signal[]
  type: 'BUY' | 'SELL'
  onSelect: (signal: Signal) => void
  selectedSymbol: string | null
  filterFactor: FactorType | 'all'
}) {
  const sortedSignals = useMemo(() => {
    let filtered = [...signals]
    if (filterFactor !== 'all') {
      filtered = filtered.filter((s) => (s[filterFactor] || 0) > 0.3)
    }
    return filtered.sort((a, b) => {
      if (filterFactor === 'all') {
        return (b.total_score || 0) - (a.total_score || 0)
      }
      return (b[filterFactor] || 0) - (a[filterFactor] || 0)
    })
  }, [signals, filterFactor])

  if (sortedSignals.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        {type === 'BUY' ? '매수' : '매도'} 시그널이 없습니다
      </div>
    )
  }

  return (
    <div className="max-h-[500px] overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[140px]">종목</TableHead>
            <TableHead className="text-right">Total</TableHead>
            <TableHead className="text-right">Mom</TableHead>
            <TableHead className="text-right">Tech</TableHead>
            <TableHead className="text-right">Val</TableHead>
            <TableHead className="text-right">Qual</TableHead>
            <TableHead className="text-right">Flow</TableHead>
            <TableHead className="text-right">Event</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedSignals.map((signal) => (
            <TableRow
              key={signal.stock_code}
              className={`cursor-pointer ${
                selectedSymbol === signal.stock_code ? 'bg-accent' : ''
              }`}
              onClick={() => onSelect(signal)}
            >
              <TableCell>
                <div>
                  <p className="font-medium truncate max-w-[120px]">
                    {signal.stock_name || signal.stock_code}
                  </p>
                  <p className="text-xs text-muted-foreground">{signal.stock_code}</p>
                </div>
              </TableCell>
              <TableCell className="text-right font-medium">
                <span
                  className={
                    signal.total_score >= 0.5
                      ? 'text-green-600'
                      : signal.total_score <= -0.5
                      ? 'text-red-600'
                      : ''
                  }
                >
                  {formatScore(signal.total_score)}
                </span>
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.momentum)}
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.technical)}
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.value)}
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.quality)}
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.flow)}
              </TableCell>
              <TableCell className="text-right text-xs">
                {formatScore(signal.event)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function SignalDetail({ signal }: { signal: Signal }) {
  return (
    <div className="space-y-6">
      {/* 종목 헤더 */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-xl font-bold">
              {signal.stock_name || signal.stock_code}
            </h2>
            <Badge variant="secondary">{signal.stock_code}</Badge>
            <Badge
              variant={signal.signal_type === 'BUY' ? 'default' : 'destructive'}
              className="gap-1"
            >
              {signal.signal_type === 'BUY' ? (
                <TrendingUp className="h-3 w-3" />
              ) : (
                <TrendingDown className="h-3 w-3" />
              )}
              {signal.signal_type}
            </Badge>
          </div>
        </div>
        <div className="text-right">
          <p className="text-sm text-muted-foreground">Total Score</p>
          <p
            className={`text-2xl font-bold ${
              signal.total_score >= 0.5
                ? 'text-green-600'
                : signal.total_score <= -0.5
                ? 'text-red-600'
                : ''
            }`}
          >
            {formatScore(signal.total_score)}
          </p>
        </div>
      </div>

      {/* 레이더 차트 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Factor Profile</CardTitle>
        </CardHeader>
        <CardContent>
          <FactorRadarChart signal={signal} />
        </CardContent>
      </Card>

      {/* 팩터별 점수 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Factor Scores</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {FACTORS.map((factor) => (
              <div
                key={factor.key}
                className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
              >
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: factor.color }}
                  />
                  <span className="text-sm">{factor.label}</span>
                </div>
                <span
                  className={`font-medium ${
                    (signal[factor.key] || 0) >= 0.5
                      ? 'text-green-600'
                      : (signal[factor.key] || 0) <= -0.5
                      ? 'text-red-600'
                      : ''
                  }`}
                >
                  {formatScore(signal[factor.key])}
                </span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 가격 정보 */}
      {signal.current_price && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Price Info</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <div>
                <p className="text-sm text-muted-foreground">현재가</p>
                <p className="text-xl font-bold">{formatKRW(signal.current_price)}</p>
              </div>
              {signal.change_rate !== undefined && (
                <div>
                  <p className="text-sm text-muted-foreground">등락률</p>
                  <p
                    className={`text-xl font-bold ${
                      signal.change_rate >= 0 ? 'text-green-600' : 'text-red-600'
                    }`}
                  >
                    {formatPercent(signal.change_rate)}
                  </p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default function SignalsPage() {
  const [selectedSignal, setSelectedSignal] = useState<Signal | null>(null)
  const [filterFactor, setFilterFactor] = useState<FactorType | 'all'>('all')
  const [searchQuery, setSearchQuery] = useState('')

  const { data: snapshot, isLoading: loadingSnapshot } = useQuery({
    queryKey: ['signalSnapshot'],
    queryFn: getLatestSignalSnapshot,
  })

  const { data: buySignals, isLoading: loadingBuy } = useQuery({
    queryKey: ['buySignals', snapshot?.id],
    queryFn: () => (snapshot ? getBuySignals(snapshot.id) : Promise.resolve([])),
    enabled: !!snapshot?.id,
  })

  const { data: sellSignals, isLoading: loadingSell } = useQuery({
    queryKey: ['sellSignals', snapshot?.id],
    queryFn: () => (snapshot ? getSellSignals(snapshot.id) : Promise.resolve([])),
    enabled: !!snapshot?.id,
  })

  const filteredBuySignals = useMemo(() => {
    if (!buySignals) return []
    if (!searchQuery) return buySignals
    const q = searchQuery.toLowerCase()
    return buySignals.filter(
      (s) =>
        s.stock_code.toLowerCase().includes(q) ||
        (s.stock_name && s.stock_name.toLowerCase().includes(q))
    )
  }, [buySignals, searchQuery])

  const filteredSellSignals = useMemo(() => {
    if (!sellSignals) return []
    if (!searchQuery) return sellSignals
    const q = searchQuery.toLowerCase()
    return sellSignals.filter(
      (s) =>
        s.stock_code.toLowerCase().includes(q) ||
        (s.stock_name && s.stock_name.toLowerCase().includes(q))
    )
  }, [sellSignals, searchQuery])

  const isLoading = loadingSnapshot || loadingBuy || loadingSell

  return (
    <div className="flex-1 p-6 space-y-6 overflow-auto">
      {/* 헤더 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Signals Ranking</h1>
          <p className="text-muted-foreground">6팩터 시그널 랭킹</p>
        </div>
        {snapshot && (
          <Badge variant="outline" className="gap-1">
            <Activity className="h-3 w-3" />
            {snapshot.calc_date} 기준
          </Badge>
        )}
      </div>

      {/* 통계 카드 */}
      {snapshot && (
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Zap className="h-5 w-5 text-primary" />
                <span className="text-sm text-muted-foreground">전체 시그널</span>
              </div>
              <p className="text-2xl font-bold mt-2">{snapshot.total_count}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5 text-green-500" />
                <span className="text-sm text-muted-foreground">매수 시그널</span>
              </div>
              <p className="text-2xl font-bold mt-2 text-green-600">
                {snapshot.buy_count}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <TrendingDown className="h-5 w-5 text-red-500" />
                <span className="text-sm text-muted-foreground">매도 시그널</span>
              </div>
              <p className="text-2xl font-bold mt-2 text-red-600">
                {snapshot.sell_count}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 필터 */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="종목 검색..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select
            value={filterFactor}
            onValueChange={(v) => setFilterFactor(v as FactorType | 'all')}
          >
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="팩터 필터" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">전체</SelectItem>
              {FACTORS.map((f) => (
                <SelectItem key={f.key} value={f.key}>
                  {f.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-48">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
        </div>
      ) : !snapshot ? (
        <div className="flex flex-col items-center justify-center h-48 text-muted-foreground">
          <BarChart3 className="h-12 w-12 mb-4" />
          <p>시그널 데이터가 없습니다</p>
          <p className="text-sm">먼저 시그널을 생성해주세요</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* 시그널 테이블 */}
          <div className="lg:col-span-2">
            <Tabs defaultValue="buy" className="w-full">
              <TabsList className="w-full justify-start">
                <TabsTrigger value="buy" className="gap-1">
                  <TrendingUp className="h-4 w-4" />
                  매수 ({filteredBuySignals.length})
                </TabsTrigger>
                <TabsTrigger value="sell" className="gap-1">
                  <TrendingDown className="h-4 w-4" />
                  매도 ({filteredSellSignals.length})
                </TabsTrigger>
              </TabsList>

              <TabsContent value="buy" className="mt-4">
                <Card>
                  <CardContent className="p-0">
                    <SignalTable
                      signals={filteredBuySignals}
                      type="BUY"
                      onSelect={setSelectedSignal}
                      selectedSymbol={selectedSignal?.stock_code || null}
                      filterFactor={filterFactor}
                    />
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="sell" className="mt-4">
                <Card>
                  <CardContent className="p-0">
                    <SignalTable
                      signals={filteredSellSignals}
                      type="SELL"
                      onSelect={setSelectedSignal}
                      selectedSymbol={selectedSignal?.stock_code || null}
                      filterFactor={filterFactor}
                    />
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>

          {/* 상세 정보 */}
          <Card className="lg:col-span-1">
            <CardContent className="pt-6">
              {selectedSignal ? (
                <SignalDetail signal={selectedSignal} />
              ) : (
                <div className="flex flex-col items-center justify-center h-[400px] text-muted-foreground">
                  <BarChart3 className="h-12 w-12 mb-4" />
                  <p>시그널을 선택하세요</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
