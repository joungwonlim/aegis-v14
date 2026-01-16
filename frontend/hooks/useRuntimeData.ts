'use client'

import { useQuery } from '@tanstack/react-query'
import {
  getHoldings,
  getOrderIntents,
  getOrders,
  getFills,
  getKISUnfilledOrders,
  getKISFilledOrders,
  type Holding,
  type OrderIntent,
  type Order,
  type Fill,
  type KISUnfilledOrder,
  type KISFill,
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
    refetchInterval: 1000, // 1초마다 갱신
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
 * KIS 미체결 주문 조회 (1초마다 자동 갱신)
 */
export function useKISUnfilledOrders() {
  return useQuery({
    queryKey: RUNTIME_KEYS.kisUnfilledOrders(),
    queryFn: async () => {
      const result = await getKISUnfilledOrders()
      return result || []
    },
    staleTime: 500,
    refetchInterval: 1000,
    placeholderData: (previousData) => previousData,
  })
}

/**
 * KIS 체결 주문 조회 (1초마다 자동 갱신)
 */
export function useKISFilledOrders() {
  return useQuery({
    queryKey: RUNTIME_KEYS.kisFilledOrders(),
    queryFn: async () => {
      const result = await getKISFilledOrders()
      return result || []
    },
    staleTime: 500,
    refetchInterval: 1000,
    placeholderData: (previousData) => previousData,
  })
}
