import { requestJson } from '@/api/http-client'
import type { ApiResponse } from '@/api/response'
import type {
  AlliancePAPFetchParams,
  AlliancePAPImportParams,
  AlliancePAPResult,
  AlliancePAPSearchParams,
  AlliancePAPSettleParams,
  AlliancePAPSummary,
} from '@/types/api/alliance-pap'
import type { PaginatedResponse } from '@/types/api/common'

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || fallbackMessage)
  }

  return response.data
}

type ApiResult<T> = ApiResponse<T>

export async function fetchMyAlliancePAP(params?: { year?: number; month?: number }) {
  const url = params
    ? `/api/v1/operation/fleets/pap/alliance?${new URLSearchParams(
        Object.entries(params).reduce<Record<string, string>>((acc, [key, value]) => {
          if (value !== undefined && value !== null) acc[key] = String(value)
          return acc
        }, {})
      ).toString()}`
    : '/api/v1/operation/fleets/pap/alliance'
  const response = await requestJson<ApiResult<AlliancePAPResult>>(url, { method: 'GET' })
  return assertSuccess(response, 'fetch alliance pap failed')
}

export async function fetchAllAlliancePAP(params?: AlliancePAPSearchParams) {
  const url = params
    ? `/api/v1/system/pap?${new URLSearchParams(
        Object.entries(params).reduce<Record<string, string>>((acc, [key, value]) => {
          if (value !== undefined && value !== null) acc[key] = String(value)
          return acc
        }, {})
      ).toString()}`
    : '/api/v1/system/pap'
  const response = await requestJson<ApiResult<PaginatedResponse<AlliancePAPSummary>>>(url, {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch all alliance pap failed')
}

export async function triggerAlliancePAPFetch(params?: AlliancePAPFetchParams) {
  const response = await requestJson<ApiResult<{ message: string }>>('/api/v1/system/pap/fetch', {
    method: 'POST',
    body: params ? JSON.stringify(params) : undefined,
  })
  return assertSuccess(response, 'trigger alliance pap fetch failed')
}

export async function importAlliancePAP(params: AlliancePAPImportParams) {
  const response = await requestJson<ApiResult<{ message: string }>>('/api/v1/system/pap/import', {
    method: 'POST',
    body: JSON.stringify(params),
  })
  return assertSuccess(response, 'import alliance pap failed')
}

export async function settleAlliancePAPMonth(params: AlliancePAPSettleParams) {
  const response = await requestJson<ApiResult<{ year: number; month: number }>>(
    '/api/v1/system/pap/settle',
    {
      method: 'POST',
      body: JSON.stringify(params),
    }
  )
  return assertSuccess(response, 'settle alliance pap failed')
}
