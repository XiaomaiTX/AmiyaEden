import { requestJson } from '@/api/http-client'
import type { EveCharacter, MeResponse, RegisteredScope, UserInfo } from '@/types/api/auth'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 200 && response.code !== 0) {
    throw new Error(response.msg || fallbackMessage)
  }
  return response.data
}

export async function getEveSSOLoginURL(scopes?: string[]) {
  const callbackURL = `${window.location.origin}/#/auth/callback`

  const params = new URLSearchParams({ redirect: callbackURL })
  if (scopes && scopes.length > 0) {
    params.set('scopes', scopes.join(','))
  }

  const response = await requestJson<ApiResponse<{ url: string }>>(
    `/api/v1/sso/eve/login?${params.toString()}`
  )
  return assertSuccess(response, 'get eve sso login url failed').url
}

export async function fetchMyCharacters() {
  const response = await requestJson<ApiResponse<EveCharacter[]>>('/api/v1/sso/eve/characters')
  return assertSuccess(response, 'fetch characters failed') ?? []
}

export async function fetchEveSSOScopes() {
  const response = await requestJson<ApiResponse<RegisteredScope[]>>('/api/v1/sso/eve/scopes')
  return assertSuccess(response, 'fetch scopes failed') ?? []
}

export async function fetchGetUserInfo() {
  const response = await requestJson<ApiResponse<MeResponse>>('/api/v1/me')
  const data = assertSuccess(response, 'fetch current user failed')

  const { user, characters, roles: backendRoles } = data
  const primaryChar =
    characters?.find((character) => character.character_id === user.primary_character_id) ??
    characters?.[0]

  const roles = backendRoles && backendRoles.length > 0 ? backendRoles : [user.role ?? 'user']

  return {
    roles,
    userId: user.id,
    userName: primaryChar?.character_name ?? user.nickname ?? `Capsuleer#${user.id}`,
    nickname: user.nickname ?? '',
    qq: user.qq ?? '',
    discordId: user.discord_id ?? '',
    profileComplete: data.profile_complete,
    enforceCharacterESIRestriction: data.enforce_character_esi_restriction !== false,
    isCurrentlyNewbro:
      typeof data.is_currently_newbro === 'boolean' ? data.is_currently_newbro : undefined,
    isMentorMenteeEligible:
      typeof data.is_mentor_mentee_eligible === 'boolean' ? data.is_mentor_mentee_eligible : undefined,
    characters: characters ?? [],
    primaryCharacterId: primaryChar?.character_id ?? user.primary_character_id ?? 0,
  } satisfies UserInfo
}

export async function getEveBindURL(scopes?: string[]) {
  const callbackURL = `${window.location.origin}/#/auth/callback`
  const params = new URLSearchParams({ redirect: callbackURL })
  if (scopes && scopes.length > 0) {
    params.set('scopes', scopes.join(','))
  }

  const response = await requestJson<ApiResponse<{ url: string }>>(
    `/api/v1/sso/eve/bind?${params.toString()}`
  )

  return assertSuccess(response, 'get eve bind url failed')?.url ?? ''
}

export async function setPrimaryCharacter(characterId: number) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/sso/eve/primary/${characterId}`, {
    method: 'PUT',
  })

  assertSuccess(response, 'set primary character failed')
}

export async function unbindCharacter(characterId: number) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/sso/eve/characters/${characterId}`, {
    method: 'DELETE',
  })

  assertSuccess(response, 'unbind character failed')
}

export async function updateMyProfile(data: {
  nickname?: string
  qq?: string
  discord_id?: string
}) {
  const response = await requestJson<ApiResponse<null>>('/api/v1/me', {
    method: 'PUT',
    body: JSON.stringify(data),
  })

  assertSuccess(response, 'update my profile failed')
}
