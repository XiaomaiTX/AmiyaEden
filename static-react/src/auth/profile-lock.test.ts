import { getProfileLockReasons } from '@/auth/profile-lock'

describe('profile lock reasons', () => {
  test('reports incomplete profile and invalid primary/secondary ESI independently', () => {
    expect(
      getProfileLockReasons({
        profileComplete: false,
        enforceCharacterESIRestriction: true,
        primaryCharacterId: 1,
        characters: [
          { characterId: 1, tokenInvalid: true },
          { characterId: 2, tokenInvalid: true },
        ],
      })
    ).toEqual([
      'profile_incomplete',
      'primary_character_token_invalid',
      'character_token_invalid',
    ])
  })

  test('does not lock on an invalid secondary character when enforcement is disabled', () => {
    expect(
      getProfileLockReasons({
        profileComplete: true,
        enforceCharacterESIRestriction: false,
        primaryCharacterId: 1,
        characters: [
          { characterId: 1, tokenInvalid: false },
          { characterId: 2, tokenInvalid: true },
        ],
      })
    ).toEqual([])
  })
})
