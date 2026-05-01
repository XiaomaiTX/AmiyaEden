import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  Application,
  ApplicationList,
  ApplicationSearchParams,
  AutoApproveParams,
  AutoApproveSummary,
  BatchFuxiPayoutActionResult,
  BatchPayoutActionResult,
  BatchPayoutSummary,
  KillmailDetailRequest,
  KillmailDetailResponse,
  KillmailListParams,
  FleetKillmailItem,
  PayoutParams,
  ReviewParams,
  ShipPrice,
  SrpConfig,
  SubmitApplicationParams,
  UpsertShipPriceParams,
} from '@/types/api/srp'

type ApiResult<T> = ApiResponse<T>

export async function fetchSrpConfig() {
  const response = await requestJson<ApiResult<SrpConfig>>('/api/v1/srp/config')
  return assertSuccess(response, 'fetch srp config failed')
}

export async function updateSrpConfig(data: SrpConfig) {
  const response = await requestJson<ApiResult<SrpConfig>>('/api/v1/srp/config', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update srp config failed')
}

export async function fetchShipPrices(keyword?: string) {
  const searchParams = new URLSearchParams()
  if (keyword) searchParams.set('keyword', keyword)
  const response = await requestJson<ApiResult<ShipPrice[]>>(
    `/api/v1/srp/prices${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch ship prices failed') ?? []
}

export async function upsertShipPrice(data: UpsertShipPriceParams) {
  const response = await requestJson<ApiResult<ShipPrice>>('/api/v1/srp/prices', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'upsert ship price failed')
}

export async function deleteShipPrice(id: number) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/srp/prices/${id}`, {
    method: 'DELETE',
  })
  return assertSuccess(response, 'delete ship price failed')
}

export async function submitApplication(data: SubmitApplicationParams) {
  const response = await requestJson<ApiResult<Application>>('/api/v1/srp/applications', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'submit srp application failed')
}

export async function fetchMyApplications(params?: Partial<{ current: number; size: number }>) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))

  const response = await requestJson<ApiResult<ApplicationList>>(
    `/api/v1/srp/applications/me${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch my srp applications failed')
}

export async function fetchFleetKillmails(fleetId: string, params?: KillmailListParams) {
  const searchParams = new URLSearchParams()
  if (params?.character_id) searchParams.set('character_id', String(params.character_id))
  if (params?.limit) searchParams.set('limit', String(params.limit))
  if (params?.exclude_submitted) searchParams.set('exclude_submitted', String(params.exclude_submitted))

  const response = await requestJson<ApiResult<FleetKillmailItem[]>>(
    `/api/v1/srp/killmails/fleet/${fleetId}${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch fleet killmails failed') ?? []
}

export async function fetchMyKillmails(params?: KillmailListParams) {
  const searchParams = new URLSearchParams()
  if (params?.character_id) searchParams.set('character_id', String(params.character_id))
  if (params?.limit) searchParams.set('limit', String(params.limit))
  if (params?.exclude_submitted) searchParams.set('exclude_submitted', String(params.exclude_submitted))

  const response = await requestJson<ApiResult<FleetKillmailItem[]>>(
    `/api/v1/srp/killmails/me${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch my killmails failed') ?? []
}

export async function fetchKillmailDetail(data: KillmailDetailRequest) {
  const response = await requestJson<ApiResult<KillmailDetailResponse>>('/api/v1/srp/killmails/detail', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'fetch killmail detail failed')
}

export async function fetchApplicationList(params?: ApplicationSearchParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.fleet_id) searchParams.set('fleet_id', params.fleet_id)
  if (params?.character_id) searchParams.set('character_id', String(params.character_id))
  if (params?.review_status) searchParams.set('review_status', params.review_status)
  if (params?.payout_status) searchParams.set('payout_status', params.payout_status)
  if (params?.tab) searchParams.set('tab', params.tab)
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<ApplicationList>>(
    `/api/v1/srp/applications${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch srp application list failed')
}

export async function fetchApplicationDetail(id: number) {
  const response = await requestJson<ApiResult<Application>>(`/api/v1/srp/applications/${id}`)
  return assertSuccess(response, 'fetch srp application detail failed')
}

export async function reviewApplication(id: number, data: ReviewParams) {
  const response = await requestJson<ApiResult<Application>>(`/api/v1/srp/applications/${id}/review`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'review srp application failed')
}

export async function payoutApplication(id: number, data?: PayoutParams) {
  const response = await requestJson<ApiResult<BatchPayoutActionResult>>(
    `/api/v1/srp/applications/${id}/payout`,
    {
      method: 'PUT',
      body: JSON.stringify(data ?? {}),
    }
  )
  return assertSuccess(response, 'payout srp application failed')
}

export async function batchPayoutAsFuxiCoin() {
  const response = await requestJson<ApiResult<BatchFuxiPayoutActionResult>>(
    '/api/v1/srp/applications/fuxi-payout',
    {
      method: 'PUT',
      body: JSON.stringify({}),
    }
  )
  return assertSuccess(response, 'batch payout fuxi failed')
}

export async function fetchBatchPayoutSummary() {
  const response = await requestJson<ApiResult<BatchPayoutSummary[]>>(
    '/api/v1/srp/applications/batch-payout-summary'
  )
  return assertSuccess(response, 'fetch batch payout summary failed') ?? []
}

export async function runFleetAutoApproval(data: AutoApproveParams) {
  const response = await requestJson<ApiResult<AutoApproveSummary>>(
    '/api/v1/srp/applications/auto-approve',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'run fleet auto approval failed')
}

export async function batchPayoutByUser(userId: number) {
  const response = await requestJson<ApiResult<BatchPayoutActionResult>>(
    `/api/v1/srp/applications/users/${userId}/payout`,
    {
      method: 'PUT',
      body: JSON.stringify({}),
    }
  )
  return assertSuccess(response, 'batch payout by user failed')
}

export async function openInfoWindow(data: { character_id: number; target_id: number }) {
  const response = await requestJson<ApiResult<null>>('/api/v1/srp/open-info-window', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'open info window failed')
}
