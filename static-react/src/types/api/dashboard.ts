import type { CommonSearchParams, SnakePaginatedResponse } from '@/types/api/common'

export interface DashboardResult {
  cards: {
    online_count: number
    total_assets_count: number
    total_assets_price: number
    my_pap_count: number
  }
  fleets: unknown[]
  srp_list: unknown[]
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
}

export interface CorporationStructureAuthorizationUpdate {
  authorizations: Array<{
    corporation_id: number
    character_id: number
  }>
  fuel_notice_threshold_days: number
  timer_notice_threshold_days: number
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
