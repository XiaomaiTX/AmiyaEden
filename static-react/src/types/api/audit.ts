import type { CommonSearchParams } from '@/types/api/common'

export type AuditResult = 'success' | 'failed'
export type AuditExportFormat = 'csv' | 'json'
export type AuditExportStatus = 'pending' | 'running' | 'done' | 'failed' | 'expired'

export interface AuditEvent {
  id: number
  event_id: string
  occurred_at: string
  category: string
  action: string
  actor_user_id: number
  target_user_id: number
  resource_type: string
  resource_id: string
  result: AuditResult
  request_id: string
  ip: string
  user_agent: string
  details_json: string
  created_at: string
}

export type AuditEventSearchParams = Partial<{
  current: number
  size: number
  start_date: string
  end_date: string
  category: string
  action: string
  actor_user_id: number
  target_user_id: number
  result: AuditResult
  request_id: string
  resource_id: string
  keyword: string
}> &
  Partial<CommonSearchParams>

export interface AuditExportTaskStatus {
  task_id: string
  status: AuditExportStatus
  format?: AuditExportFormat
  download_url?: string
  error_message?: string
  expire_at?: string
  created_at?: string
  finished_at?: string
}

export interface AuditExportCreateParams {
  format: AuditExportFormat
  filter: Omit<AuditEventSearchParams, 'current' | 'size'>
}

export interface AuditExportListParams {
  limit?: number
}
