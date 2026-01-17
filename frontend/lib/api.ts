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
