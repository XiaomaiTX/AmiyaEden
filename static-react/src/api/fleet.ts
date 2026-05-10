import { requestJson } from '@/api/http-client'
import type { ApiResponse } from '@/api/response'
import type { FleetPapLog, JoinFleetParams } from '@/types/api/fleet'

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || fallbackMessage)
  }

  return response.data
}

type ApiResult<T> = ApiResponse<T>

export async function joinFleet(data: JoinFleetParams) {
  const response = await requestJson<ApiResult<null>>('/api/v1/operation/fleets/join', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'join fleet failed')
}

export async function fetchMyPapLogs() {
  const response = await requestJson<ApiResult<FleetPapLog[]>>('/api/v1/operation/fleets/pap/me', {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch my pap logs failed')
}
