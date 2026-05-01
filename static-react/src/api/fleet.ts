import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  CharacterFleetInfo,
  CorporationPapSummaryList,
  CorporationPapSummaryParams,
  CreateFleetParams,
  ESIFleetMember,
  FleetInvite,
  FleetItem,
  FleetList,
  FleetMember,
  FleetPapLog,
  FleetSearchParams,
  JoinFleetParams,
  ManualAddFleetMembersParams,
  ManualAddFleetMembersResult,
  MemberWithPap,
  UpdateFleetParams,
} from '@/types/api/fleet'

type ApiResult<T> = ApiResponse<T>

export async function fetchFleetList(params?: FleetSearchParams) {
  const search = new URLSearchParams()
  if (params?.current != null) search.set('current', String(params.current))
  if (params?.size != null) search.set('size', String(params.size))
  if (params?.importance) search.set('importance', params.importance)
  if (params?.fc_user_id != null) search.set('fc_user_id', String(params.fc_user_id))

  const response = await requestJson<ApiResult<FleetList>>(
    `/api/v1/operation/fleets${search.toString() ? `?${search.toString()}` : ''}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet list failed') ?? {
    list: [],
    total: 0,
    page: 1,
    pageSize: 20,
  }
}

export async function createFleet(data: CreateFleetParams) {
  const response = await requestJson<ApiResult<FleetItem>>('/api/v1/operation/fleets', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create fleet failed')
}

export async function fetchMyFleetList() {
  const response = await requestJson<ApiResult<FleetItem[]>>('/api/v1/operation/fleets/me', {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch my fleet list failed') ?? []
}

export async function fetchFleetDetail(id: string) {
  const response = await requestJson<ApiResult<FleetItem>>(`/api/v1/operation/fleets/${id}`, {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch fleet detail failed')
}

export async function updateFleet(id: string, data: UpdateFleetParams) {
  const response = await requestJson<ApiResult<FleetItem>>(`/api/v1/operation/fleets/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update fleet failed')
}

export async function refreshFleetESI(id: string) {
  const response = await requestJson<ApiResult<FleetItem>>(
    `/api/v1/operation/fleets/${id}/refresh-esi`,
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'refresh fleet esi failed')
}

export async function deleteFleet(id: string) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/operation/fleets/${id}`, {
    method: 'DELETE',
  })
  return assertSuccess(response, 'delete fleet failed')
}

export async function fetchFleetMembers(fleetId: string) {
  const response = await requestJson<ApiResult<FleetMember[]>>(
    `/api/v1/operation/fleets/${fleetId}/members`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet members failed') ?? []
}

export async function fetchMembersWithPap(
  fleetId: string,
  params: { current: number; size: number }
) {
  const search = new URLSearchParams({
    current: String(params.current),
    size: String(params.size),
  })
  const response = await requestJson<ApiResult<{ list: MemberWithPap[]; total: number; page: number; pageSize: number }>>(
    `/api/v1/operation/fleets/${fleetId}/members-pap?${search.toString()}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet members with pap failed')
}

export async function syncESIFleetMembers(fleetId: string) {
  const response = await requestJson<ApiResult<ESIFleetMember[]>>(
    `/api/v1/operation/fleets/${fleetId}/members/sync`,
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'sync fleet members failed') ?? []
}

export async function addFleetMembersByCharacterNames(
  fleetId: string,
  data: ManualAddFleetMembersParams
) {
  const response = await requestJson<ApiResult<ManualAddFleetMembersResult>>(
    `/api/v1/operation/fleets/${fleetId}/members/manual`,
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'add fleet members failed')
}

export async function issuePap(fleetId: string) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/operation/fleets/${fleetId}/pap`, {
    method: 'POST',
  })
  return assertSuccess(response, 'issue pap failed')
}

export async function fetchFleetPapLogs(fleetId: string) {
  const response = await requestJson<ApiResult<FleetPapLog[]>>(
    `/api/v1/operation/fleets/${fleetId}/pap`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet pap logs failed') ?? []
}

export async function fetchMyPapLogs() {
  const response = await requestJson<ApiResult<FleetPapLog[]>>(
    '/api/v1/operation/fleets/pap/me',
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch my pap logs failed') ?? []
}

export async function fetchCorporationPapSummary(params?: CorporationPapSummaryParams) {
  const search = new URLSearchParams()
  if (params?.current != null) search.set('current', String(params.current))
  if (params?.size != null) search.set('size', String(params.size))
  if (params?.period) search.set('period', params.period)
  if (params?.year != null) search.set('year', String(params.year))
  if (params?.corp_tickers) search.set('corp_tickers', params.corp_tickers)

  const response = await requestJson<ApiResult<CorporationPapSummaryList>>(
    `/api/v1/operation/fleets/pap/corporation${search.toString() ? `?${search.toString()}` : ''}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch corporation pap summary failed')
}

export async function createFleetInvite(fleetId: string) {
  const response = await requestJson<ApiResult<FleetInvite>>(
    `/api/v1/operation/fleets/${fleetId}/invites`,
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'create fleet invite failed')
}

export async function fetchFleetInvites(fleetId: string) {
  const response = await requestJson<ApiResult<FleetInvite[]>>(
    `/api/v1/operation/fleets/${fleetId}/invites`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet invites failed') ?? []
}

export async function deactivateFleetInvite(inviteId: number) {
  const response = await requestJson<ApiResult<null>>(
    `/api/v1/operation/fleets/invites/${inviteId}`,
    {
      method: 'DELETE',
    }
  )
  return assertSuccess(response, 'deactivate fleet invite failed')
}

export async function joinFleet(data: JoinFleetParams) {
  const response = await requestJson<ApiResult<null>>('/api/v1/operation/fleets/join', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'join fleet failed')
}

export async function fetchCharacterFleetInfo(characterId: number) {
  const response = await requestJson<ApiResult<CharacterFleetInfo>>(
    `/api/v1/operation/fleets/esi/${characterId}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch character fleet info failed')
}

export async function pingFleet(fleetId: string) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/operation/fleets/${fleetId}/ping`, {
    method: 'POST',
  })
  return assertSuccess(response, 'ping fleet failed')
}

export type {
  FleetItem,
  FleetList,
  FleetMember,
  MemberWithPap,
  FleetInvite,
  CorporationPapSummaryList,
  CorporationPapSummaryParams,
}
