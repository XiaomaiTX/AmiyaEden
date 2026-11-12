import type { CommonSearchParams } from '@/types/api/common'

export type BlacklistKind = 'character' | 'qq' | 'discord'

export interface BlacklistEntry {
  kind: BlacklistKind
  value: string
  display: string
  reason: string
  updated_at: string
}

export type BlacklistListParams = CommonSearchParams & {
  kind?: BlacklistKind
  keyword?: string
}

export interface BlacklistRefreshSummary {
  finished_at: string
  operator_user_id: number
  operator_name: string
  characters: number
  qqs: number
  discords: number
  added: number
  updated: number
  removed: number
  new_unresolved: number
  skipped_unresolved: number
  lookup_failed: number
  next_refresh_at: string | null
}
