import { requestJson } from '@/api/http-client'
import type {
  NpcKillAllRequest,
  NpcKillCorpRequest,
  NpcKillCorpResponse,
  NpcKillRequest,
  NpcKillResponse,
} from '@/types/api/npc-kill'

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

export async function fetchNpcKills(data: NpcKillRequest) {
  const response = await requestJson<ApiResponse<NpcKillResponse>>('/api/v1/info/npc-kills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch npc kills failed')
}

export async function fetchNpcKillsAll(data: NpcKillAllRequest) {
  const response = await requestJson<ApiResponse<NpcKillResponse>>('/api/v1/info/npc-kills/all', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch npc kills all failed')
}

export async function fetchCorpNpcKills(data: NpcKillCorpRequest) {
  const response = await requestJson<ApiResponse<NpcKillCorpResponse>>('/api/v1/system/npc-kills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch corp npc kills failed')
}
