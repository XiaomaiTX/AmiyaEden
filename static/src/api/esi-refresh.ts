import request from '@/utils/http'

/** 获取所有 ESI 刷新任务定义 */
export function fetchESIRefreshTasks() {
  return request.get<Api.ESIRefresh.TaskInfo[]>({
    url: '/api/v1/tasks/esi/tasks'
  })
}

/** 获取任务运行时状态（分页 + 筛选） */
export function fetchESIRefreshStatuses(params?: Api.ESIRefresh.TaskStatusSearchParams) {
  return request.get<Api.ESIRefresh.TaskStatusList>({
    url: '/api/v1/tasks/esi/statuses',
    params
  })
}

/** 手动触发指定任务（单人物） */
export function runESIRefreshTask(params: Api.ESIRefresh.RunTaskParams) {
  return request.post<{ message: string }>({
    url: '/api/v1/tasks/esi/run',
    data: params
  })
}

/** 手动触发指定任务（所有人物） */
export function runESIRefreshTaskByName(params: Api.ESIRefresh.RunTaskByNameParams) {
  return request.post<{ message: string }>({
    url: '/api/v1/tasks/esi/run-task',
    data: params
  })
}

/** 手动触发全量刷新 */
export function runESIRefreshAll() {
  return request.post<{ message: string }>({
    url: '/api/v1/tasks/esi/run-all'
  })
}
