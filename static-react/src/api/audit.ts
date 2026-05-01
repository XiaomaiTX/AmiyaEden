import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AuditEvent,
  AuditEventSearchParams,
  AuditExportCreateParams,
  AuditExportListParams,
  AuditExportTaskStatus,
} from '@/types/api/audit'
import type { PaginatedResponse } from '@/types/api/common'

export async function fetchAuditEvents(params?: AuditEventSearchParams) {
  const response = await requestJson<ApiResponse<PaginatedResponse<AuditEvent>>>(
    '/api/v1/system/audit/events',
    {
      method: 'POST',
      body: JSON.stringify(params ?? { current: 1, size: 200 }),
    }
  )
  return assertSuccess(response, 'fetch audit events failed')
}

export async function createAuditExportTask(data: AuditExportCreateParams) {
  const response = await requestJson<ApiResponse<AuditExportTaskStatus>>(
    '/api/v1/system/audit/export',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'create audit export task failed')
}

export async function getAuditExportTaskStatus(taskId: string) {
  const response = await requestJson<ApiResponse<AuditExportTaskStatus>>(
    `/api/v1/system/audit/export/${taskId}`
  )
  return assertSuccess(response, 'fetch audit export task status failed')
}

export async function listAuditExportTasks(data?: AuditExportListParams) {
  const response = await requestJson<ApiResponse<AuditExportTaskStatus[]>>(
    '/api/v1/system/audit/export/list',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { limit: 20 }),
    }
  )
  return assertSuccess(response, 'fetch audit export tasks failed')
}
