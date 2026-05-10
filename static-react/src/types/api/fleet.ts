import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export type FleetImportance = 'strat_op' | 'cta' | 'other'

export type FleetAutoSrpMode = 'disabled' | 'submit_only' | 'auto_approve'

export interface FleetItem {
  id: string
  title: string
  description: string
  start_at: string
  end_at: string
  importance: FleetImportance
  pap_count: number
  fc_user_id: number
  fc_character_id: number
  fc_character_name: string
  fc_display_name?: string
  esi_fleet_id: number | null
  fleet_config_id: number | null
  auto_srp_mode: FleetAutoSrpMode
  created_at: string
  updated_at: string
}

export type FleetList = PaginatedResponse<FleetItem>

export interface FleetPapLog {
  id: number
  fleet_id: string
  fleet_title?: string
  character_id: number
  character_name?: string
  user_id: number
  pap_count: number
  issued_by: number
  created_at: string
  fleet_start_at: string
  fc_character_name: string
  fleet_importance: string
  ship_type_id: number | null
}

export interface FleetMember {
  id: number
  fleet_id: string
  character_id: number
  character_name: string
  user_id: number
  ship_type_id: number | null
  solar_system_id: number | null
  joined_at: string
  created_at: string
}

export interface MemberWithPap extends FleetMember {
  pap_count: number | null
  issued_at: string | null
}

export interface JoinFleetParams {
  code: string
  character_id: number
}

export interface FleetSearchParams extends Partial<CommonSearchParams> {
  importance?: string
  fc_user_id?: number
}

export interface CreateFleetParams {
  title: string
  description?: string
  start_at: string
  end_at: string
  importance: FleetImportance
  pap_count: number
  character_id: number
  fleet_config_id?: number | null
  send_ping?: boolean
  auto_srp_mode?: FleetAutoSrpMode
}

export interface UpdateFleetParams {
  title?: string
  description?: string
  start_at?: string
  end_at?: string
  importance?: FleetImportance | string
  pap_count?: number
  character_id?: number
  esi_fleet_id?: number
  fleet_config_id?: number | null
  auto_srp_mode?: FleetAutoSrpMode
}

export interface ManualAddFleetMembersParams {
  character_names: string[]
}

export interface ManualAddFleetMembersResult {
  added_character_names: string[]
  missing_character_names: string[]
}

export type PapSummaryPeriod = 'current_month' | 'last_month' | 'at_year' | 'all'

export interface CorporationPapSummaryParams extends Partial<CommonSearchParams> {
  period?: PapSummaryPeriod
  year?: number
  corp_tickers?: string
}

export interface CorporationPapSummaryItem {
  user_id: number
  nickname: string
  corp_ticker: string
  main_character_name: string
  character_count: number
  strat_op_paps: number
  skirmish_paps: number
  alliance_strat_paps: number
}

export interface CorporationPapOverview {
  filtered_pap_total: number
  filtered_strat_op_total: number
  all_pap_total: number
  filtered_user_count: number
  period: PapSummaryPeriod
  year?: number
}

export interface CorporationPapSummaryList extends PaginatedResponse<CorporationPapSummaryItem> {
  overview: CorporationPapOverview
}

export interface FleetInvite {
  id: number
  fleet_id: string
  code: string
  active: boolean
  expires_at: string
  created_at: string
}

export interface CharacterFleetInfo {
  fleet_id: number
  fleet_boss_id: number
  role: string
  squad_id: number
  wing_id: number
}

export interface ESIFleetMember {
  character_id: number
  join_time: string
  role: string
  role_name: string
  ship_type_id: number
  solar_system_id: number
  squad_id: number
  wing_id: number
}
