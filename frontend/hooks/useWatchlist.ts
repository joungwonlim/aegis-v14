import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

/**
 * Watchlist 타입 정의
 */
export type WatchlistCategory = 'watch' | 'candidate';

export interface WatchlistItem {
  id: number;
  stock_code: string;
  category: WatchlistCategory;
  memo?: string;
  target_price?: number;

  // AI 분석 필드
  grok_analysis?: string;
  gemini_analysis?: string;
  chatgpt_analysis?: string;
  claude_analysis?: string;

  // 알림
  alert_enabled: boolean;
  alert_price?: number;
  alert_condition?: 'above' | 'below';

  created_at: string;
  updated_at: string;

  // JOIN된 종목 정보
  stock_name: string;
  market: string;
  current_price?: number;
  change_rate?: string;
}

export interface CreateWatchlistRequest {
  stock_code: string;
  category: WatchlistCategory;
  memo?: string;
  target_price?: number;
}

export interface UpdateWatchlistRequest {
  memo?: string;
  target_price?: number;
  alert_enabled?: boolean;
  alert_price?: number;
  alert_condition?: 'above' | 'below';
}

export interface SearchResult {
  stock_code: string;
  stock_name: string;
  market: string;
  sector?: string;
}

const API_BASE = 'http://localhost:8099/api/v1';

/**
 * Watchlist Query Keys
 */
export const WATCHLIST_KEYS = {
  all: ['watchlist'] as const,
  lists: () => [...WATCHLIST_KEYS.all, 'list'] as const,
  list: (category?: WatchlistCategory) => [...WATCHLIST_KEYS.lists(), category] as const,
  search: (query: string) => [...WATCHLIST_KEYS.all, 'search', query] as const,
};

/**
 * 전체 Watchlist 조회 (카테고리별)
 */
export function useWatchlist() {
  return useQuery({
    queryKey: WATCHLIST_KEYS.lists(),
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/watchlist`);
      if (!res.ok) throw new Error('Failed to fetch watchlist');
      const data = await res.json();
      return data.data;
    },
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

/**
 * 관심종목만 조회
 */
export function useWatchItems() {
  return useQuery<WatchlistItem[]>({
    queryKey: WATCHLIST_KEYS.list('watch'),
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/watchlist/watch`);
      if (!res.ok) throw new Error('Failed to fetch watch items');
      const data = await res.json();
      return data.data || [];
    },
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

/**
 * 투자할종목만 조회
 */
export function useCandidateItems() {
  return useQuery<WatchlistItem[]>({
    queryKey: WATCHLIST_KEYS.list('candidate'),
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/watchlist/candidate`);
      if (!res.ok) throw new Error('Failed to fetch candidate items');
      const data = await res.json();
      return data.data || [];
    },
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

/**
 * Watchlist 종목 추가
 */
export function useCreateWatchlistItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateWatchlistRequest) => {
      const res = await fetch(`${API_BASE}/watchlist`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error || 'Failed to create watchlist item');
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WATCHLIST_KEYS.all });
    },
  });
}

/**
 * Watchlist 종목 수정
 */
export function useUpdateWatchlistItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...request }: UpdateWatchlistRequest & { id: number }) => {
      const res = await fetch(`${API_BASE}/watchlist/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!res.ok) throw new Error('Failed to update watchlist item');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WATCHLIST_KEYS.all });
    },
  });
}

/**
 * Watchlist 종목 삭제
 */
export function useDeleteWatchlistItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const res = await fetch(`${API_BASE}/watchlist/${id}`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error('Failed to delete watchlist item');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WATCHLIST_KEYS.all });
    },
  });
}

/**
 * 종목 검색
 */
export function useStockSearch(query: string) {
  return useQuery<SearchResult[]>({
    queryKey: WATCHLIST_KEYS.search(query),
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/stocks/search?q=${encodeURIComponent(query)}`);
      if (!res.ok) throw new Error('Failed to search stocks');
      const data = await res.json();
      return data.data || [];
    },
    enabled: query.length >= 1,
    staleTime: 60 * 1000,
  });
}

/**
 * 특정 종목이 Watchlist에 있는지 확인
 */
export function useIsInWatchlist(stockCode: string) {
  const { data: watchItems = [] } = useWatchItems();
  const { data: candidateItems = [] } = useCandidateItems();

  const allItems = [...watchItems, ...candidateItems];
  const item = allItems.find(item => item.stock_code === stockCode);

  return {
    isInWatchlist: !!item,
    item,
    category: item?.category,
  };
}
