import request from '@/utils/http'
import type { QQBotConfig, QQBotSnapshot, QQBotOperation } from '@/types/api/qqbot'
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
