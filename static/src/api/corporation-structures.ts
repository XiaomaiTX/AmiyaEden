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
  return request.post<Api.Dashboard.CorporationStructureListResponse>({
    url: '/api/v1/dashboard/corporation-structures/list',
    data
  })
}

export function fetchCorporationStructureFilterOptions(
  params?: Api.Dashboard.CorporationStructureFilterOptionsRequest
) {
  return request.get<Api.Dashboard.CorporationStructureFilterOptionsResponse>({
    url: '/api/v1/dashboard/corporation-structures/filter-options',
    params
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

export function fetchCorporationStructureAssignments(
  params?: Api.Dashboard.CorporationStructureFilterOptionsRequest
) {
  return request.get<Api.Dashboard.CorporationStructureAssignmentListResponse>({
    url: '/api/v1/dashboard/corporation-structures/assignments',
    params
  })
}

export function updateCorporationStructureAssignments(
  data: Api.Dashboard.CorporationStructureAssignmentUpdateRequest
) {
  return request.put({
    url: '/api/v1/dashboard/corporation-structures/assignments',
    data
  })
}

export function fetchFuelSalarySettings() {
  return request.get<Api.Dashboard.FuelSalarySettingsResponse>({
    url: '/api/v1/dashboard/corporation-structures/fuel-salary-settings'
  })
}

export function updateFuelSalarySettings(data: Api.Dashboard.FuelSalarySettingsUpdateRequest) {
  return request.put({
    url: '/api/v1/dashboard/corporation-structures/fuel-salary-settings',
    data
  })
}

export function runFuelSalaryPayout(data: Api.Dashboard.FuelSalaryPayoutRunRequest) {
  return request.post<Api.Dashboard.FuelSalaryPayoutRunResponse>({
    url: '/api/v1/dashboard/corporation-structures/fuel-salary-payouts/run',
    data
  })
}

export function fetchMyAssignedCorporationStructures(
  data: Api.Dashboard.CorporationStructureListRequest
) {
  return request.post<Api.Dashboard.CorporationStructureListResponse>({
    url: '/api/v1/dashboard/corporation-structures/my-assigned-list',
    data
  })
}
