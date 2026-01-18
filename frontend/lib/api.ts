// Core Runtime API Client

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8099/api'

export interface Holding {
  account_id: string
  symbol: string
  qty: number
  avg_price: number
  current_price: number
  pnl: number
  pnl_pct: number
  change_price?: number // 전일대비 가격 (원, from prices_best)
  change_rate?: number // 전일대비 등락률 (%, from prices_best)
  updated_ts: string
  exit_mode: string // ENABLED, DISABLED, MANUAL_ONLY
  raw?: {
    symbol_name?: string
    evaluate_amount?: string
    purchase_amount?: string
    evaluate_profit_loss?: string
    evaluate_profit_loss_rate?: string
    market?: string // KOSPI, KOSDAQ, UNKNOWN
  }
}

export interface OrderIntent {
  intent_id: string
  position_id: string
  symbol: string
  symbol_name?: string // 종목명 (optional)
  intent_type: string // EXIT_PARTIAL, EXIT_FULL
  qty: number
  order_type: string // MKT, LMT
  limit_price?: number
  reason_code: string // SL1, SL2, TP1, TP2, TRAILING, CUSTOM, etc.
  reason_detail?: string // 상세 사유 (예: "+4%/10% 익절")
  status: string // PENDING_APPROVAL, NEW, ACK, REJECTED, FILLED, CANCELLED, SUBMITTED
  created_ts: string
}

export interface Order {
  order_id: string
  intent_id: string
  symbol?: string
  submitted_ts: string
  status: string // SUBMITTED, PARTIAL, FILLED, CANCELLED, REJECTED
  broker_status: string
  qty: number
  open_qty: number
  filled_qty: number
  updated_ts: string
}

export interface Fill {
  fill_id: string
  order_id: string
  kis_exec_id: string
  ts: string
  qty: number
  price: number
  fee: number
  tax: number
  seq: number
}

export interface KISUnfilledOrder {
  OrderID: string
  Symbol: string
  Qty: number
  OpenQty: number
  FilledQty: number
  Status: string
  Raw: {
    stock_name?: string
    order_side?: string
    order_price?: string
    avg_exec_price?: string
    order_time?: string
  }
}

export interface KISFill {
  ExecID: string
  OrderID: string
  Symbol: string
  Qty: number
  Price: string
  Fee: string
  Tax: string
  Timestamp: string
  Seq: number
  Raw: {
    stock_name?: string
    order_side?: string
    order_qty?: string
    order_price?: string
    exec_amount?: string
    cancel_yn?: string
    order_type?: string
  }
}

export async function getHoldings(): Promise<Holding[]> {
  const response = await fetch(`${API_BASE_URL}/holdings`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to fetch holdings: ${response.statusText}`)
  }

  const data = await response.json()
  console.log('Holdings API response:', data)

  // Debug: Check specific symbol 316140
  const symbol316140 = data.find((h: Holding) => h.symbol === '316140')
  if (symbol316140) {
    console.log('Symbol 316140 exit_mode:', symbol316140.exit_mode, 'Account:', symbol316140.account_id)
  }

  return data
}

export async function getOrderIntents(): Promise<OrderIntent[]> {
  const response = await fetch(`${API_BASE_URL}/intents`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to fetch order intents: ${response.statusText}`)
  }

  return response.json()
}

export async function getOrders(): Promise<Order[]> {
  const response = await fetch(`${API_BASE_URL}/orders`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to fetch orders: ${response.statusText}`)
  }

  return response.json()
}

export async function getFills(): Promise<Fill[]> {
  const response = await fetch(`${API_BASE_URL}/fills`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to fetch fills: ${response.statusText}`)
  }

  return response.json()
}

export async function approveIntent(intentId: string): Promise<{ status: string; message?: string }> {
  const response = await fetch(`${API_BASE_URL}/intents/${intentId}/approve`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }

  return response.json()
}

export async function rejectIntent(intentId: string): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE_URL}/intents/${intentId}/reject`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to reject intent: ${response.statusText}`)
  }

  return response.json()
}

export async function updateExitMode(
  accountId: string,
  symbol: string,
  exitMode: string,
  holding?: {
    qty: number
    avg_price: number
  }
): Promise<{ status: string, exit_mode: string }> {
  const response = await fetch(`${API_BASE_URL}/holdings/${accountId}/${symbol}/exit-mode`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      exit_mode: exitMode,
      qty: holding?.qty,
      avg_price: holding?.avg_price,
    }),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`Failed to update exit mode: ${errorText || response.statusText}`)
  }

  return response.json()
}

export async function getKISUnfilledOrders(): Promise<KISUnfilledOrder[]> {
  try {
    const response = await fetch(`${API_BASE_URL}/kis/unfilled-orders`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      const errorText = await response.text()
      console.error('KIS Unfilled Orders API Error:', response.status, errorText)
      throw new Error(`Failed to fetch KIS unfilled orders: ${response.statusText}`)
    }

    const data = await response.json()
    console.log('KIS Unfilled Orders:', data)
    return data
  } catch (error) {
    console.error('KIS Unfilled Orders fetch error:', error)
    throw error
  }
}

export async function getKISFilledOrders(): Promise<KISFill[]> {
  try {
    const response = await fetch(`${API_BASE_URL}/kis/filled-orders`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      const errorText = await response.text()
      console.error('KIS Filled Orders API Error:', response.status, errorText)
      throw new Error(`Failed to fetch KIS filled orders: ${response.statusText}`)
    }

    const data = await response.json()
    console.log('KIS Filled Orders:', data)
    return data
  } catch (error) {
    console.error('KIS Filled Orders fetch error:', error)
    throw error
  }
}

export interface PlaceOrderRequest {
  symbol: string      // 종목코드 (6자리)
  side: string        // 'buy' 또는 'sell'
  order_type: string  // 'limit' 또는 'market'
  qty: number         // 주문수량
  price: number       // 주문가격 (시장가일 경우 0)
}

export interface PlaceOrderResponse {
  success: boolean
  order_id?: string
  error?: string
}

export async function placeKISOrder(req: PlaceOrderRequest): Promise<PlaceOrderResponse> {
  const response = await fetch(`${API_BASE_URL}/kis/orders`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error('KIS Place Order API Error:', response.status, errorText)
    throw new Error(`Failed to place order: ${response.statusText}`)
  }

  const data: PlaceOrderResponse = await response.json()
  return data
}

export interface CancelOrderResponse {
  success: boolean
  cancel_no?: string
  error?: string
}

export async function cancelKISOrder(orderNo: string): Promise<CancelOrderResponse> {
  const response = await fetch(`${API_BASE_URL}/kis/orders/${orderNo}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error('KIS Cancel Order API Error:', response.status, errorText)
    throw new Error(`Failed to cancel order: ${response.statusText}`)
  }

  const data: CancelOrderResponse = await response.json()
  return data
}

// =====================================
// Exit Engine API
// =====================================

/**
 * Exit Control
 */
export interface ExitControl {
  mode: string // RUNNING, PAUSE_ALL, PAUSE_PROFIT, EMERGENCY_FLATTEN
  reason?: string
  updated_by: string
  updated_ts: string
}

export async function getExitControl(): Promise<ExitControl> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/control`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to get exit control: ${response.statusText}`)
  }

  return response.json()
}

export async function updateExitControl(mode: string, reason?: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/control`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      mode,
      reason,
      updated_by: 'web-ui',
    }),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }
}

/**
 * Exit Profiles
 */
export interface CustomExitRule {
  id: string
  enabled: boolean
  condition: 'profit_above' | 'profit_below'
  threshold: number
  exit_percent: number
  priority: number
  description?: string
}

export interface ExitProfileConfig {
  atr?: {
    ref: number
    factor_min: number
    factor_max: number
  }
  sl1?: {
    base_pct: number
    min_pct: number
    max_pct: number
    qty_pct: number
  }
  sl2?: {
    base_pct: number
    min_pct: number
    max_pct: number
    qty_pct: number
  }
  tp1?: {
    base_pct: number
    min_pct: number
    max_pct: number
    qty_pct: number
    stop_floor_profit: number
  }
  tp2?: {
    base_pct: number
    min_pct: number
    max_pct: number
    qty_pct: number
  }
  tp3?: {
    base_pct: number
    min_pct: number
    max_pct: number
    qty_pct: number
    start_trailing: boolean
  }
  trailing?: {
    pct_trail: number
    atr_k: number
  }
  time_stop?: {
    max_hold_days: number
    no_momentum_days: number
    no_momentum_profit: number
  }
  hardstop?: {
    enabled: boolean
    pct: number
  }
  custom_rules?: CustomExitRule[]
}

export interface ExitProfile {
  profile_id: string
  name: string
  description: string
  config: ExitProfileConfig
  is_active: boolean
  created_by: string
  created_ts: string
}

export async function getExitProfiles(activeOnly: boolean = true): Promise<ExitProfile[]> {
  const url = activeOnly
    ? `${API_BASE_URL}/v1/exit/profiles?active_only=true`
    : `${API_BASE_URL}/v1/exit/profiles`

  const response = await fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to get exit profiles: ${response.statusText}`)
  }

  const data = await response.json()
  return data.profiles || []
}

export async function createExitProfile(profile: {
  profile_id: string
  name: string
  description: string
  config: ExitProfileConfig
}): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/profiles`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      ...profile,
      created_by: 'web-ui',
    }),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }
}

/**
 * Symbol Exit Override
 */
export interface SymbolExitOverride {
  symbol: string
  profile_id: string
  enabled: boolean
  effective_from?: string
  reason: string
}

export async function getSymbolOverride(symbol: string): Promise<SymbolExitOverride | null> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/overrides/${symbol}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (response.status === 404) {
    return null // Override 없음
  }

  if (!response.ok) {
    throw new Error(`Failed to get symbol override: ${response.statusText}`)
  }

  return response.json()
}

export async function setSymbolOverride(
  symbol: string,
  profileId: string,
  reason: string
): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/overrides/${symbol}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      profile_id: profileId,
      reason,
      created_by: 'web-ui',
    }),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }
}

export async function deleteSymbolOverride(symbol: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/overrides/${symbol}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok && response.status !== 404) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }
}

/**
 * Manual Exit
 */
export async function createManualExit(
  positionId: string,
  qty: number,
  orderType: string
): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/positions/${positionId}/manual`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      qty,
      order_type: orderType,
    }),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || response.statusText)
  }
}

/**
 * Position State
 */
export interface PositionState {
  position_id: string
  phase: string
  hwm_price?: string
  stop_floor_price?: string
  atr?: string
  cooldown_until?: string
  last_eval_ts?: string
  updated_ts: string
}

export async function getPositionState(positionId: string): Promise<PositionState> {
  const response = await fetch(`${API_BASE_URL}/v1/exit/positions/${positionId}/state`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to get position state: ${response.statusText}`)
  }

  return response.json()
}

// =====================================
// Audit API
// =====================================

export type AuditPeriod = '1M' | '3M' | '6M' | '1Y' | 'YTD'

export interface PerformanceReport {
  report_date: string
  period_start: string
  period_end: string
  period_code: string
  total_return: number
  annual_return: number
  benchmark_return: number
  alpha: number
  beta: number
  volatility: number
  sharpe_ratio: number
  sortino_ratio: number
  max_drawdown: number
  win_rate: number
  avg_win: number
  avg_loss: number
  profit_factor: number
  total_trades: number
}

export interface DailyPnL {
  date: string  // Backend JSON 태그와 일치
  realized_pnl: number
  unrealized_pnl: number
  total_pnl: number
  daily_return: number
  cumulative_return: number
  portfolio_value: number
  cash_balance: number
}

export interface AttributionAnalysis {
  analysis_date: string
  period_start: string
  period_end: string
  total_return: number
  factor_contrib: Record<string, number>
  sector_contrib: Record<string, number>
  stock_contrib: Record<string, number>
}

export interface RiskMetrics {
  metric_date: string
  portfolio_var_95: number
  portfolio_var_99: number
  concentration: number
  market_correlation: number
  sector_exposure: Record<string, number>
  avg_turnover_ratio: number
  illiquid_weight: number
}

export interface DailySnapshot {
  date: string
  total_value: number
  cash: number
  positions: unknown[]
  daily_return: number
  cum_return: number
}

export async function getPerformance(period: AuditPeriod = '1M'): Promise<PerformanceReport | null> {
  const response = await fetch(`${API_BASE_URL}/v1/audit/performance?period=${period}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    try {
      const data = await response.json()
      // 데이터 부족 에러는 null 반환 (정상 상황)
      if (data.error && data.error.includes('Insufficient data')) return null
      if (data.error && data.error.includes('insufficient data')) return null
    } catch (e) {
      // JSON 파싱 실패 시 null 반환
      return null
    }
    return null
  }

  const data = await response.json()
  return data.data
}

export async function getDailyPnL(startDate?: string, endDate?: string): Promise<DailyPnL[]> {
  let url = `${API_BASE_URL}/v1/audit/daily-pnl`
  const params = new URLSearchParams()
  if (startDate) params.append('start_date', startDate)
  if (endDate) params.append('end_date', endDate)
  if (params.toString()) url += `?${params.toString()}`

  console.log('[API] getDailyPnL request:', url)

  const response = await fetch(url, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error('[API] getDailyPnL error:', response.status, errorText)
    throw new Error(`Failed to get daily PnL: ${response.statusText}`)
  }

  const data = await response.json()
  console.log('[API] getDailyPnL response:', data)
  return data.data || []
}

export async function getAttribution(period: AuditPeriod = '1M'): Promise<AttributionAnalysis | null> {
  const response = await fetch(`${API_BASE_URL}/v1/audit/attribution?period=${period}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    const data = await response.json()
    if (data.error === 'Insufficient data for analysis') return null
    throw new Error(`Failed to get attribution: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data
}

export async function getRiskMetrics(period: AuditPeriod = '1M'): Promise<RiskMetrics | null> {
  const response = await fetch(`${API_BASE_URL}/v1/audit/risk?period=${period}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    try {
      const data = await response.json()
      // 데이터 부족 에러는 null 반환 (정상 상황)
      if (data.error && data.error.includes('Insufficient data')) return null
      if (data.error && data.error.includes('insufficient data')) return null
    } catch (e) {
      // JSON 파싱 실패 시 null 반환
      return null
    }
    return null
  }

  const data = await response.json()
  return data.data
}

export async function getSnapshots(startDate?: string, endDate?: string): Promise<DailySnapshot[]> {
  let url = `${API_BASE_URL}/v1/audit/snapshots`
  const params = new URLSearchParams()
  if (startDate) params.append('start_date', startDate)
  if (endDate) params.append('end_date', endDate)
  if (params.toString()) url += `?${params.toString()}`

  const response = await fetch(url, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get snapshots: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data || []
}

// =====================================
// Fetcher API (Stocks Universe)
// =====================================

export interface Stock {
  stock_code: string
  stock_name: string
  market: string
  sector?: string
  industry?: string
  market_cap?: number
  status: string            // ACTIVE, SUSPENDED, DELISTED
  is_tradable: boolean
  current_price?: number     // 현재가 (prices_best 조인)
  change_rate?: number       // 전일대비 (%)
}

export interface DailyPrice {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface InvestorFlow {
  date: string
  foreign_net: number
  inst_net: number
  retail_net: number
  close_price: number
  price_change: number
  change_rate: number
  volume: number
}

export interface ListStocksResponse {
  stocks: Stock[]
  pagination: {
    current_page: number
    total_pages: number
    total_count: number
    limit: number
  }
}

export async function listStocks(params?: {
  market?: string
  sector?: string
  search?: string
  page?: number
  limit?: number
}): Promise<ListStocksResponse> {
  let url = `${API_BASE_URL}/v1/stocks`
  const searchParams = new URLSearchParams()
  if (params?.market) searchParams.append('market', params.market)
  if (params?.search) searchParams.append('search', params.search)
  if (params?.page) searchParams.append('page', params.page.toString())
  if (params?.limit) searchParams.append('limit', params.limit.toString())
  if (searchParams.toString()) url += `?${searchParams.toString()}`

  const response = await fetch(url, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to list stocks: ${response.statusText}`)
  }

  const data = await response.json()

  // 응답 형식에 따라 처리
  if (data.data && data.data.stocks) {
    return data.data
  } else if (Array.isArray(data.data)) {
    // 구 API 형식 (배열만 반환)
    return {
      stocks: data.data,
      pagination: {
        current_page: 1,
        total_pages: 1,
        total_count: data.data.length,
        limit: params?.limit || 50,
      }
    }
  } else {
    return {
      stocks: [],
      pagination: {
        current_page: 1,
        total_pages: 1,
        total_count: 0,
        limit: params?.limit || 50,
      }
    }
  }
}

export async function getStock(code: string): Promise<Stock | null> {
  const response = await fetch(`${API_BASE_URL}/v1/fetcher/stocks/${code}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(`Failed to get stock: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data
}

export async function getStockData(code: string): Promise<{ stock: Stock; latestPrice: DailyPrice | null; latestFlow: InvestorFlow | null }> {
  const response = await fetch(`${API_BASE_URL}/v1/fetcher/stocks/${code}/data`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get stock data: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data
}

export async function getPriceHistory(code: string, startDate?: string, endDate?: string): Promise<DailyPrice[]> {
  let url = `${API_BASE_URL}/v1/fetcher/prices/${code}/history`
  const params = new URLSearchParams()
  if (startDate) params.append('start_date', startDate)
  if (endDate) params.append('end_date', endDate)
  if (params.toString()) url += `?${params.toString()}`

  const response = await fetch(url, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get price history: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data || []
}

export async function getFlowHistory(code: string, startDate?: string, endDate?: string): Promise<InvestorFlow[]> {
  let url = `${API_BASE_URL}/v1/fetcher/flows/${code}/history`
  const params = new URLSearchParams()
  if (startDate) params.append('start_date', startDate)
  if (endDate) params.append('end_date', endDate)
  if (params.toString()) url += `?${params.toString()}`

  const response = await fetch(url, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get flow history: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data || []
}

export async function searchStocks(query: string): Promise<Stock[]> {
  const response = await fetch(`${API_BASE_URL}/v1/stocks/search?q=${encodeURIComponent(query)}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to search stocks: ${response.statusText}`)
  }

  const data = await response.json()
  return data.results || []
}

// =====================================
// Signals API
// =====================================

export interface SignalSnapshot {
  id: string
  calc_date: string
  total_count: number
  buy_count: number
  sell_count: number
  created_at: string
}

export interface FactorScore {
  stock_code: string
  stock_name?: string
  calc_date: string
  momentum: number
  technical: number
  value: number
  quality: number
  flow: number
  event: number
  total_score: number
}

export interface Signal {
  stock_code: string
  stock_name?: string
  signal_type: 'BUY' | 'SELL'
  total_score: number
  momentum: number
  technical: number
  value: number
  quality: number
  flow: number
  event: number
  current_price?: number
  change_rate?: number
}

export async function getLatestSignalSnapshot(): Promise<SignalSnapshot | null> {
  const response = await fetch(`${API_BASE_URL}/v1/signals/snapshot/latest`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(`Failed to get latest signal snapshot: ${response.statusText}`)
  }

  const data = await response.json()
  return data // Backend returns data directly, not wrapped in {data: ...}
}

export async function getBuySignals(snapshotId: string): Promise<Signal[]> {
  const response = await fetch(`${API_BASE_URL}/v1/signals/snapshot/${snapshotId}/buy`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get buy signals: ${response.statusText}`)
  }

  const data = await response.json()
  const signals = data.signals || []

  // Map backend response to frontend Signal type
  return signals.map((s: any) => ({
    stock_code: s.symbol,
    stock_name: s.name,
    signal_type: s.signal_type,
    total_score: (s.strength || 0) / 100, // Convert 0-100 to 0-1
    momentum: (s.factors?.momentum?.score || 0) / 100,
    technical: (s.factors?.technical?.score || 0) / 100,
    value: (s.factors?.value?.score || 0) / 100,
    quality: (s.factors?.quality?.score || 0) / 100,
    flow: (s.factors?.flow?.score || 0) / 100,
    event: (s.factors?.event?.score || 0) / 100,
    current_price: s.current_price,
    change_rate: s.change_rate,
  }))
}

export async function getSellSignals(snapshotId: string): Promise<Signal[]> {
  const response = await fetch(`${API_BASE_URL}/v1/signals/snapshot/${snapshotId}/sell`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get sell signals: ${response.statusText}`)
  }

  const data = await response.json()
  const signals = data.signals || []

  // Map backend response to frontend Signal type
  return signals.map((s: any) => ({
    stock_code: s.symbol,
    stock_name: s.name,
    signal_type: s.signal_type,
    total_score: (s.strength || 0) / 100, // Convert 0-100 to 0-1
    momentum: (s.factors?.momentum?.score || 0) / 100,
    technical: (s.factors?.technical?.score || 0) / 100,
    value: (s.factors?.value?.score || 0) / 100,
    quality: (s.factors?.quality?.score || 0) / 100,
    flow: (s.factors?.flow?.score || 0) / 100,
    event: (s.factors?.event?.score || 0) / 100,
    current_price: s.current_price,
    change_rate: s.change_rate,
  }))
}

export async function getFactors(symbol: string): Promise<FactorScore | null> {
  const response = await fetch(`${API_BASE_URL}/v1/signals/factors/${symbol}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(`Failed to get factors: ${response.statusText}`)
  }

  const data = await response.json()
  return data.data
}

// ============================================================================
// Fetcher API
// ============================================================================

export interface TableStat {
  name: string
  display_name: string
  count: number
  last_update: string
  status: string
}

export interface TableStatsResponse {
  tables: TableStat[]
}

export interface FetchLog {
  id: number
  job_type: string
  source: string
  target_table: string
  records_fetched: number
  records_inserted: number
  records_updated: number
  status: string
  error_message?: string
  started_at: string
  finished_at?: string
  duration_ms: number
}

export interface FetchLogsResponse {
  logs: FetchLog[]
}

export async function getTableStats(): Promise<TableStatsResponse> {
  const response = await fetch(`${API_BASE_URL}/v1/fetcher/tables/stats`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get table stats: ${response.statusText}`)
  }

  return await response.json()
}

export async function getFetchLogs(): Promise<FetchLogsResponse> {
  const response = await fetch(`${API_BASE_URL}/v1/fetcher/execution-logs`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get fetch logs: ${response.statusText}`)
  }

  return await response.json()
}

export interface ScheduleInfo {
  collector_type: string
  display_name: string
  interval: string
  interval_sec: number
}

export interface SchedulesResponse {
  schedules: ScheduleInfo[]
}

export async function getSchedules(): Promise<SchedulesResponse> {
  const response = await fetch(`${API_BASE_URL}/v1/fetcher/schedules`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get schedules: ${response.statusText}`)
  }

  return await response.json()
}

// =====================================
// Stock Rankings API
// =====================================

export interface RankingStock {
  rank: number
  stock_code: string
  stock_name: string
  market: string
  current_price?: number
  change_rate?: number
  volume?: number
  trading_value?: number
  high_price?: number
  low_price?: number
  market_cap?: number
  foreign_net_value?: number
  inst_net_value?: number
  volume_surge_rate?: number
  high_52week?: number
}

export interface RankingResponse {
  category: string
  updated_at: string
  stocks: RankingStock[]
  total_count: number
}

export type RankingCategory = 'volume' | 'trading_value' | 'gainers' | 'losers' | 'foreign_net_buy' | 'inst_net_buy' | 'volume_surge' | 'high_52week' | 'market_cap'
export type MarketFilter = 'ALL' | 'KOSPI' | 'KOSDAQ'

export async function getStockRanking(category: RankingCategory, limit = 20, market: MarketFilter = 'ALL'): Promise<RankingResponse> {
  const categoryMap = {
    volume: 'volume',
    trading_value: 'trading-value',
    gainers: 'gainers',
    losers: 'losers',
    foreign_net_buy: 'foreign-net-buy',
    inst_net_buy: 'inst-net-buy',
    volume_surge: 'volume-surge',
    high_52week: 'high-52week',
    market_cap: 'market-cap',
  }

  const endpoint = categoryMap[category]
  const params = new URLSearchParams({ limit: limit.toString() })
  if (market !== 'ALL') {
    params.append('market', market)
  }

  const response = await fetch(`${API_BASE_URL}/v1/rankings/${endpoint}?${params.toString()}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get ${category} ranking: ${response.statusText}`)
  }

  return await response.json()
}

// =====================================
// KIS Audit Builder API
// =====================================

export interface BuildFromKISRequest {
  account_no?: string
  account_product_code?: string
  start_date?: string // YYYY-MM-DD
  end_date?: string   // YYYY-MM-DD
}

export async function buildAuditFromKIS(params?: BuildFromKISRequest): Promise<void> {
  const searchParams = new URLSearchParams()
  if (params?.account_no) searchParams.append('account_no', params.account_no)
  if (params?.account_product_code) searchParams.append('account_product_code', params.account_product_code)
  if (params?.start_date) searchParams.append('start_date', params.start_date)
  if (params?.end_date) searchParams.append('end_date', params.end_date)

  const url = searchParams.toString()
    ? `${API_BASE_URL}/v1/audit/build-from-kis?${searchParams.toString()}`
    : `${API_BASE_URL}/v1/audit/build-from-kis`

  console.log('[API] buildAuditFromKIS request:', url, params)

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    const errorText = await response.text()
    console.error('[API] buildAuditFromKIS error:', response.status, errorText)
    throw new Error(errorText || response.statusText)
  }

  const result = await response.json()
  console.log('[API] buildAuditFromKIS success:', result)
}

// =====================================
// Stock Info API (Company Overview)
// =====================================

export interface StockInfoDetail {
  symbol: string
  symbol_name?: string
  company_overview?: string
  overview_source?: string
}

/**
 * 종목의 기업 개요 조회
 * DB에 없으면 네이버증권에서 자동으로 가져와서 저장 후 반환
 */
export async function getStockInfo(symbol: string): Promise<StockInfoDetail> {
  const response = await fetch(`${API_BASE_URL}/stocks/${symbol}/info`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to get stock info: ${response.statusText}`)
  }

  return await response.json()
}

/**
 * 종목의 기업 개요 강제 새로고침 (네이버에서 다시 가져오기)
 */
export async function refreshStockInfo(symbol: string): Promise<StockInfoDetail> {
  const response = await fetch(`${API_BASE_URL}/stocks/${symbol}/info/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`Failed to refresh stock info: ${response.statusText}`)
  }

  return await response.json()
}
