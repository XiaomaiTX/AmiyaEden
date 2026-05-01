import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  CreateFleetConfigParams,
  EFTResponse,
  ExportToESIParams,
  FittingItemsResponse,
  FleetConfigItem,
  FleetConfigList,
  ImportFittingParams,
  ImportFittingResponse,
  UpdateFleetConfigParams,
  UpdateItemsSettingsParams,
} from '@/types/api/fleet-config'

type ApiResult<T> = ApiResponse<T>

export async function createFleetConfig(data: CreateFleetConfigParams) {
  const response = await requestJson<ApiResult<FleetConfigItem>>('/api/v1/operation/fleet-configs', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create fleet config failed')
}

export async function fetchFleetConfigList(params?: { current?: number; size?: number }) {
  const search = new URLSearchParams()
  if (params?.current != null) search.set('current', String(params.current))
  if (params?.size != null) search.set('size', String(params.size))

  const response = await requestJson<ApiResult<FleetConfigList>>(
    `/api/v1/operation/fleet-configs${search.toString() ? `?${search.toString()}` : ''}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet config list failed')
}

export async function fetchFleetConfigDetail(id: number) {
  const response = await requestJson<ApiResult<FleetConfigItem>>(
    `/api/v1/operation/fleet-configs/${id}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet config detail failed')
}

export async function updateFleetConfig(id: number, data: UpdateFleetConfigParams) {
  const response = await requestJson<ApiResult<FleetConfigItem>>(
    `/api/v1/operation/fleet-configs/${id}`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'update fleet config failed')
}

export async function deleteFleetConfig(id: number) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/operation/fleet-configs/${id}`, {
    method: 'DELETE',
  })
  return assertSuccess(response, 'delete fleet config failed')
}

export async function importFittingFromUser(data: ImportFittingParams) {
  const response = await requestJson<ApiResult<ImportFittingResponse>>(
    '/api/v1/operation/fleet-configs/import-fitting',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'import fitting failed')
}

export async function exportFittingToESI(data: ExportToESIParams) {
  const response = await requestJson<ApiResult<null>>(
    '/api/v1/operation/fleet-configs/export-esi',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'export fitting to esi failed')
}

export async function fetchFleetConfigEFT(id: number, lang?: string) {
  const search = new URLSearchParams()
  if (lang) search.set('lang', lang)
  const response = await requestJson<ApiResult<EFTResponse>>(
    `/api/v1/operation/fleet-configs/${id}/eft${search.toString() ? `?${search.toString()}` : ''}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fleet config eft failed')
}

export async function fetchFittingItems(configId: number, fittingId: number, lang?: string) {
  const search = new URLSearchParams()
  if (lang) search.set('lang', lang)
  const response = await requestJson<ApiResult<FittingItemsResponse>>(
    `/api/v1/operation/fleet-configs/${configId}/fittings/${fittingId}/items${search.toString() ? `?${search.toString()}` : ''}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch fitting items failed')
}

export async function updateFittingItemsSettings(
  configId: number,
  fittingId: number,
  data: UpdateItemsSettingsParams
) {
  const response = await requestJson<ApiResult<null>>(
    `/api/v1/operation/fleet-configs/${configId}/fittings/${fittingId}/items/settings`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'update fitting items settings failed')
}

export type { FleetConfigItem, FleetConfigList }
