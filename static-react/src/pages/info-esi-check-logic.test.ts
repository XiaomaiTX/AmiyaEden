import { calculateScopeCoverage, findInvalidCharacters } from '@/pages/info-esi-check-logic'

const scopes = [
  { module: 'wallet', scope: 'wallet.read', description: 'Wallet', required: true },
  { module: 'skills', scope: 'skills.read', description: 'Skills', required: false },
]

const characters = [
  {
    character_id: 1001,
    character_name: 'Amiya',
    user_id: 1,
    scopes: 'wallet.read skills.read',
    token_expiry: '2026-12-31T00:00:00Z',
    token_invalid: false,
    corporation_id: 1,
    alliance_id: 0,
  },
  {
    character_id: 1002,
    character_name: 'Beta',
    user_id: 1,
    scopes: 'skills.read',
    token_expiry: '2026-12-31T00:00:00Z',
    token_invalid: false,
    corporation_id: 1,
    alliance_id: 0,
  },
]

describe('ESI scope coverage logic', () => {
  test('marks a character missing a required scope as invalid', () => {
    expect(
      findInvalidCharacters(scopes, characters).map((character) => character.character_id)
    ).toEqual([1002])
  })

  test('calculates required coverage and row authorization for the selected character', () => {
    expect(calculateScopeCoverage(scopes, characters[1])).toMatchObject({
      grantedRequiredCount: 0,
      hasMissingRequiredScopes: true,
      scopeRows: [
        expect.objectContaining({ scope: 'wallet.read', authorized: false }),
        expect.objectContaining({ scope: 'skills.read', authorized: true }),
      ],
    })
  })
})
