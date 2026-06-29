export interface NpcKillRequest {
  character_id: number
  start_date?: string
  end_date?: string
  ref_types?: string[]
  solar_system_ids?: number[]
  character_ids?: number[]
  user_ids?: number[]
  min_amount?: number
  max_amount?: number
}

export interface NpcKillAllRequest {
  start_date?: string
  end_date?: string
  ref_types?: string[]
  solar_system_ids?: number[]
  character_ids?: number[]
  user_ids?: number[]
  min_amount?: number
  max_amount?: number
}

export interface NpcKillCorpRequest {
  start_date?: string
  end_date?: string
  corp_tickers?: string
  ref_types?: string[]
  solar_system_ids?: number[]
  character_ids?: number[]
  user_ids?: number[]
  min_amount?: number
  max_amount?: number
}

export interface NpcKillSummary {
  total_bounty: number
  total_ess: number
  total_incursion: number
  total_mission: number
  total_tax: number
  actual_income: number
  total_records: number
}

export interface NpcKillMemberItem {
  user_id: number
  display_name: string
  character_count: number
  total_bounty: number
  total_ess: number
  total_incursion: number
  total_mission: number
  total_tax: number
  actual_income: number
  record_count: number
}

export interface NpcKillSystemItem {
  solar_system_id: number
  solar_system_name: string
  count: number
  amount: number
}

export interface NpcKillTrendItem {
  date: string
  amount: number
  count: number
}

export interface NpcKillJournalItem {
  id: number
  date: string
  ref_type: string
  amount: number
  tax: number
  solar_system_name: string
  character_name: string
  reason: string
}

export interface NpcKillResponse {
  summary: NpcKillSummary
  by_npc: Array<{
    npc_id: number
    npc_name: string
    count: number
  }>
  by_system: NpcKillSystemItem[]
  trend: NpcKillTrendItem[]
  journals: NpcKillJournalItem[]
}

export interface NpcKillCorpResponse {
  summary: NpcKillSummary
  members: NpcKillMemberItem[]
  by_system: NpcKillSystemItem[]
  trend: NpcKillTrendItem[]
}

