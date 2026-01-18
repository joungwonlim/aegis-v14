'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getStockRanking, type RankingCategory, type RankingStock } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { TrendingUp, TrendingDown, Activity, DollarSign, Users, Building } from 'lucide-react'
import { StockSymbol } from '@/components/stock-symbol'
import { cn } from '@/lib/utils'

const RANKING_CATEGORIES = [
  { key: 'volume' as RankingCategory, label: '거래량 상위', icon: Activity },
  { key: 'trading_value' as RankingCategory, label: '거래대금 상위', icon: DollarSign },
  { key: 'gainers' as RankingCategory, label: '상승률 상위', icon: TrendingUp },
  { key: 'losers' as RankingCategory, label: '하락률 상위', icon: TrendingDown },
  { key: 'foreign_net_buy' as RankingCategory, label: '외국인 순매수', icon: Users },
  { key: 'inst_net_buy' as RankingCategory, label: '기관 순매수', icon: Building },
]

export function StockRankings() {
  const [selectedCategory, setSelectedCategory] = useState<RankingCategory>('volume')

  const { data, isLoading } = useQuery({
    queryKey: ['stock-ranking', selectedCategory],
    queryFn: () => getStockRanking(selectedCategory, 10),
    refetchInterval: 60000, // 1분마다 갱신
  })

  const formatNumber = (value: number | undefined, decimals = 0) => {
    if (value === undefined || value === null) return '-'
    return value.toLocaleString('ko-KR', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
  }

  const formatPercent = (value: number | undefined) => {
    if (value === undefined || value === null) return <span className="text-muted-foreground">-</span>
    const color = value >= 0 ? '#EA5455' : '#2196F3'
    const sign = value > 0 ? '+' : ''
    return <span style={{ color }}>{sign}{value.toFixed(2)}%</span>
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    return date.toLocaleString('ko-KR', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const renderValue = (stock: RankingStock, category: RankingCategory) => {
    switch (category) {
      case 'volume':
        return formatNumber(stock.volume)
      case 'trading_value':
        return formatNumber(stock.trading_value)
      case 'gainers':
      case 'losers':
        return formatPercent(stock.change_rate)
      case 'foreign_net_buy':
        return formatNumber(stock.foreign_net_value)
      case 'inst_net_buy':
        return formatNumber(stock.inst_net_value)
      default:
        return '-'
    }
  }

  const getValueLabel = (category: RankingCategory) => {
    switch (category) {
      case 'volume':
        return '거래량'
      case 'trading_value':
        return '거래대금'
      case 'gainers':
      case 'losers':
        return '등락률'
      case 'foreign_net_buy':
        return '외국인 순매수'
      case 'inst_net_buy':
        return '기관 순매수'
      default:
        return '값'
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-primary" />
          시장 순위
        </CardTitle>
        <CardDescription>
          실시간 시장 순위 데이터
          {data?.updated_at && (
            <span className="ml-2 text-xs">
              마지막 갱신: {formatTimestamp(data.updated_at)}
            </span>
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={selectedCategory} onValueChange={(v) => setSelectedCategory(v as RankingCategory)}>
          <TabsList className="grid w-full grid-cols-6">
            {RANKING_CATEGORIES.map((cat) => {
              const Icon = cat.icon
              return (
                <TabsTrigger key={cat.key} value={cat.key} className="text-xs">
                  <Icon className="h-3 w-3 mr-1" />
                  {cat.label}
                </TabsTrigger>
              )
            })}
          </TabsList>

          {RANKING_CATEGORIES.map((cat) => (
            <TabsContent key={cat.key} value={cat.key} className="mt-4">
              {isLoading ? (
                <div className="text-center py-8 text-muted-foreground">로딩 중...</div>
              ) : !data || !data.stocks || data.stocks.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">데이터가 없습니다</div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12 text-center">순위</TableHead>
                      <TableHead>종목명</TableHead>
                      <TableHead className="text-right">현재가</TableHead>
                      <TableHead className="text-right">{getValueLabel(cat.key)}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.stocks.map((stock) => (
                      <TableRow key={stock.stock_code} className="hover:bg-accent cursor-pointer">
                        <TableCell className="text-center font-medium">
                          <Badge
                            variant={stock.rank <= 3 ? 'default' : 'outline'}
                            className={cn(
                              stock.rank === 1 && 'bg-yellow-500 hover:bg-yellow-600',
                              stock.rank === 2 && 'bg-gray-400 hover:bg-gray-500',
                              stock.rank === 3 && 'bg-orange-600 hover:bg-orange-700'
                            )}
                          >
                            {stock.rank}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <StockSymbol
                            symbol={stock.stock_code}
                            symbolName={stock.stock_name}
                            market={stock.market}
                            size="sm"
                          />
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatNumber(stock.current_price)}
                        </TableCell>
                        <TableCell className="text-right font-mono font-medium">
                          {renderValue(stock, cat.key)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  )
}
