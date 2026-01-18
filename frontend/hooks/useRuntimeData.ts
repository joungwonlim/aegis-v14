'use client'

import { useQuery } from '@tanstack/react-query'
import {
  getHoldings,
  getOrderIntents,
  getOrders,
  getFills,
  getKISUnfilledOrders,
  getKISFilledOrders,
  getExitProfiles,
  getSymbolOverride,
  type Holding,
  type OrderIntent,
  type Order,
  type Fill,
  type KISUnfilledOrder,
  type KISFill,
  type ExitProfile,
  type SymbolExitOverride,
} from '@/lib/api'

// Query Keys
const RUNTIME_KEYS = {
  all: ['runtime'] as const,
  holdings: () => [...RUNTIME_KEYS.all, 'holdings'] as const,
  intents: () => [...RUNTIME_KEYS.all, 'intents'] as const,
  orders: () => [...RUNTIME_KEYS.all, 'orders'] as const,
  fills: () => [...RUNTIME_KEYS.all, 'fills'] as const,
  kisUnfilledOrders: () => [...RUNTIME_KEYS.all, 'kis-unfilled-orders'] as const,
  kisFilledOrders: () => [...RUNTIME_KEYS.all, 'kis-filled-orders'] as const,
  exitProfiles: (activeOnly: boolean) => [...RUNTIME_KEYS.all, 'exit-profiles', activeOnly] as const,
  symbolOverride: (symbol: string) => [...RUNTIME_KEYS.all, 'symbol-override', symbol] as const,
}

/**
 * Holdings 조회 (1초마다 자동 갱신)
 */
export function useHoldings() {
  return useQuery({
    queryKey: RUNTIME_KEYS.holdings(),
    queryFn: async () => {
      const result = await getHoldings()
      return result || []
    },
    staleTime: 500, // 0.5초
    refetchInterval: 5000, // 5초마다 갱신 (임시로 늘림)
    placeholderData: (previousData) => previousData, // 이전 데이터 유지 (깜빡임 방지)
  })
}

/**
 * Order Intents 조회 (1초마다 자동 갱신)
 */
export function useOrderIntents() {
  return useQuery({
    queryKey: RUNTIME_KEYS.intents(),
    queryFn: async () => {
      const result = await getOrderIntents()
      return result || []
    },
    staleTime: 500,
    refetchInterval: 1000,
    placeholderData: (previousData) => previousData,
  })
}

/**
 * Orders 조회 (1초마다 자동 갱신)
 */
export function useOrders() {
  return useQuery({
    queryKey: RUNTIME_KEYS.orders(),
    queryFn: async () => {
      const result = await getOrders()
      return result || []
    },
    staleTime: 500,
    refetchInterval: 1000,
    placeholderData: (previousData) => previousData,
  })
}

/**
 * Fills 조회 (1초마다 자동 갱신)
 */
export function useFills() {
  return useQuery({
    queryKey: RUNTIME_KEYS.fills(),
    queryFn: async () => {
      const result = await getFills()
      return result || []
    },
    staleTime: 500,
    refetchInterval: 1000,
    placeholderData: (previousData) => previousData,
  })
}

/**
 * KIS 미체결 주문 조회 (30초마다 자동 갱신)
 * Note: Backend 캐싱(30초)과 함께 KIS REST API 호출량 최소화
 */
export function useKISUnfilledOrders() {
  return useQuery({
    queryKey: RUNTIME_KEYS.kisUnfilledOrders(),
    queryFn: async () => {
      const result = await getKISUnfilledOrders()
      return result || []
    },
    staleTime: 15000, // 15초
    refetchInterval: 30000, // 30초마다 갱신 (KIS rate limit 대응)
    placeholderData: (previousData) => previousData,
  })
}

/**
 * KIS 체결 주문 조회 (30초마다 자동 갱신)
 * Note: Backend 캐싱(30초)과 함께 KIS REST API 호출량 최소화
 */
export function useKISFilledOrders() {
  return useQuery({
    queryKey: RUNTIME_KEYS.kisFilledOrders(),
    queryFn: async () => {
      const result = await getKISFilledOrders()
      return result || []
    },
    staleTime: 15000, // 15초
    refetchInterval: 30000, // 30초마다 갱신 (KIS rate limit 대응)
    placeholderData: (previousData) => previousData,
  })
}

/**
 * Exit 프로필 목록 조회 (10초마다 자동 갱신)
 * @param activeOnly - true이면 활성 프로필만 조회 (기본값: true)
 */
export function useExitProfiles(activeOnly: boolean = true) {
  return useQuery({
    queryKey: RUNTIME_KEYS.exitProfiles(activeOnly),
    queryFn: async () => {
      const result = await getExitProfiles(activeOnly)
      return result || []
    },
    staleTime: 5000, // 5초
    refetchInterval: 10000, // 10초마다 갱신 (프로필은 자주 변경되지 않음)
    placeholderData: (previousData) => previousData,
  })
}

/**
 * 종목별 Exit Override 조회 (10초마다 자동 갱신)
 * @param symbol - 종목 코드 (6자리)
 */
export function useSymbolOverride(symbol: string) {
  return useQuery({
    queryKey: RUNTIME_KEYS.symbolOverride(symbol),
    queryFn: async () => {
      const result = await getSymbolOverride(symbol)
      return result // null일 수 있음 (Override 없음)
    },
    staleTime: 5000, // 5초
    refetchInterval: 10000, // 10초마다 갱신
    placeholderData: (previousData) => previousData,
    enabled: !!symbol, // symbol이 있을 때만 쿼리 실행
  })
}
