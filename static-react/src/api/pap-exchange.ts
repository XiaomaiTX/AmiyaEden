import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type { ConfigResponse, UpdateConfigParams } from '@/types/api/pap-exchange'

export async function fetchPAPExchangeConfig() {
  const response = await requestJson<ApiResponse<ConfigResponse>>('/api/v1/system/pap-exchange/rates')
  return assertSuccess(response, 'fetch pap exchange config failed')
}

export async function updatePAPExchangeConfig(data: UpdateConfigParams) {
  const response = await requestJson<ApiResponse<ConfigResponse>>('/api/v1/system/pap-exchange/rates', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update pap exchange config failed')
}
