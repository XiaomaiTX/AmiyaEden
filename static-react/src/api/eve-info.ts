import { requestJson } from '@/api/http-client'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export async function fetchInfoWallet(data: Api.EveInfo.WalletRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.WalletResponse>>('/api/v1/info/wallet', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch wallet failed')
  }

  return response.data
}

export async function fetchInfoSkills(data: Api.EveInfo.SkillRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.SkillResponse>>('/api/v1/info/skills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch skills failed')
  }

  return response.data
}

export async function fetchInfoShips(data: Api.EveInfo.ShipRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ShipResponse>>('/api/v1/info/ships', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch ships failed')
  }

  return response.data
}

export async function fetchInfoImplants(data: Api.EveInfo.ImplantsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ImplantsResponse>>('/api/v1/info/implants', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch implants failed')
  }

  return response.data
}

export async function fetchInfoFittings(data: Api.EveInfo.FittingsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.FittingsListResponse>>('/api/v1/info/fittings', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch fittings failed')
  }

  return response.data
}

export async function fetchInfoAssets(data: Api.EveInfo.AssetsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.AssetsResponse>>('/api/v1/info/assets', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'fetch assets failed')
  }

  return response.data
}

export async function runMyCharacterESIRefresh(data: Api.ESIRefresh.RunTaskParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>('/api/v1/info/esi-refresh', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  if (response.code !== 0) {
    throw new Error(response.msg || 'trigger esi refresh failed')
  }

  return response.data
}
