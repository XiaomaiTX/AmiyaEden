import type { CommonSearchParams, PaginatedResponse } from '@/types/api/common'

export type PayoutMode = 'manual_transfer' | 'fuxi_coin'

export interface SrpConfig {
  amount_limit: number
}

export interface ShipPrice {
  id: number
  ship_type_id: number
  ship_name: string
  amount: number
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface UpsertShipPriceParams {
  id?: number
  ship_type_id: number
  ship_name: string
  amount: number
}

export interface Application {
  id: number
  user_id: number
  nickname?: string
  last_actor_nickname?: string
  character_id: number
  character_name: string
  killmail_id: number
  fleet_id: string | null
  note: string
  ship_type_id: number
  ship_name: string
  solar_system_id: number
  solar_system_name: string
  killmail_time: string
  corporation_id: number
  corporation_name: string
  alliance_id: number
  alliance_name: string
  recommended_amount: number
  final_amount: number
  review_status: 'submitted' | 'approved' | 'rejected'
  reviewed_by: number | null
  reviewed_at: string | null
  review_note: string
  payout_status: 'notpaid' | 'paid'
  paid_by: number | null
  paid_at: string | null
  created_at: string
  updated_at: string
  fleet_title?: string
  fleet_fc_name?: string
}

export type ApplicationList = PaginatedResponse<Application>

export interface SubmitApplicationParams {
  character_id: number
  killmail_id: number
  fleet_id?: string | null
  note?: string
}

export type ApplicationSearchParams = Partial<{
  fleet_id: string
  character_id: number
  review_status: string
  payout_status: string
  tab: string
  keyword: string
}> &
  Partial<CommonSearchParams>

export interface ReviewParams {
  action: 'approve' | 'reject'
  review_note?: string
  final_amount?: number
}

export interface PayoutParams {
  final_amount?: number
  mode?: PayoutMode
}

export interface BatchPayoutSummary {
  user_id: number
  nickname?: string
  main_character_id: number
  main_character_name: string
  total_amount: number
  application_count: number
}

export interface BatchFuxiPayoutSummary {
  application_count: number
  user_count: number
  total_isk_amount: number
  total_fuxi_coin: number
}

export interface BatchPayoutActionResult extends BatchPayoutSummary {
  message?: string
}

export interface BatchFuxiPayoutActionResult extends BatchFuxiPayoutSummary {
  message?: string
}

export interface AutoApproveParams {
  fleet_id: string
}

export interface AutoApproveSummary {
  checked_count: number
  approved_count: number
  skipped_count: number
}

export interface FleetKillmailItem {
  killmail_id: number
  killmail_time: string
  ship_type_id: number
  solar_system_id: number
  character_id: number
  victim_name: string
}

export interface KillmailListParams {
  character_id?: number
  limit?: number
  exclude_submitted?: boolean
}

export interface KillmailDetailRequest {
  killmail_id: number
  language?: string
}

export interface KillmailSlotItem {
  item_id: number
  item_name: string
  quantity: number
  dropped: boolean
}

export interface KillmailSlotGroup {
  flag_id: number
  flag_name: string
  flag_text: string
  order_id: number
  items: KillmailSlotItem[]
}

export interface KillmailDetailResponse {
  killmail_id: number
  killmail_time: string
  ship_type_id: number
  ship_name: string
  solar_system_id: number
  system_name: string
  character_id: number
  character_name: string
  janice_amount: number | null
  slots: KillmailSlotGroup[]
}

export type ShipPriceListResponse = ShipPrice[]
