import type { CommonSearchParams, SnakePaginatedResponse } from '@/types/api/common'

export interface DashboardCards {
  eve_wallet_balance: number
  eve_skill_points: number
  system_wallet_balance: number
  alliance_pap: number
}

export type DashboardFleetSource = 'internal' | 'alliance'

export interface DashboardFleetItem {
  source: DashboardFleetSource
  id: string
  title: string
  start_at: string
  end_at?: string
  importance?: string
  pap_count: number
  ship_type_name?: string
  character_name?: string
}

export interface DashboardPapMonthly {
  year: number
  month: number
  total_pap: number
}

export interface DashboardPapStats {
  alliance: DashboardPapMonthly[]
  internal: DashboardPapMonthly[]
}

export interface DashboardSrpItem {
  id: number
  character_name: string
  ship_name: string
  solar_system_name: string
  killmail_time: string
  recommended_amount: number
  final_amount: number
  review_status: string
  payout_status: string
  created_at: string
}

export interface DashboardResult {
  cards: DashboardCards
  fleets: DashboardFleetItem[]
  pap_stats: DashboardPapStats
  srp_list: DashboardSrpItem[]
}

export interface CorporationStructureCorporationDirectorCharacter {
  character_id: number
  character_name: string
}

export interface CorporationStructureCorporation {
  corporation_id: number
  corporation_name: string
  authorized_character_id: number
  director_characters: CorporationStructureCorporationDirectorCharacter[]
}

export interface CorporationStructuresSettings {
  corporations: CorporationStructureCorporation[]
  fuel_notice_threshold_days: number
  timer_notice_threshold_days: number
  alert_enabled: boolean
  alert_group_ids: number[]
}

export interface CorporationStructureAuthorizationUpdate {
  authorizations: Array<{
    corporation_id: number
    character_id: number
  }>
  fuel_notice_threshold_days: number
  timer_notice_threshold_days: number
  alert_enabled: boolean
  alert_group_ids: number[]
}

export interface CorporationStructureSystemOption {
  system_id: number
  system_name: string
  region_name: string
  security: number
}

export interface CorporationStructureTypeOption {
  type_id: number
  type_name: string
}

export interface CorporationStructureServiceInfo {
  name: string
  state: string
}

export interface CorporationStructureFilterOptionsResponse {
  systems: CorporationStructureSystemOption[]
  types: CorporationStructureTypeOption[]
  services: CorporationStructureServiceInfo[]
}

export interface CorporationStructureRow {
  structure_id: number
  corporation_name: string
  assigned_user_id: number
  assigned_character_id: number
  assigned_character_name: string
  state: string
  system_name: string
  region_name: string
  security: number
  name: string
  type_name: string
  services: CorporationStructureServiceInfo[]
  fuel_remaining: string
  reinforce_hour: number
  state_timer_end: string
  updated_at: number
  fuel_per_hour?: number | null
  fuel_to_month_end?: number | null
  fuel_estimate_incomplete?: boolean
  fuel_unknown_services?: string[]
  fuel_estimate_status?: 'available' | 'authorization_required' | 'activity_mapping_required' | 'module_mismatch' | 'rate_unavailable' | 'ambiguous_module'
}

export interface StructureServiceModuleCatalogItem {
  service_name: string
  type_id: number
  type_name: string
  fuel_per_hour: number
  fuel_category: 'other' | 'citadel' | 'engineering_complex' | 'refinery'
}

export interface StructureServiceActivityCatalogItem {
  activity_name: string
  type_ids: number[]
  system_managed: boolean
}

export interface StructureServicePendingActivity {
  activity_name: string
  structure_id: number
  structure_name: string
  installed_module_type_ids: number[]
}

export interface StructureServiceCatalog {
  modules: StructureServiceModuleCatalogItem[]
  activities: StructureServiceActivityCatalogItem[]
  unmapped_activities: StructureServicePendingActivity[]
}

export interface StructureServiceCatalogUpdate {
  modules: Array<Pick<StructureServiceModuleCatalogItem, 'service_name' | 'type_id' | 'fuel_category'>>
  activities: Array<Pick<StructureServiceActivityCatalogItem, 'activity_name' | 'type_ids'>>
}

export interface CorporationStructureListRequest extends Partial<CommonSearchParams> {
  corporation_id?: number
  keyword?: string
  state_groups?: string[]
  fuel_bucket?: 'all' | 'lt_24h' | 'lt_72h' | 'lt_168h' | 'custom'
  fuel_min_hours?: number
  fuel_max_hours?: number
  system_ids?: number[]
  security_bands?: Array<'highsec' | 'lowsec' | 'nullsec'>
  security_min?: number
  security_max?: number
  type_ids?: number[]
  service_names?: string[]
  service_match_mode?: 'and' | 'or'
  timer_bucket?: 'all' | 'current_hour' | 'next_2_hours' | 'custom'
  timer_start?: string
  timer_end?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export type CorporationStructureListResponse = SnakePaginatedResponse<CorporationStructureRow>

export interface CorporationStructureFilterOptionsRequest {
  corporation_id?: number
}

export interface CorporationStructureRunTaskRequest {
  corporation_id: number
}

export interface CorporationStructureRunTaskResponse {
  message: string
}
