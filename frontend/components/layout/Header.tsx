'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { Search, Bell, User, X, TrendingUp, Building2, Loader2, History } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { cn } from '@/lib/utils';
import { StockDetailSheet, useStockDetail } from '@/components/stock-detail-sheet';
import { useHoldings } from '@/hooks/useRuntimeData';

interface SearchResult {
  stock_code: string;
  stock_name: string;
  market: string;
  sector?: string;
}

interface SearchHistory {
  stock_code: string;
  stock_name: string;
  market: string;
  timestamp: number;
}

const SEARCH_HISTORY_KEY = 'aegis_search_history';
const MAX_HISTORY_ITEMS = 10;

export function Header() {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searchHistory, setSearchHistory] = useState<SearchHistory[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  // StockDetailSheet
  const { selectedStock, isOpen: isStockDetailOpen, openStockDetail, handleOpenChange } = useStockDetail();
  const { data: holdings = [] } = useHoldings();

  // Load search history from localStorage
  useEffect(() => {
    const loadHistory = () => {
      try {
        const stored = localStorage.getItem(SEARCH_HISTORY_KEY);
        if (stored) {
          const history = JSON.parse(stored) as SearchHistory[];
          setSearchHistory(history.slice(0, MAX_HISTORY_ITEMS));
        }
      } catch (error) {
        console.error('Failed to load search history:', error);
      }
    };
    loadHistory();
  }, []);

  // Keyboard shortcut (Cmd+K or Ctrl+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        setIsOpen(true);
      }
      if (e.key === 'Escape') {
        setIsOpen(false);
        inputRef.current?.blur();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        !inputRef.current?.contains(e.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Search function
  const search = useCallback(async (searchQuery: string) => {
    if (!searchQuery.trim()) {
      setResults([]);
      return;
    }

    setIsLoading(true);
    try {
      const res = await fetch(
        `http://localhost:8099/api/v1/stocks/search?q=${encodeURIComponent(searchQuery)}`
      );
      const data = await res.json();
      if (data.success && data.data) {
        setResults(data.data.slice(0, 10));
      }
    } catch (error) {
      console.error('Search failed:', error);
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Debounced search
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setQuery(value);
    setSelectedIndex(-1);

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    debounceRef.current = setTimeout(() => {
      search(value);
    }, 200);
  };

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!isOpen || results.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) => (prev < results.length - 1 ? prev + 1 : prev));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : -1));
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && results[selectedIndex]) {
          const result = results[selectedIndex];
          handleStockClick(result.stock_code, result.stock_name, result.market);
        }
        break;
    }
  };

  // Add to search history
  const addToHistory = (stockCode: string, stockName: string, market: string) => {
    try {
      const newItem: SearchHistory = {
        stock_code: stockCode,
        stock_name: stockName,
        market: market,
        timestamp: Date.now(),
      };

      // Remove duplicates and add to front
      const filtered = searchHistory.filter(item => item.stock_code !== stockCode);
      const newHistory = [newItem, ...filtered].slice(0, MAX_HISTORY_ITEMS);

      setSearchHistory(newHistory);
      localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(newHistory));
    } catch (error) {
      console.error('Failed to save search history:', error);
    }
  };

  // Open stock detail
  const handleStockClick = (stockCode: string, stockName: string, market: string) => {
    // Add to history
    addToHistory(stockCode, stockName, market);

    // Close search
    setIsOpen(false);
    setQuery('');
    setResults([]);

    // Open StockDetailSheet
    openStockDetail({
      symbol: stockCode,
      symbolName: stockName,
    });
  };

  // Clear search
  const clearSearch = () => {
    setQuery('');
    setResults([]);
    setSelectedIndex(-1);
    inputRef.current?.focus();
  };

  // Get market badge color
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
    <header className="flex h-14 items-center justify-between border-b border-border bg-background px-4">
      {/* Left Section - Search */}
      <div className="flex items-center gap-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={handleInputChange}
            onFocus={() => setIsOpen(true)}
            onKeyDown={handleKeyDown}
            placeholder="종목명, 종목코드 검색"
            className="h-9 w-80 rounded-md bg-muted pl-10 pr-16 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:ring-2 focus:ring-primary border border-border"
          />
          {query && (
            <button
              onClick={clearSearch}
              className="absolute right-10 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-accent"
            >
              <X className="h-3 w-3 text-muted-foreground" />
            </button>
          )}
          <kbd className="absolute right-3 top-1/2 -translate-y-1/2 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground border border-border">
            ⌘K
          </kbd>

          {/* Search Results Dropdown */}
          {isOpen && (query || results.length > 0) && (
            <div
              ref={dropdownRef}
              className="absolute top-full left-0 right-0 mt-2 bg-popover border border-border rounded-lg shadow-xl overflow-hidden z-50"
            >
              {isLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : results.length > 0 ? (
                <div className="max-h-96 overflow-y-auto">
                  {results.map((result, index) => (
                    <button
                      key={result.stock_code}
                      onClick={() => handleStockClick(result.stock_code, result.stock_name, result.market)}
                      className={cn(
                        'w-full flex items-center gap-3 px-4 py-3 text-left transition-colors',
                        'hover:bg-accent',
                        selectedIndex === index && 'bg-accent'
                      )}
                    >
                      <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-muted flex items-center justify-center">
                        <TrendingUp className="w-4 h-4 text-muted-foreground" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-foreground truncate">
                            {result.stock_name}
                          </span>
                          <span className={cn('px-1.5 py-0.5 text-[10px] font-medium rounded', getMarketColor(result.market))}>
                            {result.market}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <span>{result.stock_code}</span>
                          {result.sector && (
                            <>
                              <span>·</span>
                              <span className="flex items-center gap-1">
                                <Building2 className="w-3 h-3" />
                                {result.sector}
                              </span>
                            </>
                          )}
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              ) : query ? (
                <div className="py-8 text-center text-muted-foreground">
                  <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">검색 결과가 없습니다</p>
                  <p className="text-xs mt-1">다른 종목명이나 코드로 검색해보세요</p>
                </div>
              ) : null}

              {/* Search history */}
              {!query && results.length === 0 && searchHistory.length > 0 && (
                <div className="py-2">
                  <div className="px-4 py-2 flex items-center gap-2">
                    <History className="w-4 h-4 text-muted-foreground" />
                    <p className="text-xs text-muted-foreground">최근 검색</p>
                  </div>
                  <div>
                    {searchHistory.map((item) => (
                      <button
                        key={item.stock_code}
                        onClick={() => handleStockClick(item.stock_code, item.stock_name, item.market)}
                        className="w-full flex items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-accent border-b last:border-b-0"
                      >
                        <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-muted flex items-center justify-center">
                          <History className="w-4 h-4 text-muted-foreground" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-foreground truncate">
                              {item.stock_name}
                            </span>
                            <span className={cn('px-1.5 py-0.5 text-[10px] font-medium rounded', getMarketColor(item.market))}>
                              {item.market}
                            </span>
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {item.stock_code}
                          </div>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Right Section - Actions */}
      <div className="flex items-center gap-3">
        <button className="rounded-full p-2 hover:bg-accent">
          <Bell className="h-5 w-5 text-muted-foreground" />
        </button>
        <button className="rounded-full p-2 hover:bg-accent">
          <User className="h-5 w-5 text-muted-foreground" />
        </button>
      </div>

      {/* StockDetailSheet */}
      <StockDetailSheet
        stock={selectedStock}
        open={isStockDetailOpen}
        onOpenChange={handleOpenChange}
        holdings={holdings}
      />
    </header>
  );
}
