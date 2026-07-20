import request from '@/utils/http'

const base = '/api/v1/system/qq-governance'

export function fetchQQGovernancePolicies() {
  return request.get<Api.QQGovernance.Policy[]>({ url: `${base}/policies` })
}
export function saveQQGovernancePolicy(
  data: Omit<Api.QQGovernance.Policy, 'id' | 'updated_by' | 'updated_at'>
) {
  return request.post<Api.QQGovernance.Policy>({ url: `${base}/policies`, data })
}
export function updateQQGovernancePolicy(
  groupId: number,
  data: Omit<Api.QQGovernance.Policy, 'id' | 'group_id' | 'updated_by' | 'updated_at'>
) {
  return request.put<Api.QQGovernance.Policy>({ url: `${base}/policies/${groupId}`, data })
}
export function deleteQQGovernancePolicy(groupId: number) {
  return request.del({ url: `${base}/policies/${groupId}` })
}
export function fetchQQGovernanceMembers(params?: Api.QQGovernance.PageParams) {
  return request.get<Api.QQGovernance.PageResult<Api.QQGovernance.MemberState>>({
    url: `${base}/members`,
    params
  })
}
export function fetchQQGovernanceReviews(params?: Api.QQGovernance.PageParams) {
  return request.get<Api.QQGovernance.PageResult<Api.QQGovernance.Review>>({
    url: `${base}/reviews`,
    params
  })
}
export function fetchQQGovernanceTasks(params?: Api.QQGovernance.PageParams) {
  return request.get<Api.QQGovernance.PageResult<Api.QQGovernance.ActionTask>>({
    url: `${base}/tasks`,
    params
  })
}
export function retryQQGovernanceTask(id: number) {
  return request.post({ url: `${base}/tasks/${id}/retry` })
}
export function fetchQQGovernanceAlerts(params?: Api.QQGovernance.PageParams) {
  return request.get<Api.QQGovernance.PageResult<Api.QQGovernance.Alert>>({
    url: `${base}/alerts`,
    params
  })
}
export function acknowledgeQQGovernanceAlert(id: number) {
  return request.post({ url: `${base}/alerts/${id}/acknowledge` })
}
export function fetchQQGovernanceMetrics() {
  return request.get<Api.QQGovernance.Metrics>({ url: `${base}/metrics` })
}
export function fetchQQGovernanceConnection() {
  return request.get<Api.QQGovernance.Connection>({ url: `${base}/connection` })
}
export function fetchQQGovernanceGroups() {
  return request.get<Api.QQGovernance.GroupStatus[]>({ url: `${base}/groups` })
}
export function searchQQGovernanceCorporations(query: string) {
  return request.get<Api.QQGovernance.CorporationOption[]>({
    url: `${base}/corporations`,
    params: { query }
  })
}
export function fetchQQGovernanceSettings() {
  return request.get<Api.QQGovernance.Settings>({ url: `${base}/settings` })
}
export function updateQQGovernanceSettings(data: Api.QQGovernance.Settings) {
  return request.put<Api.QQGovernance.Settings>({ url: `${base}/settings`, data })
}
export function triggerQQGovernanceReconcile(groupId: number) {
  return request.post({ url: `${base}/groups/${groupId}/reconcile` })
}
export function runQQGovernanceManualAction(data: {
  action: string
  group_id: number
  qq: number
}) {
  return request.post({ url: `${base}/actions`, data })
}
export function resetQQGovernanceRisk() {
  return request.post({ url: `${base}/risk-control/reset` })
}
