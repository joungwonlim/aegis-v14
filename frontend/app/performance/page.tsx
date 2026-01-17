'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  getPerformance,
  getDailyPnL,
  getRiskMetrics,
  type AuditPeriod,
  type PerformanceReport,
  type DailyPnL,
  type RiskMetrics,
} from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  TrendingUp,
  TrendingDown,
  Activity,
  Target,
  Shield,
  BarChart3,
  AlertTriangle,
} from 'lucide-react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  ReferenceLine,
} from 'recharts'

const PERIODS: { value: AuditPeriod; label: string }[] = [
  { value: '1M', label: '1개월' },
  { value: '3M', label: '3개월' },
  { value: '6M', label: '6개월' },
  { value: '1Y', label: '1년' },
  { value: 'YTD', label: 'YTD' },
]

function formatPercent(value: number | undefined | null): string {
  if (value === undefined || value === null) return '-'
  return `${(value * 100).toFixed(2)}%`
}

function formatNumber(value: number | undefined | null, decimals = 2): string {
  if (value === undefined || value === null) return '-'
  return value.toFixed(decimals)
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

function MetricCard({
  title,
  value,
  subtitle,
  icon: Icon,
  trend,
  className,
}: {
  title: string
  value: string
  subtitle?: string
  icon: React.ComponentType<{ className?: string }>
  trend?: 'up' | 'down' | 'neutral'
  className?: string
}) {
  return (
    <Card className={className}>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{title}</p>
            <p
              className={`text-2xl font-bold ${
                trend === 'up'
                  ? 'text-green-600'
                  : trend === 'down'
                  ? 'text-red-600'
                  : ''
              }`}
            >
              {value}
            </p>
            {subtitle && (
              <p className="text-xs text-muted-foreground mt-1">{subtitle}</p>
            )}
          </div>
          <Icon
            className={`h-8 w-8 ${
              trend === 'up'
                ? 'text-green-500'
                : trend === 'down'
                ? 'text-red-500'
                : 'text-muted-foreground'
            }`}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function PerformanceSection({ report }: { report: PerformanceReport | null }) {
  if (!report) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        데이터가 없습니다
      </div>
    )
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="총 수익률"
        value={formatPercent(report.total_return)}
        icon={report.total_return >= 0 ? TrendingUp : TrendingDown}
        trend={report.total_return >= 0 ? 'up' : 'down'}
      />
      <MetricCard
        title="연환산 수익률"
        value={formatPercent(report.annual_return)}
        icon={BarChart3}
        trend={report.annual_return >= 0 ? 'up' : 'down'}
      />
      <MetricCard
        title="Alpha"
        value={formatPercent(report.alpha)}
        subtitle={`Beta: ${formatNumber(report.beta)}`}
        icon={Target}
        trend={report.alpha >= 0 ? 'up' : 'down'}
      />
      <MetricCard
        title="MDD"
        value={formatPercent(report.max_drawdown)}
        icon={AlertTriangle}
        trend="down"
      />
    </div>
  )
}

function RiskSection({ metrics }: { metrics: RiskMetrics | null }) {
  if (!metrics) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        리스크 데이터가 없습니다
      </div>
    )
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="VaR 95%"
        value={formatKRW(metrics.portfolio_var_95)}
        icon={Shield}
      />
      <MetricCard
        title="VaR 99%"
        value={formatKRW(metrics.portfolio_var_99)}
        icon={Shield}
      />
      <MetricCard
        title="집중도 (HHI)"
        value={formatNumber(metrics.concentration, 4)}
        icon={Activity}
      />
      <MetricCard
        title="시장 상관관계"
        value={formatNumber(metrics.market_correlation, 4)}
        icon={Activity}
      />
    </div>
  )
}

function TradingSection({ report }: { report: PerformanceReport | null }) {
  if (!report) {
    return null
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="승률"
        value={formatPercent(report.win_rate)}
        icon={Target}
        trend={report.win_rate >= 0.5 ? 'up' : 'down'}
      />
      <MetricCard
        title="총 거래"
        value={`${report.total_trades || 0}건`}
        icon={Activity}
      />
      <MetricCard
        title="평균 이익"
        value={formatKRW(report.avg_win)}
        icon={TrendingUp}
        trend="up"
      />
      <MetricCard
        title="평균 손실"
        value={formatKRW(Math.abs(report.avg_loss || 0))}
        icon={TrendingDown}
        trend="down"
      />
    </div>
  )
}

function RatioSection({ report }: { report: PerformanceReport | null }) {
  if (!report) {
    return null
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="Sharpe Ratio"
        value={formatNumber(report.sharpe_ratio)}
        icon={BarChart3}
        trend={report.sharpe_ratio >= 1 ? 'up' : report.sharpe_ratio >= 0 ? 'neutral' : 'down'}
      />
      <MetricCard
        title="Sortino Ratio"
        value={formatNumber(report.sortino_ratio)}
        icon={BarChart3}
        trend={report.sortino_ratio >= 1 ? 'up' : report.sortino_ratio >= 0 ? 'neutral' : 'down'}
      />
      <MetricCard
        title="Profit Factor"
        value={formatNumber(report.profit_factor)}
        icon={Target}
        trend={report.profit_factor >= 1 ? 'up' : 'down'}
      />
      <MetricCard
        title="변동성"
        value={formatPercent(report.volatility)}
        icon={Activity}
      />
    </div>
  )
}

function PnLChart({ data }: { data: DailyPnL[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        PnL 데이터가 없습니다
      </div>
    )
  }

  const chartData = data.map((d) => ({
    date: d.pnl_date.slice(5), // MM-DD
    cumReturn: (d.cumulative_return || 0) * 100,
    dailyReturn: (d.daily_return || 0) * 100,
  }))

  return (
    <ResponsiveContainer width="100%" height={300}>
      <AreaChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis dataKey="date" tick={{ fontSize: 12 }} />
        <YAxis
          tick={{ fontSize: 12 }}
          tickFormatter={(v) => `${v.toFixed(1)}%`}
        />
        <Tooltip
          formatter={(value) => [`${Number(value).toFixed(2)}%`, '수익률']}
          labelFormatter={(label) => `날짜: ${label}`}
        />
        <Area
          type="monotone"
          dataKey="cumReturn"
          stroke="#10b981"
          fill="#10b981"
          fillOpacity={0.3}
          name="누적 수익률"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}

function DailyReturnChart({ data }: { data: DailyPnL[] }) {
  if (!data || data.length === 0) {
    return null
  }

  const chartData = data.slice(-30).map((d) => ({
    date: d.pnl_date.slice(5),
    return: (d.daily_return || 0) * 100,
  }))

  return (
    <ResponsiveContainer width="100%" height={200}>
      <BarChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis dataKey="date" tick={{ fontSize: 10 }} />
        <YAxis tick={{ fontSize: 10 }} tickFormatter={(v) => `${v.toFixed(1)}%`} />
        <Tooltip formatter={(value) => [`${Number(value).toFixed(2)}%`, '일간 수익률']} />
        <ReferenceLine y={0} stroke="#888" />
        <Bar
          dataKey="return"
          fill="#10b981"
          radius={[2, 2, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  )
}

export default function PerformancePage() {
  const [period, setPeriod] = useState<AuditPeriod>('1M')

  const { data: performance, isLoading: loadingPerf } = useQuery({
    queryKey: ['performance', period],
    queryFn: () => getPerformance(period),
  })

  const { data: risk, isLoading: loadingRisk } = useQuery({
    queryKey: ['risk', period],
    queryFn: () => getRiskMetrics(period),
  })

  const { data: pnl, isLoading: loadingPnL } = useQuery({
    queryKey: ['dailyPnL', period],
    queryFn: () => {
      const endDate = new Date().toISOString().slice(0, 10)
      const startDate = new Date()
      switch (period) {
        case '1M':
          startDate.setMonth(startDate.getMonth() - 1)
          break
        case '3M':
          startDate.setMonth(startDate.getMonth() - 3)
          break
        case '6M':
          startDate.setMonth(startDate.getMonth() - 6)
          break
        case '1Y':
          startDate.setFullYear(startDate.getFullYear() - 1)
          break
        case 'YTD':
          startDate.setMonth(0, 1)
          break
      }
      return getDailyPnL(startDate.toISOString().slice(0, 10), endDate)
    },
  })

  const isLoading = loadingPerf || loadingRisk || loadingPnL

  return (
    <div className="flex-1 p-6 space-y-6 overflow-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Performance Dashboard</h1>
          <p className="text-muted-foreground">포트폴리오 성과 분석</p>
        </div>
        <Tabs value={period} onValueChange={(v) => setPeriod(v as AuditPeriod)}>
          <TabsList>
            {PERIODS.map((p) => (
              <TabsTrigger key={p.value} value={p.value}>
                {p.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-48">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
        </div>
      ) : (
        <>
          {/* 성과 지표 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                수익률 지표
              </CardTitle>
            </CardHeader>
            <CardContent>
              <PerformanceSection report={performance ?? null} />
            </CardContent>
          </Card>

          {/* 비율 지표 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BarChart3 className="h-5 w-5" />
                리스크 조정 수익률
              </CardTitle>
            </CardHeader>
            <CardContent>
              <RatioSection report={performance ?? null} />
            </CardContent>
          </Card>

          {/* 누적 수익률 차트 */}
          <Card>
            <CardHeader>
              <CardTitle>누적 수익률</CardTitle>
            </CardHeader>
            <CardContent>
              <PnLChart data={pnl || []} />
            </CardContent>
          </Card>

          {/* 일간 수익률 차트 */}
          <Card>
            <CardHeader>
              <CardTitle>일간 수익률 (최근 30일)</CardTitle>
            </CardHeader>
            <CardContent>
              <DailyReturnChart data={pnl || []} />
            </CardContent>
          </Card>

          {/* 트레이딩 지표 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Target className="h-5 w-5" />
                트레이딩 지표
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TradingSection report={performance ?? null} />
            </CardContent>
          </Card>

          {/* 리스크 지표 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="h-5 w-5" />
                리스크 지표
              </CardTitle>
            </CardHeader>
            <CardContent>
              <RiskSection metrics={risk ?? null} />
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
