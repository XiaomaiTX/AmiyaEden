import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export type FleetImportance = 'strat_op' | 'cta' | 'other'

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
  character_name_full?: string
  fleet_title_full?: string
  fleet_start_at: string
  fc_character_name: string
  fleet_importance: string
  ship_type_id: number | null
}

export interface JoinFleetParams {
  code: string
  character_id: number
}

export interface FleetSearchParams extends Partial<CommonSearchParams> {
  importance?: string
  fc_user_id?: number
}

export type FleetListResponse = PaginatedResponse<unknown>
