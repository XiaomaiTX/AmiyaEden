import request from '@/utils/http'

const baseURL = '/api/v1/dashboard/skill-readiness'

export function fetchSkillReadinessOptions() {
  return request.get<Api.SkillReadiness.Options>({ url: `${baseURL}/options` })
}

export function querySkillReadiness(data: Api.SkillReadiness.Request) {
  return request.post<Api.SkillReadiness.Result>({ url: `${baseURL}/query`, data })
}

export function fetchSkillReadinessUserCharacters(
  userId: number,
  data: Api.SkillReadiness.Request
) {
  return request.post<Api.SkillReadiness.CharacterRow[]>({
    url: `${baseURL}/users/${userId}/characters`,
    data
  })
}

export function fetchSkillReadinessCharacterSkills(
  characterId: number,
  data: Api.SkillReadiness.Request
) {
  return request.post<Api.SkillReadiness.CharacterDetail>({
    url: `${baseURL}/characters/${characterId}/skills`,
    data
  })
}
