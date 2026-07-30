export interface QQPageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export type QQPageParams = Partial<{
  page: number
  page_size: number
  group_id: number
  qq: number
  status: string
  decision: string
  action_type: string
}>

export interface QQPolicy {
  id: number
  group_id: number
  enabled: boolean
  allowed_corporation_ids: number[]
  allowed_role_codes: string[]
  auto_reject_unmatched: boolean
  member_violation_policy: 'review_only' | 'auto_kick_after_confirmed_mismatch'
  card_template: string
  card_sync_enabled: boolean
}

export interface QQGroupStatus {
  group_id: number
  group_name: string
  enabled: boolean
  member_count: number
  max_member_count: number
  bot_is_admin: boolean | null
  valid_count: number
  review_count: number
  invalid_candidate_count: number
  invalid_confirmed_count: number
  snapshot_state: 'fresh' | 'stale' | 'never_synced'
  reconcile_run_status: string
  reconcile_expected: number
  reconcile_processed: number
  reconcile_failed: number
}

export interface QQActionTask {
  id: number
  action_type: string
  group_id: number
  qq: number
  status: string
  retry_count: number
  retry_cause: string
  last_error: string
  created_at: string
}

export interface QQAlert {
  id: number
  kind: string
  group_id: number
  qq: number
  status: string
  message: string
  created_at: string
}

export interface QQMetrics {
  window_minutes: number
  created: number
  succeeded: number
  failed: number
  dead: number
  failure_rate: number
  connected: boolean
  risk_level: number
}

export interface QQConnection {
  connected: boolean
  risk_level: number
}

export interface QQSettings {
  scan_interval_minutes: number
  mismatch_confirmations: number
  mismatch_observation_hours: number
}
