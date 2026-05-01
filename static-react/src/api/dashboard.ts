import { requestJson } from '@/api/http-client'
import type { DashboardResult } from '@/types/api/dashboard'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export async function fetchDashboard() {
  const response = await requestJson<ApiResponse<DashboardResult>>('/api/v1/dashboard', {
    method: 'POST',
  })

  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || 'dashboard request failed')
  }

  return response.data
}
