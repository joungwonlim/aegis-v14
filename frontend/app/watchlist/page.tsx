'use client';

import { useState, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Plus, RefreshCw, Star, Trash2, Loader2, Search, X } from 'lucide-react';
import { StockSymbol } from '@/components/stock-symbol';
import { ChangeIndicator } from '@/components/ui/change-indicator';
import { StockDetailSheet, useStockDetail, type StockInfo } from '@/components/stock-detail-sheet';
import { useHoldings } from '@/hooks/useRuntimeData';
import {
  useWatchItems,
  useCandidateItems,
  useDeleteWatchlistItem,
  useCreateWatchlistItem,
  type WatchlistCategory
} from '@/hooks/useWatchlist';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';

interface SearchResult {
  stock_code: string;
  stock_name: string;
  market: string;
  sector?: string;
}

export default function WatchlistPage() {
  const [activeTab, setActiveTab] = useState<WatchlistCategory>('watch');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  // 데이터 조회
  const { data: watchItems = [], isLoading: watchLoading, refetch: refetchWatch } = useWatchItems();
  const { data: candidateItems = [], isLoading: candidateLoading, refetch: refetchCandidate } = useCandidateItems();
  const { data: holdings = [] } = useHoldings();

  // Mutations
  const deleteItem = useDeleteWatchlistItem();
  const createItem = useCreateWatchlistItem();

  // StockDetailSheet
  const { selectedStock, isOpen: isStockDetailOpen, openStockDetail, handleOpenChange } = useStockDetail();

  const isLoading = activeTab === 'watch' ? watchLoading : candidateLoading;
  const items = activeTab === 'watch' ? watchItems : candidateItems;

  const handleRefresh = () => {
    if (activeTab === 'watch') {
      refetchWatch();
    } else {
      refetchCandidate();
    }
    toast.success('새로고침 완료');
  };

  const handleDelete = async (id: number, stockName: string) => {
    if (!confirm(`${stockName} 종목을 삭제하시겠습니까?`)) {
      return;
    }

    try {
      await deleteItem.mutateAsync(id);
      toast.success('종목이 삭제되었습니다');
    } catch (error) {
      toast.error('삭제 실패: ' + (error instanceof Error ? error.message : '알 수 없는 오류'));
    }
  };

  // 종목 검색
  const searchStocks = async (query: string) => {
    if (!query.trim()) {
      setSearchResults([]);
      return;
    }

    setSearchLoading(true);
    try {
      const res = await fetch(`http://localhost:8099/api/v1/stocks/search?q=${encodeURIComponent(query)}`);
      const data = await res.json();
      if (data.success && data.data) {
        setSearchResults(data.data.slice(0, 10));
      }
    } catch (error) {
      console.error('Search failed:', error);
      setSearchResults([]);
    } finally {
      setSearchLoading(false);
    }
  };

  // Debounced search
  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchQuery(value);

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    debounceRef.current = setTimeout(() => {
      searchStocks(value);
    }, 200);
  };

  // 종목 추가
  const handleAddStock = async (stockCode: string, stockName: string) => {
    try {
      await createItem.mutateAsync({
        stock_code: stockCode,
        category: activeTab,
      });
      toast.success(`${stockName} 종목이 ${activeTab === 'watch' ? '관심종목' : '투자할종목'}에 추가되었습니다`);
      setAddDialogOpen(false);
      setSearchQuery('');
      setSearchResults([]);
    } catch (error) {
      toast.error('추가 실패: ' + (error instanceof Error ? error.message : '알 수 없는 오류'));
    }
  };

  // 종목 클릭
  const handleStockClick = (stockCode: string, stockName: string) => {
    openStockDetail({
      symbol: stockCode,
      symbolName: stockName,
    });
  };

  const formatNumber = (value: number | undefined, decimals = 0) => {
    return value?.toLocaleString('ko-KR', { minimumFractionDigits: decimals, maximumFractionDigits: decimals }) ?? '-';
  };

  const formatPercent = (value: string | undefined) => {
    if (!value) return <span className="text-muted-foreground">-</span>;
    const numValue = parseFloat(value);
    const color = numValue >= 0 ? '#EA5455' : '#2196F3';
    const sign = numValue > 0 ? '+' : '';
    return <span style={{ color }}>{sign}{numValue.toFixed(2)}%</span>;
  };

  const getMarketColor = (market: string) => {
    switch (market) {
      case 'KOSPI':
        return 'bg-blue-500/20 text-blue-400';
      case 'KOSDAQ':
        return 'bg-purple-500/20 text-purple-400';
      case 'ETF':
        return 'bg-green-500/20 text-green-400';
      default:
        return 'bg-gray-500/20 text-gray-400';
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Star className="h-8 w-8 text-yellow-500" />
            관심종목
          </h1>
          <p className="text-muted-foreground">관심종목과 투자할종목을 관리합니다</p>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as WatchlistCategory)}>
        <TabsList>
          <TabsTrigger value="watch">관심종목 ({watchItems.length})</TabsTrigger>
          <TabsTrigger value="candidate">투자할종목 ({candidateItems.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="watch" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex justify-between items-center">
                <div>
                  <CardTitle>관심종목</CardTitle>
                  <CardDescription>모니터링 중인 종목 목록</CardDescription>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading}>
                    <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    새로고침
                  </Button>
                  <Button size="sm" onClick={() => setAddDialogOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    종목 추가
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12">순번</TableHead>
                      <TableHead>종목명</TableHead>
                      <TableHead className="text-right">현재가</TableHead>
                      <TableHead className="text-right">전일대비</TableHead>
                      <TableHead className="text-center">액션</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                          관심종목이 없습니다
                        </TableCell>
                      </TableRow>
                    ) : (
                      items.map((item, index) => {
                        const holding = holdings.find(h => h.symbol === item.stock_code);
                        return (
                          <TableRow key={item.id}>
                            <TableCell className="text-center text-muted-foreground">{index + 1}</TableCell>
                            <TableCell
                              className="cursor-pointer hover:opacity-80"
                              onClick={() => handleStockClick(item.stock_code, item.stock_name)}
                            >
                              <StockSymbol
                                symbol={item.stock_code}
                                symbolName={item.stock_name}
                                market={item.market}
                                size="sm"
                                isHolding={!!holding}
                                isExitEnabled={holding?.exit_mode === 'ENABLED'}
                              />
                            </TableCell>
                            <TableCell className="text-right font-mono">
                              {formatNumber(item.current_price, 0)}
                            </TableCell>
                            <TableCell className="text-right font-mono">
                              {formatPercent(item.change_rate)}
                            </TableCell>
                            <TableCell className="text-center">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-destructive hover:text-destructive"
                                onClick={() => handleDelete(item.id, item.stock_name)}
                                disabled={deleteItem.isPending}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="candidate" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex justify-between items-center">
                <div>
                  <CardTitle>투자할종목</CardTitle>
                  <CardDescription>매수 검토 중인 종목 목록</CardDescription>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading}>
                    <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                    새로고침
                  </Button>
                  <Button size="sm" onClick={() => setAddDialogOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    종목 추가
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12">순번</TableHead>
                      <TableHead>종목명</TableHead>
                      <TableHead className="text-right">현재가</TableHead>
                      <TableHead className="text-right">전일대비</TableHead>
                      <TableHead>선정이유</TableHead>
                      <TableHead className="text-right">목표가</TableHead>
                      <TableHead className="text-center">액션</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                          투자할종목이 없습니다
                        </TableCell>
                      </TableRow>
                    ) : (
                      items.map((item, index) => {
                        const holding = holdings.find(h => h.symbol === item.stock_code);
                        return (
                          <TableRow key={item.id}>
                            <TableCell className="text-center text-muted-foreground">{index + 1}</TableCell>
                            <TableCell
                              className="cursor-pointer hover:opacity-80"
                              onClick={() => handleStockClick(item.stock_code, item.stock_name)}
                            >
                              <StockSymbol
                                symbol={item.stock_code}
                                symbolName={item.stock_name}
                                market={item.market}
                                size="sm"
                                isHolding={!!holding}
                                isExitEnabled={holding?.exit_mode === 'ENABLED'}
                              />
                            </TableCell>
                            <TableCell className="text-right font-mono">
                              {formatNumber(item.current_price, 0)}
                            </TableCell>
                            <TableCell className="text-right font-mono">
                              {formatPercent(item.change_rate)}
                            </TableCell>
                            <TableCell className="text-sm text-muted-foreground max-w-xs truncate">
                              {item.memo || '-'}
                            </TableCell>
                            <TableCell className="text-right font-mono">
                              {item.target_price ? formatNumber(item.target_price, 0) : '-'}
                            </TableCell>
                            <TableCell className="text-center">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-destructive hover:text-destructive"
                                onClick={() => handleDelete(item.id, item.stock_name)}
                                disabled={deleteItem.isPending}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 종목 추가 다이얼로그 */}
      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{activeTab === 'watch' ? '관심종목' : '투자할종목'} 추가</DialogTitle>
            <DialogDescription>
              종목명 또는 종목코드를 검색하여 추가할 종목을 선택하세요
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* 검색 입력 */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={handleSearchChange}
                placeholder="종목명, 종목코드 검색"
                className="h-10 w-full rounded-md bg-muted pl-10 pr-10 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-primary border border-border"
              />
              {searchQuery && (
                <button
                  onClick={() => {
                    setSearchQuery('');
                    setSearchResults([]);
                  }}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-accent"
                >
                  <X className="h-3 w-3 text-muted-foreground" />
                </button>
              )}
            </div>

            {/* 검색 결과 */}
            <div className="max-h-96 overflow-y-auto border rounded-lg">
              {searchLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : searchResults.length > 0 ? (
                <div>
                  {searchResults.map((result) => (
                    <button
                      key={result.stock_code}
                      onClick={() => handleAddStock(result.stock_code, result.stock_name)}
                      className="w-full flex items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-accent border-b last:border-b-0"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-foreground truncate">
                            {result.stock_name}
                          </span>
                          <Badge variant="outline" className={cn('text-xs', getMarketColor(result.market))}>
                            {result.market}
                          </Badge>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <span>{result.stock_code}</span>
                          {result.sector && (
                            <>
                              <span>·</span>
                              <span>{result.sector}</span>
                            </>
                          )}
                        </div>
                      </div>
                      <Plus className="h-4 w-4 text-muted-foreground" />
                    </button>
                  ))}
                </div>
              ) : searchQuery ? (
                <div className="py-8 text-center text-muted-foreground">
                  <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">검색 결과가 없습니다</p>
                  <p className="text-xs mt-1">다른 종목명이나 코드로 검색해보세요</p>
                </div>
              ) : (
                <div className="py-8 text-center text-muted-foreground">
                  <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">종목을 검색하세요</p>
                </div>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* StockDetailSheet */}
      <StockDetailSheet
        stock={selectedStock}
        open={isStockDetailOpen}
        onOpenChange={handleOpenChange}
        holdings={holdings}
      />
    </div>
  );
}
