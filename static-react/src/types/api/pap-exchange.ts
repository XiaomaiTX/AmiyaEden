export interface RateItem {
  pap_type: string
  display_name: string
  rate: number
  updated_at: string
}

export interface ConfigResponse {
  rates: RateItem[]
  fc_salary: number
  fc_salary_monthly_limit: number
  admin_award: number
  multichar_full_reward_count: number
  multichar_reduced_reward_count: number
  multichar_reduced_reward_pct: number
}

export interface UpdateRateItem {
  pap_type: string
  display_name: string
  rate: number
}

export interface UpdateConfigParams {
  rates: UpdateRateItem[]
  fc_salary: number
  fc_salary_monthly_limit: number
  admin_award: number
  multichar_full_reward_count: number
  multichar_reduced_reward_count: number
  multichar_reduced_reward_pct: number
}
