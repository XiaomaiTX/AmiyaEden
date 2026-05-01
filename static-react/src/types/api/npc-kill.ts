export interface NpcKillRequest {
  character_id: number
  start_date?: string
  end_date?: string
}

export interface NpcKillAllRequest {
  start_date?: string
  end_date?: string
}

export interface NpcKillCorpRequest {
  start_date?: string
  end_date?: string
}

export interface NpcKillSummary {
  total_bounty: number
  total_tax: number
  actual_income: number
  total_records: number
  estimated_hours: number
}

export interface NpcKillMemberItem {
  character_id: number
  character_name: string
  total_bounty: number
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

