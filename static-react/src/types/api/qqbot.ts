export interface QQBotConfig {
  enabled: boolean
  bot_id: number
  cidrs: string
  interval_ms: number
  token?: string
}
export interface QQBotOperation {
  id: string
  business_key: string
  request: string
  kind: string
  class: string
  status: string
  priority: number
  expires_at: string
  available_at: string
  lease_until: string
  failures: number
  cancel_requested: boolean
  wait_reason: string
  result: string
  created_at: string
  updated_at: string
  terminal_at: string | null
}
export interface QQBotAttempt {
  id: string
  operation_id: string
  classification: string
  started_at: string
  finished_at: string | null
}
export interface QQBotDelivery {
  id: string
  topic: string
  item_id: string
  consumer: string
  status: string
  attempts: number
  available_at: string
  lease_until: string
  last_error: string
  created_at: string
  updated_at: string
}
export interface QQBotCircuit {
  class: string
  state: string
  failures: number
  successes: number
  errors: number
  probe_at: string
  paused: boolean
}
export interface QQBotSnapshot {
  redis_available: boolean
  rate_wait_ms: number
  connected: boolean
  last_frame: string
  ingress_error: string
  operations: QQBotOperation[]
  attempts: QQBotAttempt[]
  events: { id: string; payload: string; created_at: string }[]
  deliveries: QQBotDelivery[]
  circuits: QQBotCircuit[]
}
