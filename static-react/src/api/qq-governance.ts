import { requestJson } from '@/api/http-client'
import type {
  QQActionTask,
  QQAlert,
  QQConnection,
  QQGroupStatus,
  QQMetrics,
  QQPageParams,
  QQPageResult,
  QQPolicy,
  QQSettings,
} from '@/types/api/qq-governance'

interface ApiResponse<T> { code: number; msg: string; data: T }
const base = '/api/v1/system/qq-governance'

function unwrap<T>(response: ApiResponse<T>) {
  if (response.code !== 0 && response.code !== 200) throw new Error(response.msg)
  return response.data
}

function query(params?: QQPageParams) {
  const search = new URLSearchParams()
  Object.entries(params ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== '') search.set(key, String(value))
  })
  return search.size ? `?${search}` : ''
}

export const fetchQQGovernancePolicies = async () =>
  unwrap(await requestJson<ApiResponse<QQPolicy[]>>(`${base}/policies`))
export const fetchQQGovernanceGroups = async () =>
  unwrap(await requestJson<ApiResponse<QQGroupStatus[]>>(`${base}/groups`))
export const fetchQQGovernanceMetrics = async () =>
  unwrap(await requestJson<ApiResponse<QQMetrics>>(`${base}/metrics`))
export const fetchQQGovernanceConnection = async () =>
  unwrap(await requestJson<ApiResponse<QQConnection>>(`${base}/connection`))
export const fetchQQGovernanceTasks = async (params?: QQPageParams) =>
  unwrap(await requestJson<ApiResponse<QQPageResult<QQActionTask>>>(`${base}/tasks${query(params)}`))
export const fetchQQGovernanceAlerts = async (params?: QQPageParams) =>
  unwrap(await requestJson<ApiResponse<QQPageResult<QQAlert>>>(`${base}/alerts${query(params)}`))
export const fetchQQGovernanceSettings = async () =>
  unwrap(await requestJson<ApiResponse<QQSettings>>(`${base}/settings`))

export async function updateQQGovernanceSettings(data: QQSettings) {
  return unwrap(await requestJson<ApiResponse<QQSettings>>(`${base}/settings`, {
    method: 'PUT', body: JSON.stringify(data),
  }))
}
export async function retryQQGovernanceTask(id: number) {
  return unwrap(await requestJson<ApiResponse<unknown>>(`${base}/tasks/${id}/retry`, { method: 'POST' }))
}
export async function recoverQQGovernanceDisconnectedTasks() {
  return unwrap(await requestJson<ApiResponse<{ recovered_tasks: number }>>(`${base}/tasks/recover-disconnected`, { method: 'POST' }))
}
export async function acknowledgeQQGovernanceAlert(id: number) {
  return unwrap(await requestJson<ApiResponse<unknown>>(`${base}/alerts/${id}/acknowledge`, { method: 'POST' }))
}
export async function triggerQQGovernanceReconcile(groupId: number) {
  return unwrap(await requestJson<ApiResponse<unknown>>(`${base}/groups/${groupId}/reconcile`, { method: 'POST' }))
}
export async function resetQQGovernanceRisk() {
  return unwrap(await requestJson<ApiResponse<unknown>>(`${base}/risk-control/reset`, { method: 'POST' }))
}
