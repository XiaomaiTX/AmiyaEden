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

export async function fetchCorporationStructureSettings() {
  const response = await requestJson<ApiResponse<Api.Dashboard.CorporationStructuresSettings>>(
    '/api/v1/dashboard/corporation-structures/settings'
  )

  return assertSuccess(response, 'fetch corporation structure settings failed')
}

export async function updateCorporationStructureAuthorizations(
  data: Api.Dashboard.CorporationStructureAuthorizationUpdate
) {
  const response = await requestJson<ApiResponse<Api.Dashboard.CorporationStructuresSettings>>(
    '/api/v1/dashboard/corporation-structures/settings/authorizations',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'update corporation structure authorizations failed')
}

export async function fetchCorporationStructureList(data: Api.Dashboard.CorporationStructureListRequest) {
  const response = await requestJson<ApiResponse<Api.Dashboard.CorporationStructureListResponse>>(
    '/api/v1/dashboard/corporation-structures/list',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'fetch corporation structure list failed')
}

export async function fetchCorporationStructureFilterOptions(
  params?: Api.Dashboard.CorporationStructureFilterOptionsRequest
) {
  const search = params?.corporation_id ? `?corporation_id=${params.corporation_id}` : ''
  const response = await requestJson<ApiResponse<Api.Dashboard.CorporationStructureFilterOptionsResponse>>(
    `/api/v1/dashboard/corporation-structures/filter-options${search}`
  )

  return assertSuccess(response, 'fetch corporation structure filter options failed')
}

export async function runCorporationStructuresTask(
  data: Api.Dashboard.CorporationStructureRunTaskRequest
) {
  const response = await requestJson<ApiResponse<Api.Dashboard.CorporationStructureRunTaskResponse>>(
    '/api/v1/dashboard/corporation-structures/run-task',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'run corporation structures task failed')
}
