'use client'

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  searchStocks,
  listStocks,
  getPriceHistory,
  getFlowHistory,
  type Stock,
  type DailyPrice,
  type InvestorFlow,
} from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
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
  Search,
  Database,
  TrendingUp,
  TrendingDown,
  Building2,
  Users,
  Briefcase,
} from 'lucide-react'
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
  Legend,
  ReferenceLine,
} from 'recharts'

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

function formatPercent(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  const sign = value >= 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

function PriceChart({ data }: { data: DailyPrice[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        가격 데이터가 없습니다
      </div>
    )
  }

  const chartData = data.slice(-60).map((d) => ({
    date: d.trade_date.slice(5),
    close: d.close_price,
    volume: d.volume,
  }))

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis dataKey="date" tick={{ fontSize: 10 }} />
        <YAxis
          tick={{ fontSize: 10 }}
          tickFormatter={(v) => formatKRW(v)}
          domain={['auto', 'auto']}
        />
        <Tooltip
          formatter={(value) => [formatKRW(Number(value)), '종가']}
          labelFormatter={(label) => `날짜: ${label}`}
        />
        <Line
          type="monotone"
          dataKey="close"
          stroke="#3b82f6"
          strokeWidth={2}
          dot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  )
}

function FlowChart({ data }: { data: InvestorFlow[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        수급 데이터가 없습니다
      </div>
    )
  }

  const chartData = data.slice(-30).map((d) => ({
    date: d.trade_date.slice(5),
    foreign: d.foreign_net / 100000000, // 억 단위
    inst: d.inst_net / 100000000,
    retail: d.retail_net / 100000000,
  }))

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis dataKey="date" tick={{ fontSize: 10 }} />
        <YAxis tick={{ fontSize: 10 }} tickFormatter={(v) => `${v.toFixed(0)}억`} />
        <Tooltip formatter={(value) => [`${Number(value).toFixed(1)}억`, '']} />
        <Legend />
        <ReferenceLine y={0} stroke="#888" />
        <Bar dataKey="foreign" name="외국인" fill="#3b82f6" radius={[2, 2, 0, 0]} />
        <Bar dataKey="inst" name="기관" fill="#f59e0b" radius={[2, 2, 0, 0]} />
        <Bar dataKey="retail" name="개인" fill="#6b7280" radius={[2, 2, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}

function StockDetail({ stock }: { stock: Stock }) {
  const { data: priceData } = useQuery({
    queryKey: ['priceHistory', stock.stock_code],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setMonth(startDate.getMonth() - 3)
      return getPriceHistory(stock.stock_code, startDate.toISOString().slice(0, 10), endDate)
    },
  })

  const { data: flowData } = useQuery({
    queryKey: ['flowHistory', stock.stock_code],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      startDate.setMonth(startDate.getMonth() - 1)
      return getFlowHistory(stock.stock_code, startDate.toISOString().slice(0, 10), endDate)
    },
  })

  const latestPrice = priceData?.[priceData.length - 1]
  const latestFlow = flowData?.[flowData.length - 1]

  // Calculate cumulative flows for the period
  const cumFlow = useMemo(() => {
    if (!flowData || flowData.length === 0) return null
    return {
      foreign: flowData.reduce((sum, d) => sum + d.foreign_net, 0),
      inst: flowData.reduce((sum, d) => sum + d.inst_net, 0),
      retail: flowData.reduce((sum, d) => sum + d.retail_net, 0),
    }
  }, [flowData])

  return (
    <div className="space-y-6">
      {/* 종목 정보 헤더 */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-xl font-bold">{stock.stock_name}</h2>
            <Badge variant="secondary">{stock.stock_code}</Badge>
            <Badge variant="outline">{stock.market}</Badge>
          </div>
          {stock.sector && (
            <p className="text-sm text-muted-foreground mt-1">{stock.sector}</p>
          )}
        </div>
        {latestPrice && (
          <div className="text-right">
            <p className="text-2xl font-bold">{formatKRW(latestPrice.close_price)}</p>
            <p
              className={`text-sm ${
                latestPrice.change_rate >= 0 ? 'text-green-600' : 'text-red-600'
              }`}
            >
              {formatPercent(latestPrice.change_rate)}
            </p>
          </div>
        )}
      </div>

      {/* 차트 탭 */}
      <Tabs defaultValue="price" className="w-full">
        <TabsList>
          <TabsTrigger value="price">일봉 차트</TabsTrigger>
          <TabsTrigger value="flow">수급 차트</TabsTrigger>
        </TabsList>

        <TabsContent value="price" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>일봉 차트 (최근 60일)</CardTitle>
            </CardHeader>
            <CardContent>
              <PriceChart data={priceData || []} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="flow" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>투자자별 순매수 (최근 30일)</CardTitle>
            </CardHeader>
            <CardContent>
              <FlowChart data={flowData || []} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 수급 요약 */}
      {cumFlow && (
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Building2 className="h-5 w-5 text-blue-500" />
                <span className="text-sm text-muted-foreground">외국인 누적</span>
              </div>
              <p
                className={`text-xl font-bold mt-2 ${
                  cumFlow.foreign >= 0 ? 'text-green-600' : 'text-red-600'
                }`}
              >
                {formatKRW(cumFlow.foreign)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Briefcase className="h-5 w-5 text-amber-500" />
                <span className="text-sm text-muted-foreground">기관 누적</span>
              </div>
              <p
                className={`text-xl font-bold mt-2 ${
                  cumFlow.inst >= 0 ? 'text-green-600' : 'text-red-600'
                }`}
              >
                {formatKRW(cumFlow.inst)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Users className="h-5 w-5 text-gray-500" />
                <span className="text-sm text-muted-foreground">개인 누적</span>
              </div>
              <p
                className={`text-xl font-bold mt-2 ${
                  cumFlow.retail >= 0 ? 'text-green-600' : 'text-red-600'
                }`}
              >
                {formatKRW(cumFlow.retail)}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 최근 가격 정보 */}
      {latestPrice && (
        <Card>
          <CardHeader>
            <CardTitle>최근 거래 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">시가</p>
                <p className="font-medium">{formatKRW(latestPrice.open_price)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">고가</p>
                <p className="font-medium text-red-600">{formatKRW(latestPrice.high_price)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">저가</p>
                <p className="font-medium text-blue-600">{formatKRW(latestPrice.low_price)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">종가</p>
                <p className="font-medium">{formatKRW(latestPrice.close_price)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">거래량</p>
                <p className="font-medium">{formatKRW(latestPrice.volume)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default function StocksPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedStock, setSelectedStock] = useState<Stock | null>(null)

  const { data: stocks, isLoading } = useQuery({
    queryKey: ['stocks', searchQuery],
    queryFn: () => {
      if (searchQuery.length >= 2) {
        return searchStocks(searchQuery)
      }
      return listStocks({ limit: 50 })
    },
    enabled: searchQuery.length === 0 || searchQuery.length >= 2,
  })

  return (
    <div className="flex-1 p-6 space-y-6 overflow-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Stocks Universe</h1>
          <p className="text-muted-foreground">종목 검색 및 데이터 조회</p>
        </div>
        <Badge variant="outline" className="gap-1">
          <Database className="h-3 w-3" />
          {stocks?.length || 0} 종목
        </Badge>
      </div>

      {/* 검색 */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="종목명 또는 종목코드 검색..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 종목 리스트 */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle>종목 리스트</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="flex items-center justify-center h-48">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
              </div>
            ) : (
              <div className="max-h-[600px] overflow-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>종목</TableHead>
                      <TableHead>시장</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {stocks?.map((stock) => (
                      <TableRow
                        key={stock.stock_code}
                        className={`cursor-pointer ${
                          selectedStock?.stock_code === stock.stock_code
                            ? 'bg-accent'
                            : ''
                        }`}
                        onClick={() => setSelectedStock(stock)}
                      >
                        <TableCell>
                          <div>
                            <p className="font-medium">{stock.stock_name}</p>
                            <p className="text-xs text-muted-foreground">
                              {stock.stock_code}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {stock.market}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* 종목 상세 */}
        <Card className="lg:col-span-2">
          <CardContent className="pt-6">
            {selectedStock ? (
              <StockDetail stock={selectedStock} />
            ) : (
              <div className="flex flex-col items-center justify-center h-[500px] text-muted-foreground">
                <Database className="h-12 w-12 mb-4" />
                <p>종목을 선택하세요</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
