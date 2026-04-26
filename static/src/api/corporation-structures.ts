import request from '@/utils/http'

export function fetchCorporationStructureSettings() {
  return request.get<Api.Dashboard.CorporationStructuresSettings>({
    url: '/api/v1/dashboard/corporation-structures/settings'
  })
}

export function updateCorporationStructureAuthorizations(
  data: Api.Dashboard.CorporationStructureAuthorizationUpdate
) {
  return request.put({
    url: '/api/v1/dashboard/corporation-structures/settings/authorizations',
    data
  })
}

export function fetchCorporationStructureList(data: Api.Dashboard.CorporationStructureListRequest) {
  return request.post<{ items: Api.Dashboard.CorporationStructureRow[] }>({
    url: '/api/v1/dashboard/corporation-structures/list',
    data
  })
}

export function refreshCorporationStructures(corporationId: number) {
  return request.post<{ corporation_id: number; refreshed: number; message: string }>({
    url: '/api/v1/dashboard/corporation-structures/refresh',
    data: { corporation_id: corporationId }
  })
}
