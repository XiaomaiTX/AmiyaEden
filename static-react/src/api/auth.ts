import * as api from './generated/client'
import type { EveCharacter, UserInfo } from '@/types/api/auth'

import { unwrapContract } from './response'

function resolveProfileComplete(data: api.Me) {
  if (typeof data.profile_complete === 'boolean') {
    return data.profile_complete
  }

  const nickname = data.user.nickname?.trim() ?? ''
  const qq = data.user.qq?.trim() ?? ''
  const discordId = data.user.discord_id?.trim() ?? ''
  return nickname.length > 0 && (qq.length > 0 || discordId.length > 0)
}

export async function getEveSSOLoginURL(scopes?: string[]) {
  const callbackURL = `${window.location.origin}/#/auth/callback`
  const response = await unwrapContract(
    api.getEveSsoLogin({
      redirect: callbackURL,
      scopes: scopes?.length ? scopes.join(',') : undefined,
    }),
    'get eve sso login url failed'
  )
  return response.url
}

export async function fetchGetUserInfo(): Promise<UserInfo & { characters: EveCharacter[] }> {
  const data = await unwrapContract(api.getMe(), 'fetch current user failed')
  if (!data) throw new Error('fetch current user failed')

  const { user, characters, roles: backendRoles } = data
  const primaryChar =
    characters?.find((character) => character.character_id === user.primary_character_id) ??
    characters?.[0]

  const roles = backendRoles && backendRoles.length > 0 ? backendRoles : [user.role ?? 'user']

  return {
    roles,
    corpCapabilities: Array.isArray(data.corp_capabilities) ? data.corp_capabilities : [],
    userId: user.id,
    userName: primaryChar?.character_name ?? user.nickname ?? `Capsuleer#${user.id}`,
    nickname: user.nickname ?? '',
    qq: user.qq ?? '',
    discordId: user.discord_id ?? '',
    profileComplete: resolveProfileComplete(data),
    enforceCharacterESIRestriction: data.enforce_character_esi_restriction !== false,
    isCurrentlyNewbro:
      typeof data.is_currently_newbro === 'boolean' ? data.is_currently_newbro : undefined,
    isMentorMenteeEligible:
      typeof data.is_mentor_mentee_eligible === 'boolean'
        ? data.is_mentor_mentee_eligible
        : undefined,
    characters: characters ?? [],
    primaryCharacterId: primaryChar?.character_id ?? user.primary_character_id ?? 0,
  }
}

export async function getEveBindURL(scopes?: string[]) {
  const callbackURL = `${window.location.origin}/#/auth/callback`
  const response = await unwrapContract(
    api.getEveSsoBind({
      redirect: callbackURL,
      scopes: scopes?.length ? scopes.join(',') : undefined,
    }),
    'get eve bind url failed'
  )
  return response?.url ?? ''
}
