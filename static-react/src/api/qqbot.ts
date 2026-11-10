import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  QQGroupRule,
  QQGroupBusinessAudit,
  QQBotConfig,
  QQBotSnapshot,
  QQBotOperation,
  QQNotificationChannel,
  QQNotificationTarget,
} from '@/types/api/qqbot'
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

export const fetchQQNotificationChannels = () =>
  call<QQNotificationChannel[]>('/notifications/channels')
export const fetchQQNotificationTargets = () =>
  call<QQNotificationTarget[]>('/notifications/targets')
export const saveQQNotificationChannel = (name: string, enabled: boolean, group_ids: number[]) =>
  call<null>(`/notifications/channels/${encodeURIComponent(name)}`, 'PUT', { enabled, group_ids })
export const testQQNotification = (key: string, expires_at: string) =>
  call<QQNotificationTarget[]>('/notifications/test', 'POST', { key, expires_at })

export const fetchQQGroupBusiness = () => call<QQGroupBusinessAudit>('/groups')
export const saveQQGroupRule = (data: QQGroupRule) => call<null>('/groups/rules', 'PUT', data)
