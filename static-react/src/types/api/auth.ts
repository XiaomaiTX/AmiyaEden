export interface EveCharacter {
  character_id: number
  character_name: string
  user_id: number
  scopes: string
  token_expiry: string
  token_invalid: boolean
  corporation_id: number
  alliance_id: number
  fuxi_legion_tenure_days?: number | null
}

export interface RegisteredScope {
  module: string
  scope: string
  description: string
  required: boolean
}

export interface MeResponse {
  user: {
    id: number
    nickname: string
    qq: string
    discord_id: string
    status: number
    role: string
    primary_character_id: number
    last_login_at: string | null
    last_login_ip: string
  }
  characters: EveCharacter[]
  roles: string[]
  permissions: string[]
  profile_complete: boolean
  enforce_character_esi_restriction: boolean
  is_currently_newbro?: boolean | null
  is_mentor_mentee_eligible?: boolean | null
}

export interface UserInfo {
  roles: string[]
  userId: number
  userName: string
  nickname: string
  qq: string
  discordId: string
  profileComplete: boolean
  enforceCharacterESIRestriction: boolean
  isCurrentlyNewbro?: boolean
  isMentorMenteeEligible?: boolean
  characters?: EveCharacter[]
  primaryCharacterId?: number
}

