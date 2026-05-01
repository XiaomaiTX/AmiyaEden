import { requestJson } from '@/api/http-client'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || fallbackMessage)
  }

  return response.data
}

export async function fetchNpcKills(data: Api.NpcKill.NpcKillRequest) {
  const response = await requestJson<ApiResponse<Api.NpcKill.NpcKillResponse>>('/api/v1/info/npc-kills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch npc kills failed')
}

export async function fetchNpcKillsAll(data: Api.NpcKill.NpcKillAllRequest) {
  const response = await requestJson<ApiResponse<Api.NpcKill.NpcKillResponse>>('/api/v1/info/npc-kills/all', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch npc kills all failed')
}

export async function fetchCorpNpcKills(data: Api.NpcKill.NpcKillCorpRequest) {
  const response = await requestJson<ApiResponse<Api.NpcKill.NpcKillCorpResponse>>('/api/v1/system/npc-kills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch corp npc kills failed')
}
