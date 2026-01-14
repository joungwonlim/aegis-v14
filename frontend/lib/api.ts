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
  updated_ts: string
  exit_mode: string // ENABLED, DISABLED, MANUAL_ONLY
  raw?: {
    symbol_name?: string
    evaluate_amount?: string
    purchase_amount?: string
    evaluate_profit_loss?: string
    evaluate_profit_loss_rate?: string
  }
}

export interface OrderIntent {
  intent_id: string
  position_id: string
  symbol: string
  intent_type: string // EXIT_PARTIAL, EXIT_FULL
  qty: number
  order_type: string // MKT, LMT
  limit_price?: number
  reason_code: string // SL1, SL2, TP1, TP2, TRAILING, etc.
  status: string // PENDING_APPROVAL, NEW, ACK, REJECTED, FILLED, CANCELLED
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

  return response.json()
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

export async function approveIntent(intentId: string): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE_URL}/intents/${intentId}/approve`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error(`Failed to approve intent: ${response.statusText}`)
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

export async function updateExitMode(accountId: string, symbol: string, exitMode: string): Promise<{ status: string, exit_mode: string }> {
  const response = await fetch(`${API_BASE_URL}/holdings/${accountId}/${symbol}/exit-mode`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ exit_mode: exitMode }),
  })

  if (!response.ok) {
    throw new Error(`Failed to update exit mode: ${response.statusText}`)
  }

  return response.json()
}
