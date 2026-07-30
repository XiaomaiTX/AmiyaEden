import { requestJson } from '@/api/http-client'
import type {
  GalaxyRegistryAdminSystem,
  GalaxyRegistryEntryItem,
  GalaxyRegistryEntryListParams,
  GalaxyRegistryEntryPage,
  GalaxyRegistrySystemsResponse,
} from '@/types/api/galaxy-registry'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

const base = '/api/v1/dashboard/galaxy-registry'

function unwrap<T>(response: ApiResponse<T>) {
  if (response.code !== 0 && response.code !== 200) throw new Error(response.msg)
  return response.data
}

function query(params?: Record<string, unknown>) {
  const search = new URLSearchParams()
  Object.entries(params ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== '') search.set(key, String(value))
  })
  const value = search.toString()
  return value ? `?${value}` : ''
}

export async function fetchGalaxyRegistrySystems() {
  return unwrap(await requestJson<ApiResponse<GalaxyRegistrySystemsResponse>>(`${base}/systems`))
}

export async function createGalaxyRegistryEntry(data: {
  system_config_id: number
  expected_end_at: string
}) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryItem>>(`${base}/entries`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  )
}

export async function endGalaxyRegistryEntry(id: number) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryItem>>(`${base}/entries/${id}/end`, {
      method: 'POST',
    })
  )
}

export async function fetchMyGalaxyRegistryEntries(params?: GalaxyRegistryEntryListParams) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryPage>>(
      `${base}/my-entries${query(params as Record<string, unknown>)}`
    )
  )
}

export async function fetchAdminGalaxyRegistrySystems() {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryAdminSystem[]>>(`${base}/admin/systems`)
  )
}

export async function fetchAdminGalaxyRegistryEntries(params?: GalaxyRegistryEntryListParams) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryPage>>(
      `${base}/admin/entries${query(params as Record<string, unknown>)}`
    )
  )
}

export async function forceEndAdminGalaxyRegistryEntry(id: number) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryItem>>(
      `${base}/admin/entries/${id}/force-end`,
      { method: 'POST' }
    )
  )
}

export async function revalidateAdminGalaxyRegistryEntry(id: number) {
  return unwrap(
    await requestJson<ApiResponse<GalaxyRegistryEntryItem>>(
      `${base}/admin/entries/${id}/revalidate`,
      { method: 'POST' }
    )
  )
}
