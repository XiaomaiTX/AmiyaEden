import type { CommonSearchParams, PaginatedResponse } from '@/types/api/common'

export interface WelfareItem {
  id: number
  name: string
  description: string
  dist_mode: 'per_user' | 'per_character'
  pay_by_fuxi_coin: number | null
  require_skill_plan: boolean
  skill_plan_ids: number[]
  max_char_age_months: number | null
  minimum_pap: number | null
  minimum_fuxi_legion_years: number | null
  require_evidence: boolean
  example_evidence: string
  status: number
  sort_order: number
  created_by: number
  created_at: string
  updated_at: string
}

export interface CreateParams {
  name: string
  description?: string
  dist_mode: 'per_user' | 'per_character'
  pay_by_fuxi_coin?: number | null
  require_skill_plan?: boolean
  skill_plan_ids?: number[]
  max_char_age_months?: number | null
  minimum_pap?: number | null
  minimum_fuxi_legion_years?: number | null
  require_evidence?: boolean
  example_evidence?: string
  status?: number
  sort_order?: number
}

export interface UpdateParams extends CreateParams {
  id: number
}

export type SearchParams = Partial<{
  current: number
  size: number
  status: number
  name: string
}>

export interface AutoApproveConfig {
  auto_approve_fuxi_coin_threshold: number
}

export interface UpdateAutoApproveConfigParams {
  auto_approve_fuxi_coin_threshold: number
}

export interface EligibleCharacter {
  character_id: number
  character_name: string
  can_apply_now: boolean
  ineligible_reason?:
    | 'pap'
    | 'skill'
    | 'pap_skill'
    | 'legion_years'
    | 'pap_legion_years'
    | 'skill_legion_years'
    | 'pap_skill_legion_years'
}

export interface EligibleWelfare {
  id: number
  name: string
  description: string
  dist_mode: 'per_user' | 'per_character'
  skill_plan_names: string[]
  require_evidence: boolean
  example_evidence: string
  can_apply_now: boolean
  ineligible_reason?:
    | 'pap'
    | 'skill'
    | 'pap_skill'
    | 'legion_years'
    | 'pap_legion_years'
    | 'skill_legion_years'
    | 'pap_skill_legion_years'
  eligible_characters: EligibleCharacter[]
}

export interface MyApplication {
  id: number
  welfare_id: number
  welfare_name: string
  character_name: string
  status: 'requested' | 'delivered' | 'rejected'
  reviewer_name: string
  created_at: string
  reviewed_at: string | null
}

export type MyApplicationSearchParams = Partial<{
  current: number
  size: number
  status: string
}>

export interface ApplyParams {
  welfare_id: number
  character_id?: number
  evidence_image?: string
}

export interface ImportRecordsParams {
  welfare_id: number
  csv: string
}

export interface ImportRecordsResult {
  count: number
}

export interface AdminApplication {
  id: number
  welfare_id: number
  welfare_name: string
  welfare_description: string
  user_id: number | null
  applicant_nickname: string
  character_name: string
  qq: string
  discord_id: string
  evidence_image: string
  status: 'requested' | 'delivered' | 'rejected'
  reviewed_by: number
  reviewer_name: string
  created_at: string
  reviewed_at: string | null
}

export type AdminApplicationSearchParams = Partial<{
  current: number
  size: number
  status: string
  keyword: string
}>

export interface ReviewParams {
  id: number
  action: 'deliver' | 'reject'
}

export interface ReviewResult {
  code?: number
  msg?: string
  data?: unknown
}

export interface AdminDeleteApplicationParams {
  id: number
}

export type WelfareListResponse = PaginatedResponse<WelfareItem>
export type EligibleWelfareListResponse = EligibleWelfare[]
export type MyApplicationListResponse = PaginatedResponse<MyApplication>
export type AdminApplicationListResponse = PaginatedResponse<AdminApplication>

export interface WelfareSearchParams extends Partial<CommonSearchParams> {
  status?: number
  name?: string
}
