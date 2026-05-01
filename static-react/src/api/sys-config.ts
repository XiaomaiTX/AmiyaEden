import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AllowCorporationsConfig,
  BasicConfig,
  CharacterESIRestrictionConfig,
  SDEConfig,
  UpdateAllowCorporationsParams,
  UpdateCharacterESIRestrictionParams,
  UpdateSDEConfigParams,
} from '@/types/api/sys-config'

export async function fetchBasicConfig() {
  const response = await requestJson<ApiResponse<BasicConfig>>('/api/v1/system/basic-config')
  return assertSuccess(response, 'fetch basic config failed')
}

export async function fetchAllowCorporations() {
  const response = await requestJson<ApiResponse<AllowCorporationsConfig>>(
    '/api/v1/system/basic-config/allow-corporations'
  )
  return assertSuccess(response, 'fetch allow corporations failed')
}

export async function updateAllowCorporations(data: UpdateAllowCorporationsParams) {
  const response = await requestJson<ApiResponse<null>>(
    '/api/v1/system/basic-config/allow-corporations',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  assertSuccess(response, 'update allow corporations failed')
}

export async function fetchCharacterESIRestrictionConfig() {
  const response = await requestJson<ApiResponse<CharacterESIRestrictionConfig>>(
    '/api/v1/system/basic-config/character-esi-restriction'
  )
  return assertSuccess(response, 'fetch character esi restriction config failed')
}

export async function updateCharacterESIRestrictionConfig(data: UpdateCharacterESIRestrictionParams) {
  const response = await requestJson<ApiResponse<null>>(
    '/api/v1/system/basic-config/character-esi-restriction',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  assertSuccess(response, 'update character esi restriction config failed')
}

export async function fetchSDEConfig() {
  const response = await requestJson<ApiResponse<SDEConfig>>('/api/v1/system/sde-config')
  return assertSuccess(response, 'fetch sde config failed')
}

export async function updateSDEConfig(data: UpdateSDEConfigParams) {
  const response = await requestJson<ApiResponse<null>>('/api/v1/system/sde-config', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  assertSuccess(response, 'update sde config failed')
}
