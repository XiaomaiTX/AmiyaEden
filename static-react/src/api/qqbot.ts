import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type { QQBotConfig, QQBotSnapshot, QQBotOperation } from '@/types/api/qqbot'
const base = '/api/v1/system/qqbot'
async function call<T>(path: string, method = 'GET', body?: unknown) {
  return assertSuccess(
    await requestJson<ApiResponse<T>>(`${base}${path}`, {
      method,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    }),
    'QQBot request failed'
  )
}
export const fetchQQBotConfig = () => call<QQBotConfig>('/config')
export const fetchQQBotSnapshot = () => call<QQBotSnapshot>('/snapshot')
export const saveQQBotConfig = (data: QQBotConfig) => call<null>('/config', 'PUT', data)
export const cancelQQBotOperation = (id: string) =>
  call<null>(`/operations/${encodeURIComponent(id)}/cancel`, 'POST')
export const retryQQBotOperation = (id: string) =>
  call<QQBotOperation>(`/operations/${encodeURIComponent(id)}/retry`, 'POST')
export const retryQQBotDelivery = (id: string) =>
  call<null>(`/deliveries/${encodeURIComponent(id)}/retry`, 'POST')
export const pauseQQBotCircuit = (name: string, paused: boolean) =>
  call<null>(`/circuits/${encodeURIComponent(name)}`, 'PUT', { paused })

export const fetchQQBotOperation = (id: string) =>
  call<QQBotOperation>(`/operations/${encodeURIComponent(id)}`)
