import type { UserInfo } from '@/types/api/auth'
import type { SessionSnapshot } from '@/stores'

export function toSessionSnapshot(userInfo: UserInfo): Partial<SessionSnapshot> {
  return {
    isLoggedIn: true,
    characterId: userInfo.primaryCharacterId ?? null,
    characterName: userInfo.userName,
    roles: userInfo.roles,
    corpCapabilities: userInfo.corpCapabilities,
    isCurrentlyNewbro: userInfo.isCurrentlyNewbro === true,
    isMentorMenteeEligible: userInfo.isMentorMenteeEligible === true,
    profileComplete: userInfo.profileComplete,
    enforceCharacterESIRestriction: userInfo.enforceCharacterESIRestriction,
    primaryCharacterId: userInfo.primaryCharacterId ?? null,
    characters: (userInfo.characters ?? []).map((character) => ({
      characterId: character.character_id,
      tokenInvalid: character.token_invalid,
    })),
  }
}
