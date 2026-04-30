import request from '@/utils/http'

/** 管理端审计事件分页查询 */
export function adminListAuditEvents(data?: Api.Audit.AuditEventSearchParams) {
  return request.post<Api.Common.PaginatedResponse<Api.Audit.AuditEvent>>({
    url: '/api/v1/system/audit/events',
    data: data ?? { current: 1, size: 200 }
  })
}

/** 创建审计导出任务 */
export function createAuditExportTask(data: Api.Audit.AuditExportCreateParams) {
  return request.post<Api.Audit.AuditExportTaskStatus>({
    url: '/api/v1/system/audit/export',
    data
  })
}

/** 查询审计导出任务状态 */
export function getAuditExportTaskStatus(taskId: string) {
  return request.get<Api.Audit.AuditExportTaskStatus>({
    url: `/api/v1/system/audit/export/${taskId}`
  })
}

/** 查询审计导出任务历史 */
export function listAuditExportTasks(data?: Api.Audit.AuditExportListParams) {
  return request.post<Api.Audit.AuditExportTaskStatus[]>({
    url: '/api/v1/system/audit/export/list',
    data: data ?? { limit: 20 }
  })
}
