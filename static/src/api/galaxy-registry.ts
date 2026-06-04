import request from '@/utils/http'

export function fetchGalaxyRegistrySystems() {
  return request.get<Api.Dashboard.GalaxyRegistrySystemsResponse>({
    url: '/api/v1/dashboard/galaxy-registry/systems'
  })
}

export function createGalaxyRegistryEntry(data: Api.Dashboard.GalaxyRegistryCreateEntryRequest) {
  return request.post<Api.Dashboard.GalaxyRegistryEntryItem>({
    url: '/api/v1/dashboard/galaxy-registry/entries',
    data
  })
}

export function endGalaxyRegistryEntry(id: number) {
  return request.post<Api.Dashboard.GalaxyRegistryEntryItem>({
    url: `/api/v1/dashboard/galaxy-registry/entries/${id}/end`
  })
}

export function fetchMyGalaxyRegistryEntries(params?: Api.Dashboard.GalaxyRegistryEntryListParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Dashboard.GalaxyRegistryEntryItem>>({
    url: '/api/v1/dashboard/galaxy-registry/my-entries',
    params
  })
}

export function searchGalaxyRegistrySdeSystems(
  params: Api.Dashboard.GalaxyRegistrySdeSystemSearchParams
) {
  return request.get<Api.Dashboard.GalaxyRegistrySdeSystem[]>({
    url: '/api/v1/dashboard/galaxy-registry/admin/sde-systems',
    params
  })
}

export function fetchAdminGalaxyRegistrySystems() {
  return request.get<Api.Dashboard.GalaxyRegistryAdminSystem[]>({
    url: '/api/v1/dashboard/galaxy-registry/admin/systems'
  })
}

export function createAdminGalaxyRegistrySystem(
  data: Api.Dashboard.GalaxyRegistryAdminCreateSystemRequest
) {
  return request.post<Api.Dashboard.GalaxyRegistryAdminSystem>({
    url: '/api/v1/dashboard/galaxy-registry/admin/systems',
    data
  })
}

export function updateAdminGalaxyRegistrySystem(
  id: number,
  data: Api.Dashboard.GalaxyRegistryAdminUpdateSystemRequest
) {
  return request.put<Api.Dashboard.GalaxyRegistryAdminSystem>({
    url: `/api/v1/dashboard/galaxy-registry/admin/systems/${id}`,
    data
  })
}

export function deleteAdminGalaxyRegistrySystem(id: number) {
  return request.del({
    url: `/api/v1/dashboard/galaxy-registry/admin/systems/${id}`
  })
}

export function fetchAdminGalaxyRegistryEntries(
  params?: Api.Dashboard.GalaxyRegistryEntryListParams
) {
  return request.get<Api.Common.PaginatedResponse<Api.Dashboard.GalaxyRegistryEntryItem>>({
    url: '/api/v1/dashboard/galaxy-registry/admin/entries',
    params
  })
}

export function forceEndAdminGalaxyRegistryEntry(id: number) {
  return request.post<Api.Dashboard.GalaxyRegistryEntryItem>({
    url: `/api/v1/dashboard/galaxy-registry/admin/entries/${id}/force-end`
  })
}

export function fetchAdminGalaxyRegistryAnalytics(
  params?: Api.Dashboard.GalaxyRegistryAnalyticsParams
) {
  return request.get<Api.Dashboard.GalaxyRegistryAdminAnalytics>({
    url: '/api/v1/dashboard/galaxy-registry/admin/analytics',
    params
  })
}
