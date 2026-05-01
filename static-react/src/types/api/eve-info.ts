import type { CommonSearchParams, PaginatedResponse } from '@/types/api/common'

export interface WalletRequest {
  character_id: number
  page: number
  page_size: number
  ref_types?: string[]
}

export interface WalletJournal {
  id: number
  date: string
  ref_type: string
  amount: number
  balance: number
  description: string
}

export interface WalletResponse {
  balance: number
  ref_types: string[]
  journals: WalletJournal[]
}

export interface SkillRequest {
  character_id: number
  language: string
}

export interface SkillItem {
  skill_id: number
  skill_name: string
  group_name: string
  active_level: number
  trained_level: number
}

export interface SkillQueueItem {
  queue_position: number
  skill_name: string
  finished_level: number
}

export interface SkillResponse {
  skills: SkillItem[]
  skill_queue: SkillQueueItem[]
}

export interface ShipRequest {
  character_id: number
  language: string
}

export interface ShipItem {
  type_id: number
  type_name: string
  group_name: string
  race_name: string
  can_fly: boolean
}

export interface ShipResponse {
  total_ships: number
  flyable_ships: number
  ships: ShipItem[]
}

export interface ImplantsRequest {
  character_id: number
  language: string
}

export interface ImplantItem {
  implant_id: number
  implant_name: string
}

export interface JumpCloneLocation {
  location_name: string
  location_type: string
  location_id: number
}

export interface JumpClone {
  jump_clone_id: number
  location: JumpCloneLocation
  implants: ImplantItem[]
}

export interface ImplantsResponse {
  jump_fatigue_expire: string | null
  last_jump_date: string | null
  last_clone_jump_date: string | null
  active_implants: ImplantItem[]
  jump_clones: JumpClone[]
}

export interface AssetsRequest {
  language: string
}

export interface AssetItemNode {
  item_id: number
  type_id: number
  type_name: string
  group_name: string
  asset_name?: string
  category_id: number
  is_blueprint_copy?: boolean
  quantity: number
  character_name: string
  children?: AssetItemNode[]
}

export interface AssetLocationNode {
  location_id: number
  location_name: string
  items: AssetItemNode[]
}

export interface AssetsResponse {
  locations: AssetLocationNode[]
}

export interface ContractsRequest extends Partial<CommonSearchParams> {
  type?: string
  status?: string
  language: string
}

export interface ContractItem {
  contract_id: number
  type: string
  status: string
  title?: string
  for_corporation: boolean
  price?: number | null
  reward?: number | null
  character_name: string
  character_id: number
  date_expired?: string
  date_issued?: string
}

export interface ContractItemDetail {
  type_id: number
  type_name: string
  group_name: string
  quantity: number
  is_included: boolean
}

export interface ContractBidItem {
  bid_id: number
  amount: number
  bidder_id: number
  date_bid: string
}

export interface ContractDetailRequest {
  character_id: number
  contract_id: number
  language: string
}

export interface ContractDetailResponse {
  items: ContractItemDetail[]
  bids: ContractBidItem[]
}

export type ContractsResponse = PaginatedResponse<ContractItem>

export interface FittingsRequest {
  language: string
}

export interface FittingSlotItem {
  type_id: number
  type_name: string
  quantity: number
}

export interface FittingSlot {
  flag_name: string
  flag_text: string
  items: FittingSlotItem[]
}

export interface FittingResponse {
  fitting_id: number
  character_id: number
  race_id: number
  race_name: string
  group_name: string
  name: string
  ship_name: string
  slots: FittingSlot[]
}

export interface FittingsListResponse {
  fittings: FittingResponse[]
}

export interface RunTaskParams {
  task_name: string
  character_id: number
}
