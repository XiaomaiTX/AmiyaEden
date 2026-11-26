import type * as contract from '@/api/generated/client'

export type EveCharacter = Pick<
  contract.EveCharacter,
  | 'character_id'
  | 'character_name'
  | 'user_id'
  | 'scopes'
  | 'token_expiry'
  | 'token_invalid'
  | 'corporation_id'
  | 'alliance_id'
> & { fuxi_legion_tenure_days?: number | null }

export interface UserInfo {
  roles: string[]
  corpCapabilities: string[]
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
