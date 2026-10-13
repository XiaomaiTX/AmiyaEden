import { requestJson } from '@/api/http-client'
import type { ApiResponse } from '@/api/response'
import type {
  SkillReadinessCharacter,
  SkillReadinessCharacterDetail,
  SkillReadinessOptions,
  SkillReadinessRequest,
  SkillReadinessResult,
} from '@/types/api/skill-readiness'

function unwrap<T>(response: ApiResponse<T>) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || 'Skill readiness request failed')
  }
  return response.data
}

export async function fetchSkillReadinessOptions() {
  const response = await requestJson<ApiResponse<SkillReadinessOptions>>(
    '/api/v1/dashboard/skill-readiness/options',
    { method: 'GET' }
  )
  return unwrap(response)
}

export async function querySkillReadiness(data: SkillReadinessRequest) {
  const response = await requestJson<ApiResponse<SkillReadinessResult>>(
    '/api/v1/dashboard/skill-readiness/query',
    { method: 'POST', body: JSON.stringify(data) }
  )
  return unwrap(response)
}

export async function fetchSkillReadinessUserCharacters(
  userId: number,
  data: SkillReadinessRequest
) {
  const response = await requestJson<ApiResponse<SkillReadinessCharacter[]>>(
    `/api/v1/dashboard/skill-readiness/users/${userId}/characters`,
    { method: 'POST', body: JSON.stringify(data) }
  )
  return unwrap(response)
}

export async function fetchSkillReadinessCharacterSkills(
  characterId: number,
  data: SkillReadinessRequest
) {
  const response = await requestJson<ApiResponse<SkillReadinessCharacterDetail>>(
    `/api/v1/dashboard/skill-readiness/characters/${characterId}/skills`,
    { method: 'POST', body: JSON.stringify(data) }
  )
  return unwrap(response)
}
