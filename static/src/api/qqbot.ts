import request from '@/utils/http'
import type {
  QQGroupRule,
  QQGroupBusinessAudit,
  QQBotConfig,
  QQBotSnapshot,
  QQBotOperation,
  QQNotificationChannel,
  QQNotificationTarget
} from '@/types/api/qqbot'
const base = '/api/v1/system/qqbot'
export const fetchQQBotConfig = () => request.get<QQBotConfig>({ url: `${base}/config` })
export const fetchQQBotSnapshot = () => request.get<QQBotSnapshot>({ url: `${base}/snapshot` })
export const saveQQBotConfig = (data: QQBotConfig) => request.put({ url: `${base}/config`, data })
export const cancelQQBotOperation = (id: string) =>
  request.post({ url: `${base}/operations/${encodeURIComponent(id)}/cancel` })
export const retryQQBotOperation = (id: string) =>
  request.post<QQBotOperation>({ url: `${base}/operations/${encodeURIComponent(id)}/retry` })
export const retryQQBotDelivery = (id: string) =>
  request.post({ url: `${base}/deliveries/${encodeURIComponent(id)}/retry` })
export const pauseQQBotCircuit = (name: string, paused: boolean) =>
  request.put({ url: `${base}/circuits/${encodeURIComponent(name)}`, data: { paused } })

export const fetchQQBotOperation = (id: string) =>
  request.get<QQBotOperation>({ url: `${base}/operations/${encodeURIComponent(id)}` })

export const fetchQQNotificationChannels = () =>
  request.get<QQNotificationChannel[]>({ url: `${base}/notifications/channels` })
export const fetchQQNotificationTargets = () =>
  request.get<QQNotificationTarget[]>({ url: `${base}/notifications/targets` })
export const saveQQNotificationChannel = (name: string, enabled: boolean, group_ids: number[]) =>
  request.put({
    url: `${base}/notifications/channels/${encodeURIComponent(name)}`,
    data: { enabled, group_ids }
  })
export const testQQNotification = (key: string, expires_at: string) =>
  request.post<QQNotificationTarget[]>({
    url: `${base}/notifications/test`,
    data: { key, expires_at }
  })

export const fetchQQGroupBusiness = () =>
  request.get<QQGroupBusinessAudit>({ url: `${base}/groups` })
export const saveQQGroupRule = (data: QQGroupRule) =>
  request.put({ url: `${base}/groups/rules`, data })
