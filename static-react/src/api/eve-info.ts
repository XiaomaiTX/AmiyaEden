import { requestJson } from '@/api/http-client'
import type {
  AssetsRequest,
  AssetsResponse,
  ContractDetailRequest,
  ContractDetailResponse,
  ContractsRequest,
  ContractsResponse,
  FittingsListResponse,
  FittingsRequest,
  ImplantsRequest,
  ImplantsResponse,
  RunTaskParams,
  ShipRequest,
  ShipResponse,
  SkillRequest,
  SkillResponse,
  WalletRequest,
  WalletResponse,
} from '@/types/api/eve-info'

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

export async function fetchInfoWallet(data: WalletRequest) {
  const response = await requestJson<ApiResponse<WalletResponse>>('/api/v1/info/wallet', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'fetch wallet failed')
}

export async function fetchInfoSkills(data: SkillRequest) {
  const response = await requestJson<ApiResponse<SkillResponse>>('/api/v1/info/skills', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch skills failed')
}

export async function fetchInfoShips(data: ShipRequest) {
  const response = await requestJson<ApiResponse<ShipResponse>>('/api/v1/info/ships', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch ships failed')
}

export async function fetchInfoImplants(data: ImplantsRequest) {
  const response = await requestJson<ApiResponse<ImplantsResponse>>('/api/v1/info/implants', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch implants failed')
}

export async function fetchInfoFittings(data: FittingsRequest) {
  const response = await requestJson<ApiResponse<FittingsListResponse>>('/api/v1/info/fittings', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch fittings failed')
}

export async function fetchInfoAssets(data: AssetsRequest) {
  const response = await requestJson<ApiResponse<AssetsResponse>>('/api/v1/info/assets', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch assets failed')
}

export async function fetchInfoContracts(data: ContractsRequest) {
  const response = await requestJson<ApiResponse<ContractsResponse>>('/api/v1/info/contracts', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'fetch contracts failed')
}

export async function fetchInfoContractDetail(data: ContractDetailRequest) {
  const response = await requestJson<ApiResponse<ContractDetailResponse>>(
    '/api/v1/info/contracts/detail',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'fetch contract detail failed')
}

export async function runMyCharacterESIRefresh(data: RunTaskParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>('/api/v1/info/esi-refresh', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'trigger esi refresh failed')
}
