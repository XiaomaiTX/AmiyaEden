import type { SessionSnapshot } from '@/stores'

export type ProfileLockReason =
  | 'profile_incomplete'
  | 'primary_character_token_invalid'
  | 'character_token_invalid'

type ProfileLockSnapshot = Pick<
  SessionSnapshot,
  'profileComplete' | 'enforceCharacterESIRestriction' | 'primaryCharacterId' | 'characters'
>

export function getProfileLockReasons(snapshot: ProfileLockSnapshot): ProfileLockReason[] {
  const reasons: ProfileLockReason[] = []

  if (!snapshot.profileComplete) {
    reasons.push('profile_incomplete')
  }

  const primary = snapshot.characters.find(
    (character) => character.characterId === snapshot.primaryCharacterId
  )
  if (primary?.tokenInvalid) {
    reasons.push('primary_character_token_invalid')
  }

  if (
    snapshot.enforceCharacterESIRestriction &&
    snapshot.characters.some(
      (character) =>
        character.characterId !== snapshot.primaryCharacterId && character.tokenInvalid
    )
  ) {
    reasons.push('character_token_invalid')
  }

  return reasons
}
