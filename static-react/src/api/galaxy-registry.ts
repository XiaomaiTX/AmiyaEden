import { requestJson } from '@/api/http-client'
import type {
  GalaxyRegistryAdminAnalytics, GalaxyRegistryAdminCreateSystemRequest, GalaxyRegistryAdminSystem,
  GalaxyRegistryAdminUpdateSystemRequest, GalaxyRegistryAdminUpdateValidationRequest,
  GalaxyRegistryCreateEntryRequest, GalaxyRegistryEntryItem, GalaxyRegistryEntryListParams,
  GalaxyRegistryEntryPage, GalaxyRegistrySdeSystem, GalaxyRegistrySystemsResponse,
  GalaxyRegistryUpdateExpectedEndAtRequest,
} from '@/types/api/galaxy-registry'

interface ApiResponse<T> { code: number; msg: string; data: T }
const base = '/api/v1/dashboard/galaxy-registry'
function unwrap<T>(response: ApiResponse<T>) { if (response.code !== 0 && response.code !== 200) throw new Error(response.msg); return response.data }
function query(params?: Record<string, unknown>) { const search = new URLSearchParams(); Object.entries(params ?? {}).forEach(([key, value]) => { if (value !== undefined && value !== '') search.set(key, String(value)) }); return search.size ? `?${search}` : '' }
const get = <T>(path: string) => requestJson<ApiResponse<T>>(path).then(unwrap)
const send = <T>(path: string, method: 'POST' | 'PUT' | 'DELETE', body?: unknown) => requestJson<ApiResponse<T>>(path, { method, ...(body === undefined ? {} : { body: JSON.stringify(body) }) }).then(unwrap)

export const fetchGalaxyRegistrySystems = () => get<GalaxyRegistrySystemsResponse>(`${base}/systems`)
export const createGalaxyRegistryEntry = (data: GalaxyRegistryCreateEntryRequest) => send<GalaxyRegistryEntryItem>(`${base}/entries`, 'POST', data)
export const endGalaxyRegistryEntry = (id: number) => send<GalaxyRegistryEntryItem>(`${base}/entries/${id}/end`, 'POST')
export const updateGalaxyRegistryEntryExpectedEndAt = (id: number, data: GalaxyRegistryUpdateExpectedEndAtRequest) => send<GalaxyRegistryEntryItem>(`${base}/entries/${id}/expected-end-at`, 'PUT', data)
export const fetchMyGalaxyRegistryEntries = (params?: GalaxyRegistryEntryListParams) => get<GalaxyRegistryEntryPage>(`${base}/my-entries${query(params)}`)
export const searchGalaxyRegistrySdeSystems = (params: { keyword: string; limit?: number }) => get<GalaxyRegistrySdeSystem[]>(`${base}/admin/sde-systems${query(params)}`)
export const fetchAdminGalaxyRegistrySystems = () => get<GalaxyRegistryAdminSystem[]>(`${base}/admin/systems`)
export const createAdminGalaxyRegistrySystem = (data: GalaxyRegistryAdminCreateSystemRequest) => send<GalaxyRegistryAdminSystem>(`${base}/admin/systems`, 'POST', data)
export const updateAdminGalaxyRegistrySystem = (id: number, data: GalaxyRegistryAdminUpdateSystemRequest) => send<GalaxyRegistryAdminSystem>(`${base}/admin/systems/${id}`, 'PUT', data)
export const deleteAdminGalaxyRegistrySystem = (id: number) => send<unknown>(`${base}/admin/systems/${id}`, 'DELETE')
export const fetchAdminGalaxyRegistryEntries = (params?: GalaxyRegistryEntryListParams) => get<GalaxyRegistryEntryPage>(`${base}/admin/entries${query(params)}`)
export const forceEndAdminGalaxyRegistryEntry = (id: number) => send<GalaxyRegistryEntryItem>(`${base}/admin/entries/${id}/force-end`, 'POST')
export const revalidateAdminGalaxyRegistryEntry = (id: number) => send<GalaxyRegistryEntryItem>(`${base}/admin/entries/${id}/revalidate`, 'POST')
export const updateAdminGalaxyRegistryEntryValidation = (id: number, data: GalaxyRegistryAdminUpdateValidationRequest) => send<GalaxyRegistryEntryItem>(`${base}/admin/entries/${id}/validation`, 'PUT', data)
export const fetchAdminGalaxyRegistryAnalytics = (params?: { start_date?: string; end_date?: string }) => get<GalaxyRegistryAdminAnalytics>(`${base}/admin/analytics${query(params)}`)
