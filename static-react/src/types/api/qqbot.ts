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

export interface QQNotificationChannel {
  name: string
  enabled: boolean
  group_ids: string
  updated_at: string
}
export interface QQNotificationTarget {
  channel: string
  correlation_id: string
  business_key: string
  notification_id: string
  group_id: number
  operation_id: string
  status: 'pending' | 'confirmed' | 'failed' | 'expired' | 'unknown' | 'cancelled'
  result: string
  updated_at: string
}
export interface QQGroupRule {
  group_id: number
  version: number
  admission_enabled: boolean
  auto_reject: boolean
  corporation_ids: number[] | null
  roles: string[] | null
  card_enabled: boolean
  card_template: string
  inspection_enabled: boolean
  auto_kick: boolean
  mismatch_confirmations: number
  observation_hours: number
  last_inspection_at: string
  updated_at: string
}
export interface QQGroupApplication {
  id: string
  group_id: number
  qq: number
  status: string
  decision: string
  evidence: string
  operation_id: string
  rule_version: number
  expires_at: string
}
export interface QQGroupMember {
  group_id: number
  qq: number
  present: boolean
  epoch: number
  evidence: string
  confirmations: number
  mismatch_since: string | null
  card_target: string
  card_status: string
  card_operation_id: string
  kick_status: string
  kick_operation_id: string
}
export interface QQGroupInspection {
  id: string
  group_id: number
  rule_version: number
  status: string
  operation_id: string
  snapshot_at: string
}
export interface QQGroupJudgement {
  run_id: string
  qq: number
  present: boolean
  done: boolean
  decision: string
  evidence: string
  operation_id: string
}
export interface QQGroupBusinessAudit {
  rules: QQGroupRule[]
  applications: QQGroupApplication[]
  members: QQGroupMember[]
  inspections: QQGroupInspection[]
  judgements: QQGroupJudgement[]
}
