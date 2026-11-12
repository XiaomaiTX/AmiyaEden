import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type { SalarySettings, OperationSalaryRates } from '@/types/api/salary'
export async function fetchSalarySettings() {
  return assertSuccess(
    await requestJson<ApiResponse<SalarySettings>>('/api/v1/system/salary-settings'),
    'fetch salary settings failed'
  )
}
export async function updateSalarySettings(data: SalarySettings) {
  return assertSuccess(
    await requestJson<ApiResponse<SalarySettings>>('/api/v1/system/salary-settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
    'update salary settings failed'
  )
}
export async function fetchOperationSalaryRates() {
  return assertSuccess(
    await requestJson<ApiResponse<OperationSalaryRates>>('/api/v1/operation-salary-rates'),
    'fetch operation salary rates failed'
  )
}
