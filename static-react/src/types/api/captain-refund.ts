export interface RefundSettings {
  max_total_sp: number
  max_age_days: number
  tax_rate: number
  allowed_system_ids: number[]
}
export interface RefundRequest {
  captain_character_id: number
  member_names: string[]
  system_id: number
  start: string
  end: string
}
export interface RefundEligibility {
  is_currently_newbro: boolean
  disqualified_reason: string
  total_sp: number
  oldest_birthday: string | null
  max_total_sp: number
  max_age_days: number
  evaluated_at: string
  characters: {
    character_id: number
    total_sp: number
    has_skill_data: boolean
    birthday: string | null
  }[]
}
export interface RefundCalculation {
  request: RefundRequest
  settings: RefundSettings
  members: {
    character_id: number
    character_name: string
    user_id: number
    is_captain: boolean
    eligibility: RefundEligibility
    bounty: number
    wallet_updated_at: string | null
  }[]
  ignored_names: string[]
  journals: {
    character_id: number
    journal_id: number
    date: string
    amount: number
    match_journal_id: number
    match_date: string | null
  }[]
  conflicts: { character_id: number; journal_id: number; journal_at: string; refund_id: number }[]
  bounty: number
  reward: number
  has_newbro: boolean
  can_submit: boolean
}
export interface RefundApplication {
  id: number
  user_id: number
  status: 'pending' | 'approved' | 'rejected'
  reward: number
  created_at: string
  reviewed_at: string | null
  reviewed_by: number
  reject_reason: string
  wallet_ref_id: string
  calculation: RefundCalculation
}
export interface RefundOptions {
  characters: { character_id: number; character_name: string }[]
  settings: RefundSettings
}
export interface RefundList {
  list: RefundApplication[]
  total: number
  page: number
  page_size: number
}
