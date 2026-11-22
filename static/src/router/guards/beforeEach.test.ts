import assert from 'node:assert/strict'
import test, { afterEach } from 'node:test'
import * as charactersGate from './charactersGate'

const {
  applyCharactersGateTransition,
  getCharactersGateReasons,
  invalidateCharactersGateStateFreshness,
  markCharactersGateStateFresh,
  refreshCharactersGateState,
  resetCharactersGateStateRefreshForTest,
  resolveCharactersGateTransition,
  shouldRedirectToCharactersPage
} = charactersGate

afterEach(() => {
  resetCharactersGateStateRefreshForTest()
})

test('invalid non-primary character still requires redirect to characters page', () => {
  assert.equal(
    shouldRedirectToCharactersPage(
      { isLogin: true, path: '/system/user' },
      {
        enforceCharacterESIRestriction: true,
        profileComplete: true,
        characters: [
          { token_invalid: false } as Api.Auth.EveCharacter,
          { token_invalid: true } as Api.Auth.EveCharacter
        ]
      }
    ),
    true
  )
})

test('characters page is allowed even when there are invalid tokens', () => {
  assert.equal(
    shouldRedirectToCharactersPage(
      { isLogin: true, path: '/characters' },
      {
        enforceCharacterESIRestriction: true,
        profileComplete: true,
        characters: [{ token_invalid: true } as Api.Auth.EveCharacter]
      }
    ),
    false
  )
})

test('invalid non-primary character does not redirect when enforcement is disabled', () => {
  assert.equal(
    shouldRedirectToCharactersPage(
      { isLogin: true, path: '/system/user' },
      {
        enforceCharacterESIRestriction: false,
        profileComplete: true,
        characters: [{ token_invalid: true } as Api.Auth.EveCharacter]
      }
    ),
    false
  )
})

test('invalid primary character still redirects when enforcement is disabled', () => {
  assert.equal(
    shouldRedirectToCharactersPage(
      { isLogin: true, path: '/system/user' },
      {
        enforceCharacterESIRestriction: false,
        profileComplete: true,
        primaryCharacterId: 9001,
        characters: [
          { character_id: 9001, token_invalid: true } as Api.Auth.EveCharacter,
          { character_id: 9002, token_invalid: false } as Api.Auth.EveCharacter
        ]
      }
    ),
    true
  )
})

test('characters gate reports every active lock reason in priority order', () => {
  assert.equal(typeof getCharactersGateReasons, 'function')
  assert.deepEqual(
    getCharactersGateReasons(
      { isLogin: true, path: '/system/user' },
      {
        profileComplete: false,
        enforceCharacterESIRestriction: true,
        primaryCharacterId: 9001,
        characters: [
          { character_id: 9001, token_invalid: true } as Api.Auth.EveCharacter,
          { character_id: 9002, token_invalid: true } as Api.Auth.EveCharacter
        ]
      }
    ),
    ['profile_incomplete', 'primary_character_token_invalid', 'character_token_invalid']
  )
})

test('primary-character lock reason is not duplicated as all-character enforcement', () => {
  assert.equal(typeof getCharactersGateReasons, 'function')
  assert.deepEqual(
    getCharactersGateReasons(
      { isLogin: true, path: '/system/user' },
      {
        profileComplete: true,
        enforceCharacterESIRestriction: true,
        primaryCharacterId: 9001,
        characters: [{ character_id: 9001, token_invalid: true } as Api.Auth.EveCharacter]
      }
    ),
    ['primary_character_token_invalid']
  )
})

test('characters gate transition warns when locked navigation lands on characters page', () => {
  const transition = resolveCharactersGateTransition(
    { isLogin: true, path: '/system/user' },
    {
      profileComplete: false,
      enforceCharacterESIRestriction: true,
      primaryCharacterId: 9001,
      characters: [{ character_id: 9001, token_invalid: false } as Api.Auth.EveCharacter]
    }
  )

  let redirectCalls = 0
  const warningCalls: string[][] = []

  assert.equal(
    applyCharactersGateTransition(
      transition,
      () => {
        redirectCalls += 1
      },
      (reasons) => {
        warningCalls.push([...reasons])
      }
    ),
    true
  )
  assert.equal(redirectCalls, 1)
  assert.deepEqual(warningCalls, [['profile_incomplete']])
})

test('characters gate transition suppresses warning when permission fallback avoids characters page', () => {
  const transition = resolveCharactersGateTransition(
    { isLogin: true, path: '/system/user' },
    {
      profileComplete: true,
      enforceCharacterESIRestriction: true,
      primaryCharacterId: 9001,
      characters: [{ character_id: 9001, token_invalid: true } as Api.Auth.EveCharacter]
    },
    '/dashboard'
  )

  let redirectCalls = 0
  let warningCalls = 0

  assert.equal(
    applyCharactersGateTransition(
      transition,
      () => {
        redirectCalls += 1
      },
      () => {
        warningCalls += 1
      }
    ),
    true
  )
  assert.equal(redirectCalls, 1)
  assert.equal(warningCalls, 0)
})

test('characters gate transition does nothing when the user is not locked out', () => {
  let redirectCalls = 0
  let warningCalls = 0

  assert.equal(
    applyCharactersGateTransition(
      resolveCharactersGateTransition(
        { isLogin: true, path: '/system/user' },
        {
          profileComplete: true,
          enforceCharacterESIRestriction: true,
          primaryCharacterId: 9001,
          characters: [{ character_id: 9001, token_invalid: false } as Api.Auth.EveCharacter]
        }
      ),
      () => {
        redirectCalls += 1
      },
      () => {
        warningCalls += 1
      }
    ),
    false
  )
  assert.equal(redirectCalls, 0)
  assert.equal(warningCalls, 0)
})

test('guard refresh stores authoritative user info before redirect evaluation', async () => {
  const refreshedUser = {
    userId: 7,
    roles: ['user'],
    userName: 'Amiya',
    nickname: 'Amiya',
    qq: '12345',
    discordId: '',
    profileComplete: true,
    enforceCharacterESIRestriction: true,
    characters: [{ token_invalid: true } as Api.Auth.EveCharacter]
  } as Api.Auth.UserInfo

  let storedUser: Api.Auth.UserInfo | undefined

  const result = await refreshCharactersGateState(
    async () => refreshedUser,
    (value) => {
      storedUser = value
    }
  )

  assert.equal(storedUser?.characters?.[0]?.token_invalid, true)
  assert.equal(result.characters?.[0]?.token_invalid, true)
})

test('concurrent gate refreshes share one current-user request', async () => {
  let calls = 0
  const refreshedUser = {
    userId: 7,
    roles: ['user'],
    userName: 'Amiya',
    nickname: 'Amiya',
    qq: '12345',
    discordId: '',
    profileComplete: true,
    enforceCharacterESIRestriction: true,
    characters: []
  } as Api.Auth.UserInfo

  const fetchUserInfo = async () => {
    calls += 1
    return refreshedUser
  }

  await Promise.all([
    refreshCharactersGateState(fetchUserInfo, () => {}),
    refreshCharactersGateState(fetchUserInfo, () => {})
  ])

  assert.equal(calls, 1)
})

test('healthy gate state is reused within TTL and refreshed after expiry or forceRefresh', async () => {
  let calls = 0
  let storeCalls = 0
  const healthyUser = {
    userId: 7,
    roles: ['user'],
    userName: 'Amiya',
    nickname: 'Amiya',
    qq: '12345',
    discordId: '',
    profileComplete: true,
    enforceCharacterESIRestriction: true,
    primaryCharacterId: 9001,
    characters: [{ character_id: 9001, token_invalid: false } as Api.Auth.EveCharacter]
  } as Api.Auth.UserInfo

  const fetchUserInfo = async () => {
    calls += 1
    return healthyUser
  }
  const storeUserInfo = () => {
    storeCalls += 1
  }

  await refreshCharactersGateState(fetchUserInfo, storeUserInfo, { now: 1_000, ttlMs: 30_000 })
  await refreshCharactersGateState(fetchUserInfo, storeUserInfo, { now: 15_000, ttlMs: 30_000 })
  assert.equal(calls, 1)
  assert.equal(storeCalls, 1)

  // Force refresh bypasses TTL (e.g. leaving /characters)
  await refreshCharactersGateState(fetchUserInfo, storeUserInfo, {
    now: 16_000,
    ttlMs: 30_000,
    forceRefresh: true
  })
  assert.equal(calls, 2)
  assert.equal(storeCalls, 2)

  // Expired TTL fetches again
  await refreshCharactersGateState(fetchUserInfo, storeUserInfo, { now: 50_000, ttlMs: 30_000 })
  assert.equal(calls, 3)
  assert.equal(storeCalls, 3)
})

test('locked-out gate state is never served from TTL cache', async () => {
  let calls = 0
  const lockedUser = {
    userId: 7,
    roles: ['user'],
    userName: 'Amiya',
    nickname: '',
    qq: '',
    discordId: '',
    profileComplete: false,
    enforceCharacterESIRestriction: true,
    characters: []
  } as Api.Auth.UserInfo

  const fetchUserInfo = async () => {
    calls += 1
    return lockedUser
  }

  await refreshCharactersGateState(fetchUserInfo, () => {}, { now: 1_000, ttlMs: 30_000 })
  await refreshCharactersGateState(fetchUserInfo, () => {}, { now: 2_000, ttlMs: 30_000 })
  assert.equal(calls, 2)
})

test('markCharactersGateStateFresh skips immediate duplicate fetch and invalidation resets it', async () => {
  let calls = 0
  const healthyUser = {
    userId: 7,
    roles: ['user'],
    userName: 'Amiya',
    nickname: 'Amiya',
    qq: '12345',
    discordId: '',
    profileComplete: true,
    enforceCharacterESIRestriction: true,
    primaryCharacterId: 9001,
    characters: [{ character_id: 9001, token_invalid: false } as Api.Auth.EveCharacter]
  } as Api.Auth.UserInfo

  markCharactersGateStateFresh(healthyUser, 10_000)

  const cached = await refreshCharactersGateState(
    async () => {
      calls += 1
      return healthyUser
    },
    () => {},
    { now: 10_500, ttlMs: 30_000 }
  )
  assert.equal(calls, 0)
  assert.equal(cached.userId, 7)

  invalidateCharactersGateStateFreshness()
  await refreshCharactersGateState(
    async () => {
      calls += 1
      return healthyUser
    },
    () => {},
    { now: 11_000, ttlMs: 30_000 }
  )
  assert.equal(calls, 1)
})

test('gate state TTL cache is scoped by expectedUserId and ignores invalidated in-flight requests', async () => {
  const userA = {
    userId: 7,
    roles: ['user'],
    userName: 'UserA',
    nickname: 'UserA',
    qq: '12345',
    discordId: '',
    profileComplete: true,
    enforceCharacterESIRestriction: true,
    primaryCharacterId: 9001,
    characters: [{ character_id: 9001, token_invalid: false } as Api.Auth.EveCharacter]
  } as Api.Auth.UserInfo

  const userB = {
    ...userA,
    userId: 8,
    userName: 'UserB',
    nickname: 'UserB',
    profileComplete: false
  } as Api.Auth.UserInfo

  markCharactersGateStateFresh(userA, 10_000)

  let calls = 0
  const fetchedB = await refreshCharactersGateState(
    async () => {
      calls += 1
      return userB
    },
    () => {},
    { now: 10_500, ttlMs: 30_000, expectedUserId: 8 }
  )
  assert.equal(calls, 1)
  assert.equal(fetchedB.userId, 8)

  // Invalidate while an in-flight request for user A is pending; its resolution must not repopulate cache.
  // The superseded waiter must not write user A back into the store; it resolves against the new session.
  resetCharactersGateStateRefreshForTest()
  const stored: number[] = []
  let resolveA!: (value: Api.Auth.UserInfo) => void
  let staleFetches = 0
  const pendingA = refreshCharactersGateState(
    () => {
      staleFetches += 1
      if (staleFetches > 1) {
        return Promise.resolve(userB)
      }
      return new Promise<Api.Auth.UserInfo>((resolve) => {
        resolveA = resolve
      })
    },
    (userInfo) => {
      stored.push(userInfo.userId)
    },
    { now: 20_000, ttlMs: 30_000, expectedUserId: 7 }
  )
  invalidateCharactersGateStateFreshness()
  resolveA(userA)
  const resolvedAfterInvalidation = await pendingA
  assert.equal(resolvedAfterInvalidation.userId, 8)
  assert.deepEqual(stored, [8])
  resetCharactersGateStateRefreshForTest()

  let bCalls = 0
  await refreshCharactersGateState(
    async () => {
      bCalls += 1
      return { ...userB, profileComplete: true }
    },
    () => {},
    { now: 20_500, ttlMs: 30_000, expectedUserId: 8 }
  )
  assert.equal(bCalls, 1)
})
