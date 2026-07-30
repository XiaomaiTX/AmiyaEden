import type { PaginatedResponse } from '@/types/api/common'

export type GalaxyRegistrySystemStatus = 'idle' | 'busy' | 'overdue'
export type GalaxyRegistryEntryStatus = 'active' | 'completed'
export type GalaxyRegistryValidationStatus = 'pending' | 'valid' | 'violation'

export interface GalaxyRegistrySystemItem {
  system_config_id: number
  solar_system_id: number
  solar_system_name: string
  region_name: string
  constellation_name: string
  security: number
  note: string
  min_bounty_amount: number
  is_enabled: boolean
  status: GalaxyRegistrySystemStatus
  active_entry: {
    entry_id: number
    captain_character_name: string
    captain_nickname: string
    expected_end_at: string
    actual_start_at: string
    is_overdue: boolean
    is_mine: boolean
  } | null
}

export interface GalaxyRegistrySystemsResponse {
  summary: { idle_count: number; busy_count: number; overdue_count: number }
  items: GalaxyRegistrySystemItem[]
}

export interface GalaxyRegistryEntryItem {
  id: number
  system_config_id: number
  solar_system_id: number
  solar_system_name: string
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  status: GalaxyRegistryEntryStatus
  validation_status: GalaxyRegistryValidationStatus
  expected_end_at: string
  actual_start_at: string
  actual_end_at: string | null
  force_ended_by_admin: boolean
  frozen_min_bounty_amount: number
  validated_at: string | null
  validated_bounty_amount: number
  validated_bounty_count: number
  violation_reason: string
}

export interface GalaxyRegistryAdminSystem {
  id: number
  solar_system_id: number
  solar_system_name: string
  region_name: string
  constellation_name: string
  security: number
  note: string
  min_bounty_amount: number
  is_enabled: boolean
}

export type GalaxyRegistryEntryPage = PaginatedResponse<GalaxyRegistryEntryItem>
export type GalaxyRegistryEntryListParams = Partial<{
  current: number
  size: number
  keyword: string
  status: GalaxyRegistryEntryStatus | ''
  validation_status: GalaxyRegistryValidationStatus | ''
}>
