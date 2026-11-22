import {
  hasInvalidCharacterToken,
  hasInvalidPrimaryCharacterToken,
  isUserProfileComplete
} from '@/api/auth-helpers'

export const PROFILE_SETUP_PATH = '/characters'
export const DEFAULT_CHARACTERS_GATE_REFRESH_TTL_MS = 30_000

export type CharactersGateRefreshOptions = {
  forceRefresh?: boolean
  ttlMs?: number
  now?: number
  expectedUserId?: number
}

export type CharactersGateReason =
  | 'profile_incomplete'
  | 'primary_character_token_invalid'
  | 'character_token_invalid'

export type CharactersGateTransition = {
  reasons: CharactersGateReason[]
  redirectPath: string | null
  shouldWarn: boolean
}

type CharactersGateContext = {
  isLogin: boolean
  path: string
}

type CharactersGateUserInfo = Partial<
  Pick<
    Api.Auth.UserInfo,
    | 'profileComplete'
    | 'nickname'
    | 'qq'
    | 'discordId'
    | 'characters'
    | 'primaryCharacterId'
    | 'enforceCharacterESIRestriction'
  >
>

let inFlightCharactersGateRefresh: Promise<Api.Auth.UserInfo> | null = null
let inFlightCharactersGateUserId: number | undefined
// Most recently started refresh; cleared on invalidation. Waiters whose task is no longer the
// latest were superseded by a newer identity or a logout and must not write into the store.
let latestCharactersGateRefresh: Promise<Api.Auth.UserInfo> | null = null
let cachedCharactersGateUserInfo: Api.Auth.UserInfo | null = null
let cachedCharactersGateFetchedAt = 0

function hasInvalidSecondaryCharacterToken(
  primaryCharacterId?: number,
  characters?: Array<Pick<Api.Auth.EveCharacter, 'character_id' | 'token_invalid'>>
): boolean {
  if (!primaryCharacterId) {
    return hasInvalidCharacterToken(characters)
  }

  return (
    characters?.some(
      (character) =>
        character.character_id !== primaryCharacterId && character.token_invalid === true
    ) ?? false
  )
}

export function getCharactersGateReasons(
  context: CharactersGateContext,
  userInfo?: CharactersGateUserInfo
): CharactersGateReason[] {
  if (!context.isLogin || context.path === PROFILE_SETUP_PATH) {
    return []
  }

  const reasons: CharactersGateReason[] = []

  if (!isUserProfileComplete(userInfo)) {
    reasons.push('profile_incomplete')
  }

  if (hasInvalidPrimaryCharacterToken(userInfo?.primaryCharacterId, userInfo?.characters)) {
    reasons.push('primary_character_token_invalid')
  }

  if (
    userInfo?.enforceCharacterESIRestriction !== false &&
    hasInvalidSecondaryCharacterToken(userInfo?.primaryCharacterId, userInfo?.characters)
  ) {
    reasons.push('character_token_invalid')
  }

  return reasons
}

export function resolveCharactersGateTransition(
  context: CharactersGateContext,
  userInfo?: CharactersGateUserInfo,
  destinationPath: string = PROFILE_SETUP_PATH
): CharactersGateTransition {
  const reasons = getCharactersGateReasons(context, userInfo)

  return {
    reasons,
    redirectPath: reasons.length > 0 ? PROFILE_SETUP_PATH : null,
    shouldWarn: reasons.length > 0 && destinationPath === PROFILE_SETUP_PATH
  }
}

export function applyCharactersGateTransition(
  transition: CharactersGateTransition,
  redirect: () => void,
  notify: (reasons: CharactersGateReason[]) => void
): boolean {
  if (!transition.redirectPath) {
    return false
  }

  redirect()

  if (transition.shouldWarn) {
    notify(transition.reasons)
  }

  return true
}

export function shouldRedirectToCharactersPage(
  context: CharactersGateContext,
  userInfo?: CharactersGateUserInfo
): boolean {
  return resolveCharactersGateTransition(context, userInfo).redirectPath !== null
}

function hasBlockingGateReasons(userInfo?: CharactersGateUserInfo): boolean {
  return getCharactersGateReasons({ isLogin: true, path: '/' }, userInfo).length > 0
}

export function markCharactersGateStateFresh(
  userInfo: Api.Auth.UserInfo,
  now: number = Date.now()
): void {
  cachedCharactersGateUserInfo = userInfo
  cachedCharactersGateFetchedAt = now
}

export function invalidateCharactersGateStateFreshness(): void {
  inFlightCharactersGateRefresh = null
  inFlightCharactersGateUserId = undefined
  latestCharactersGateRefresh = null
  cachedCharactersGateUserInfo = null
  cachedCharactersGateFetchedAt = 0
}

export async function refreshCharactersGateState(
  fetchUserInfo: () => Promise<Api.Auth.UserInfo>,
  storeUserInfo: (userInfo: Api.Auth.UserInfo) => void,
  options: CharactersGateRefreshOptions = {}
): Promise<Api.Auth.UserInfo> {
  const ttlMs = options.ttlMs ?? DEFAULT_CHARACTERS_GATE_REFRESH_TTL_MS
  const now = options.now ?? Date.now()
  const hasExpectedUserId = options.expectedUserId !== undefined

  if (
    cachedCharactersGateUserInfo &&
    hasExpectedUserId &&
    cachedCharactersGateUserInfo.userId !== options.expectedUserId
  ) {
    cachedCharactersGateUserInfo = null
    cachedCharactersGateFetchedAt = 0
  }

  if (
    !options.forceRefresh &&
    ttlMs > 0 &&
    cachedCharactersGateUserInfo &&
    (!hasExpectedUserId || cachedCharactersGateUserInfo.userId === options.expectedUserId) &&
    !hasBlockingGateReasons(cachedCharactersGateUserInfo) &&
    now - cachedCharactersGateFetchedAt < ttlMs
  ) {
    return cachedCharactersGateUserInfo
  }

  if (
    !inFlightCharactersGateRefresh ||
    (hasExpectedUserId && inFlightCharactersGateUserId !== options.expectedUserId)
  ) {
    const requestUserId = options.expectedUserId
    const task = fetchUserInfo()
      .then((userInfo) => {
        if (inFlightCharactersGateRefresh === task) {
          markCharactersGateStateFresh(userInfo, now)
        }
        return userInfo
      })
      .finally(() => {
        if (inFlightCharactersGateRefresh === task) {
          inFlightCharactersGateRefresh = null
          inFlightCharactersGateUserId = undefined
        }
      })
    inFlightCharactersGateUserId = requestUserId
    inFlightCharactersGateRefresh = task
    latestCharactersGateRefresh = task
  }

  const task = inFlightCharactersGateRefresh
  const userInfo = await task
  if (task !== latestCharactersGateRefresh) {
    // Superseded: resolve against the current session instead of restoring a stale identity.
    return refreshCharactersGateState(fetchUserInfo, storeUserInfo, {
      forceRefresh: options.forceRefresh,
      ttlMs: options.ttlMs,
      now: options.now
    })
  }
  storeUserInfo(userInfo)
  return userInfo
}

export function resetCharactersGateStateRefreshForTest(): void {
  inFlightCharactersGateRefresh = null
  invalidateCharactersGateStateFreshness()
}
