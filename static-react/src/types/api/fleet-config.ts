import type { PaginatedResponse } from '@/types/api/common'

export interface FittingItem {
  id: number
  fleet_config_id: number
  ship_type_id: number
  fitting_name: string
  srp_amount: number
}

export interface FleetConfigItem {
  id: number
  name: string
  description: string
  created_by: number
  created_at: string
  updated_at: string
  fittings: FittingItem[]
}

export type FleetConfigList = PaginatedResponse<FleetConfigItem>

export interface FittingReq {
  id?: number
  fitting_name: string
  eft: string
  srp_amount: number
}

export interface CreateFleetConfigParams {
  name: string
  description?: string
  fittings: FittingReq[]
}

export interface UpdateFleetConfigParams {
  name?: string
  description?: string
  fittings?: FittingReq[]
}

export interface ImportFittingParams {
  character_id: number
  fitting_id: number
}

export interface ImportFittingResponse {
  fitting_name: string
  eft: string
  srp_amount: number
}

export interface ExportToESIParams {
  character_id: number
  fleet_config_id: number
  fitting_item_id: number
}

export interface EFTFittingItem {
  id: number
  eft: string
}

export interface EFTResponse {
  fittings: EFTFittingItem[]
}

export interface FittingItemReplacement {
  id: number
  type_id: number
  type_name: string
}

export interface FittingItemDetail {
  id: number
  type_id: number
  type_name: string
  quantity: number
  flag: string
  flag_group: string
  importance: 'required' | 'optional' | 'replaceable'
  penalty: 'none' | 'half'
  replacement_penalty: 'none' | 'half'
  replacements: FittingItemReplacement[]
}

export interface FittingItemsResponse {
  fitting_id: number
  fitting_name: string
  ship_type_id: number
  items: FittingItemDetail[]
}

export interface ItemSettingUpdate {
  id: number
  importance: 'required' | 'optional' | 'replaceable'
  penalty: 'none' | 'half'
  replacement_penalty: 'none' | 'half'
  replacements?: number[]
}

export interface UpdateItemsSettingsParams {
  items: ItemSettingUpdate[]
}
