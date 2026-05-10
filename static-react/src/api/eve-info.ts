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

export async function fetchInfoWallet(data: Api.EveInfo.WalletRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.WalletResponse>>('/api/v1/info/wallet', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'fetch wallet failed')
}

export async function fetchInfoSkills(data: Api.EveInfo.SkillRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.SkillResponse>>('/api/v1/info/skills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch skills failed')
}

export async function fetchInfoShips(data: Api.EveInfo.ShipRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ShipResponse>>('/api/v1/info/ships', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch ships failed')
}

export async function fetchInfoImplants(data: Api.EveInfo.ImplantsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ImplantsResponse>>('/api/v1/info/implants', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch implants failed')
}

export async function fetchInfoFittings(data: Api.EveInfo.FittingsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.FittingsListResponse>>('/api/v1/info/fittings', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch fittings failed')
}

export async function fetchInfoAssets(data: Api.EveInfo.AssetsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.AssetsResponse>>('/api/v1/info/assets', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch assets failed')
}

export async function fetchInfoContracts(data: Api.EveInfo.ContractsRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ContractsResponse>>('/api/v1/info/contracts', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch contracts failed')
}

export async function fetchInfoContractDetail(data: Api.EveInfo.ContractDetailRequest) {
  const response = await requestJson<ApiResponse<Api.EveInfo.ContractDetailResponse>>(
    '/api/v1/info/contracts/detail',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'fetch contract detail failed')
}

export async function runMyCharacterESIRefresh(data: Api.ESIRefresh.RunTaskParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>('/api/v1/info/esi-refresh', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'trigger esi refresh failed')
}
