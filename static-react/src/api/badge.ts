import { requestJson } from '@/api/http-client'
import type { BadgeCounts } from '@/types/api/badge'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export async function fetchBadgeCounts() {
  const response = await requestJson<ApiResponse<BadgeCounts>>('/api/v1/badge-counts')
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg)
  }
  return response.data ?? {}
}
