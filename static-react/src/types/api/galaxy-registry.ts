import type { PaginatedResponse } from '@/types/api/common'

export type GalaxyRegistrySystemStatus = 'idle' | 'busy' | 'overdue'
export type GalaxyRegistryEntryStatus = 'active' | 'completed'
export type GalaxyRegistryValidationStatus = 'pending' | 'valid' | 'violation'

export interface GalaxyRegistrySystemActiveEntry {
  entry_id: number
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  expected_end_at: string
  actual_start_at: string
  is_overdue: boolean
  is_mine: boolean
}

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
  active_entry: GalaxyRegistrySystemActiveEntry | null
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
  created_at: string
  updated_at: string
}

export interface GalaxyRegistrySdeSystem {
  solar_system_id: number
  solar_system_name: string
  region_id: number
  region_name: string
  constellation_id: number
  constellation_name: string
  security: number
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
  created_at: string
  updated_at: string
}

export interface GalaxyRegistryAdminAnalytics {
  range_start: string
  range_end: string
  current_snapshot: { idle_count: number; busy_count: number; overdue_count: number }
  recent_7d: { entry_count: number; valid_count: number; violation_count: number; pending_count: number; valid_rate: number }
  recent_30d: { entry_count: number; valid_count: number; violation_count: number; pending_count: number; valid_rate: number }
  top_systems: Array<{ system_config_id: number; solar_system_id: number; solar_system_name: string; register_count: number }>
  recent_violations: GalaxyRegistryEntryItem[]
}

export type GalaxyRegistryEntryPage = PaginatedResponse<GalaxyRegistryEntryItem>
export type GalaxyRegistryEntryListParams = Partial<{
  current: number; size: number; system_config_id: number; keyword: string
  status: GalaxyRegistryEntryStatus | ''; validation_status: GalaxyRegistryValidationStatus | ''
  start_date: string; end_date: string
}>
export interface GalaxyRegistryCreateEntryRequest { system_config_id: number; expected_end_at: string }
export interface GalaxyRegistryUpdateExpectedEndAtRequest { expected_end_at: string }
export interface GalaxyRegistryAdminCreateSystemRequest { solar_system_id: number; note?: string; min_bounty_amount?: number; is_enabled?: boolean }
export interface GalaxyRegistryAdminUpdateSystemRequest { note?: string; min_bounty_amount?: number; is_enabled?: boolean }
export interface GalaxyRegistryAdminUpdateValidationRequest { validation_status: 'valid' | 'violation'; violation_reason?: string }
