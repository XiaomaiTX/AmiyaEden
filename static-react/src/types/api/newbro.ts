export interface DirectReferralStatus {
  show_card: boolean
  needs_profile_qq: boolean
}

export interface DirectReferrerCandidate {
  user_id: number
  nickname: string
  primary_character_id: number
  primary_character_name: string
}

export interface CheckDirectReferrerParams {
  qq: string
}

export interface ConfirmDirectReferrerParams {
  referrer_user_id: number
}

