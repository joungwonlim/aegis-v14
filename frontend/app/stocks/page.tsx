'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listStocks, getStockRanking, type Stock, type ListStocksResponse, type RankingCategory, type MarketFilter } from '@/lib/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { RefreshCw, Search, Loader2, Database, ChevronLeft, ChevronRight, Activity, DollarSign, TrendingUp, TrendingDown, Users, Building, Zap, Award, BarChart3 } from 'lucide-react'
import { StockSymbol } from '@/components/stock-symbol'
import { StockDetailSheet, useStockDetail } from '@/components/stock-detail-sheet'
import { useHoldings } from '@/hooks/useRuntimeData'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

const RANKING_CATEGORIES = [
  { key: 'all' as const, label: '전체', icon: Database },
  { key: 'volume' as RankingCategory, label: '거래량 상위', icon: Activity },
  { key: 'volume_surge' as RankingCategory, label: '거래량 급증', icon: Zap },
  { key: 'trading_value' as RankingCategory, label: '거래대금 상위', icon: DollarSign },
  { key: 'gainers' as RankingCategory, label: '상승', icon: TrendingUp },
  { key: 'losers' as RankingCategory, label: '하락', icon: TrendingDown },
  { key: 'foreign_net_buy' as RankingCategory, label: '외국인 순매수', icon: Users },
  { key: 'inst_net_buy' as RankingCategory, label: '기관 순매수', icon: Building },
  { key: 'high_52week' as RankingCategory, label: '52주 최고', icon: Award },
  { key: 'market_cap' as RankingCategory, label: '시가총액', icon: BarChart3 },
]

export default function StocksPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [page, setPage] = useState(1)
  const [selectedCategory, setSelectedCategory] = useState<'all' | RankingCategory>('all')
  const [selectedMarket, setSelectedMarket] = useState<MarketFilter>('ALL')
  const limit = 100

  // 데이터 조회 (전체 또는 순위)
  const { data: stocksData, isLoading: isLoadingStocks, refetch: refetchStocks } = useQuery({
    queryKey: ['stocks', searchQuery, page, selectedMarket],
    queryFn: async () => {
      return listStocks({
        search: searchQuery.trim() || undefined,
        page,
        limit,
        market: selectedMarket !== 'ALL' ? selectedMarket : undefined,
      })
    },
    enabled: selectedCategory === 'all',
  })

  const { data: rankingData, isLoading: isLoadingRanking, refetch: refetchRanking } = useQuery({
    queryKey: ['stock-ranking', selectedCategory, selectedMarket],
    queryFn: async () => {
      if (selectedCategory === 'all') return null
      return getStockRanking(selectedCategory, 100, selectedMarket)
    },
    enabled: selectedCategory !== 'all',
  })

  // 표시할 데이터 결정
  const isLoading = selectedCategory === 'all' ? isLoadingStocks : isLoadingRanking
  const stocks = selectedCategory === 'all'
    ? (stocksData?.stocks || [])
    : (rankingData?.stocks.map(s => ({
        stock_code: s.stock_code,
        stock_name: s.stock_name,
        market: s.market,
        current_price: s.current_price,
        change_rate: s.change_rate,
        sector: undefined,
      })) || [])
  const pagination = selectedCategory === 'all' ? stocksData?.pagination : undefined

  const { data: holdings = [] } = useHoldings()

  // StockDetailSheet
  const { selectedStock, isOpen: isStockDetailOpen, openStockDetail, handleOpenChange } = useStockDetail()

  const handleRefresh = () => {
    if (selectedCategory === 'all') {
      refetchStocks()
    } else {
      refetchRanking()
    }
    toast.success('새로고침 완료')
  }

  const handleCategoryChange = (category: 'all' | RankingCategory) => {
    setSelectedCategory(category)
    setPage(1)
    setSearchQuery('')
  }

  // 종목 클릭
  const handleStockClick = (stock: Stock) => {
    openStockDetail({
      symbol: stock.stock_code,
      symbolName: stock.stock_name,
    })
  }

  const formatNumber = (value: number | undefined | null, decimals = 0) => {
    if (value === undefined || value === null) return '-'
    return value.toLocaleString('ko-KR', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
  }

  const formatPercent = (value: number | undefined | null) => {
    if (value === undefined || value === null) return <span className="text-muted-foreground">-</span>
    const color = value >= 0 ? '#EA5455' : '#2196F3'
    const sign = value > 0 ? '+' : ''
    return <span style={{ color }}>{sign}{value.toFixed(2)}%</span>
  }

  // 페이지 이동
  const handlePrevPage = () => {
    if (page > 1) {
      setPage(page - 1)
    }
  }

  const handleNextPage = () => {
    if (pagination && page < pagination.total_pages) {
      setPage(page + 1)
    }
  }

  const handlePageClick = (pageNum: number) => {
    setPage(pageNum)
  }

  // 페이지 번호 목록 생성 (최대 7개 표시)
  const getPageNumbers = () => {
    if (!pagination) return []
    const { current_page, total_pages } = pagination
    const pages: number[] = []
    const maxVisible = 7

    if (total_pages <= maxVisible) {
      // 전체 페이지가 7개 이하면 모두 표시
      for (let i = 1; i <= total_pages; i++) {
        pages.push(i)
      }
    } else {
      // 현재 페이지 기준으로 앞뒤 3개씩
      let start = Math.max(1, current_page - 3)
      let end = Math.min(total_pages, current_page + 3)

      // 앞쪽에 공간이 남으면 뒤로 채우기
      if (end - start < maxVisible - 1) {
        if (start === 1) {
          end = Math.min(total_pages, start + maxVisible - 1)
        } else {
          start = Math.max(1, end - maxVisible + 1)
        }
      }

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }
    }

    return pages
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Database className="h-8 w-8 text-primary" />
            Stocks Universe
          </h1>
          <p className="text-muted-foreground">전체 종목 조회 및 검색</p>
        </div>
        {pagination && (
          <Badge variant="outline" className="gap-1">
            <Database className="h-3 w-3" />
            {pagination.total_count.toLocaleString()} 종목
          </Badge>
        )}
      </div>

      {/* Market & Category Filters - Naver Style */}
      <div className="flex gap-2 items-center flex-wrap">
        {/* Market Filters */}
        {(['ALL', 'KOSPI', 'KOSDAQ'] as MarketFilter[]).map((market) => {
          const isSelected = selectedMarket === market
          const label = market === 'ALL' ? '전체' : market === 'KOSPI' ? '코스피' : '코스닥'
          return (
            <Button
              key={market}
              variant="outline"
              size="default"
              onClick={() => setSelectedMarket(market)}
              className={cn(
                'rounded-md border transition-all font-medium',
                isSelected
                  ? 'bg-black text-white border-black hover:bg-black/90 hover:text-white'
                  : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              )}
            >
              {label}
            </Button>
          )
        })}

        {/* Category Filters */}
        {RANKING_CATEGORIES.filter(cat => cat.key !== 'all').map((cat) => {
          const isSelected = selectedCategory === cat.key
          return (
            <Button
              key={cat.key}
              variant="outline"
              size="default"
              onClick={() => handleCategoryChange(cat.key)}
              className={cn(
                'rounded-md border transition-all font-medium',
                isSelected
                  ? 'bg-black text-white border-black hover:bg-black/90 hover:text-white'
                  : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              )}
            >
              {cat.label}
            </Button>
          )
        })}
      </div>

      {/* Search & Actions */}
      {selectedCategory === 'all' && (
        <div className="flex gap-2">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="종목명 또는 종목코드 검색..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value)
              setPage(1) // 검색 시 첫 페이지로
            }}
            className="pl-10"
          />
        </div>
        <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
          새로고침
        </Button>
      </div>
      )}

      {/* Stocks Table */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <div>
              <CardTitle>
                {RANKING_CATEGORIES.find(c => c.key === selectedCategory)?.label || '종목 목록'}
              </CardTitle>
              <CardDescription>
                {selectedCategory === 'all' ? (
                  searchQuery && pagination
                    ? `검색 결과: "${searchQuery}" - ${pagination.total_count.toLocaleString()}개 (페이지 ${pagination.current_page} / ${pagination.total_pages})`
                    : searchQuery
                    ? `검색 결과: "${searchQuery}"`
                    : pagination
                    ? `페이지 ${pagination.current_page} / ${pagination.total_pages}`
                    : '전체 종목'
                ) : (
                  `${stocks.length}개 종목`
                )}
              </CardDescription>
            </div>
            {selectedCategory !== 'all' && rankingData?.updated_at && (
              <Badge variant="outline" className="text-xs">
                마지막 갱신: {new Date(rankingData.updated_at).toLocaleString('ko-KR', {
                  month: 'short',
                  day: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit'
                })}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">순위</TableHead>
                    <TableHead>종목명</TableHead>
                    <TableHead className="text-right">현재가</TableHead>
                    <TableHead className="text-right">전일대비</TableHead>
                    {selectedCategory === 'all' && <TableHead>업종</TableHead>}
                    {selectedCategory === 'volume' && <TableHead className="text-right">거래량</TableHead>}
                    {selectedCategory === 'volume_surge' && <TableHead className="text-right">거래량 증가율</TableHead>}
                    {selectedCategory === 'trading_value' && <TableHead className="text-right">거래대금</TableHead>}
                    {selectedCategory === 'high_52week' && <TableHead className="text-right">52주 최고가</TableHead>}
                    {selectedCategory === 'market_cap' && <TableHead className="text-right">시가총액</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {stocks.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                        {searchQuery ? '검색 결과가 없습니다' : '종목이 없습니다'}
                      </TableCell>
                    </TableRow>
                  ) : (
                    stocks.map((stock, index) => {
                      const holding = holdings.find((h) => h.symbol === stock.stock_code)
                      const globalIndex = pagination
                        ? (pagination.current_page - 1) * pagination.limit + index + 1
                        : index + 1
                      const rankingStock = selectedCategory !== 'all' && rankingData ? rankingData.stocks[index] : null

                      return (
                        <TableRow
                          key={stock.stock_code}
                          className="cursor-pointer hover:bg-accent"
                          onClick={() => handleStockClick(stock)}
                        >
                          <TableCell className="text-center">
                            {selectedCategory !== 'all' && rankingStock ? (
                              <Badge
                                variant={rankingStock.rank <= 3 ? 'default' : 'outline'}
                                className={cn(
                                  'min-w-[2rem] justify-center',
                                  rankingStock.rank === 1 && 'bg-yellow-500 hover:bg-yellow-600',
                                  rankingStock.rank === 2 && 'bg-gray-400 hover:bg-gray-500',
                                  rankingStock.rank === 3 && 'bg-orange-600 hover:bg-orange-700'
                                )}
                              >
                                {rankingStock.rank}
                              </Badge>
                            ) : (
                              <span className="text-muted-foreground">{globalIndex}</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <StockSymbol
                              symbol={stock.stock_code}
                              symbolName={stock.stock_name}
                              market={stock.market}
                              size="sm"
                              isHolding={!!holding}
                              isExitEnabled={holding?.exit_mode === 'ENABLED'}
                            />
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {formatNumber(stock.current_price, 0)}
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {formatPercent(stock.change_rate)}
                          </TableCell>
                          {selectedCategory === 'all' && (
                            <TableCell className="text-sm text-muted-foreground truncate max-w-xs">
                              {stock.sector || '-'}
                            </TableCell>
                          )}
                          {selectedCategory === 'volume' && rankingStock && (
                            <TableCell className="text-right font-mono">
                              {formatNumber(rankingStock.volume)}
                            </TableCell>
                          )}
                          {selectedCategory === 'volume_surge' && rankingStock && (
                            <TableCell className="text-right font-mono font-medium text-red-600">
                              +{rankingStock.volume_surge_rate?.toFixed(2)}%
                            </TableCell>
                          )}
                          {selectedCategory === 'trading_value' && rankingStock && (
                            <TableCell className="text-right font-mono">
                              {formatNumber(rankingStock.trading_value)}
                            </TableCell>
                          )}
                          {selectedCategory === 'high_52week' && rankingStock && (
                            <TableCell className="text-right font-mono">
                              {formatNumber(rankingStock.high_52week, 0)}
                            </TableCell>
                          )}
                          {selectedCategory === 'market_cap' && rankingStock && (
                            <TableCell className="text-right font-mono">
                              {formatNumber(rankingStock.market_cap)}
                            </TableCell>
                          )}
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>

              {/* Pagination (전체 탭에만 표시) */}
              {selectedCategory === 'all' && pagination && pagination.total_pages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    {pagination.total_count.toLocaleString()}개 중{' '}
                    {((pagination.current_page - 1) * pagination.limit + 1).toLocaleString()} -{' '}
                    {Math.min(
                      pagination.current_page * pagination.limit,
                      pagination.total_count
                    ).toLocaleString()}
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handlePrevPage}
                      disabled={pagination.current_page === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>

                    {getPageNumbers().map((pageNum) => (
                      <Button
                        key={pageNum}
                        variant={pageNum === pagination.current_page ? 'default' : 'outline'}
                        size="sm"
                        onClick={() => handlePageClick(pageNum)}
                        className="min-w-[2.5rem]"
                      >
                        {pageNum}
                      </Button>
                    ))}

                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleNextPage}
                      disabled={pagination.current_page === pagination.total_pages}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* StockDetailSheet */}
      <StockDetailSheet
        stock={selectedStock}
        open={isStockDetailOpen}
        onOpenChange={handleOpenChange}
        holdings={holdings}
        unfilledOrders={[]}
        executedOrders={[]}
        totalEvaluation={0}
        onExitModeToggle={() => {}}
      />
    </div>
  )
}
