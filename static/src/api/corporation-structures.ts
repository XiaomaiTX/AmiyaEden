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

export function runCorporationStructuresTask(
  data: Api.Dashboard.CorporationStructureRunTaskRequest
) {
  return request.post<Api.Dashboard.CorporationStructureRunTaskResponse>({
    url: '/api/v1/dashboard/corporation-structures/run-task',
    data
  })
}
