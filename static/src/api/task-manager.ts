import request from '@/utils/http'

/** 获取所有注册任务定义 */
export function fetchTasks() {
  return request.get<Api.TaskManager.TaskItem[]>({
    url: '/api/v1/tasks'
  })
}

/** 获取任务执行历史（分页 + 筛选） */
export function fetchTaskHistory(params?: Api.TaskManager.TaskHistorySearchParams) {
  return request.get<Api.TaskManager.TaskHistoryList>({
    url: '/api/v1/tasks/history',
    params
  })
}

/** 手动触发任务 */
export function runTask(name: string) {
  return request.post<{ message: string }>({
    url: `/api/v1/tasks/${name}/run`,
    showErrorMessage: false
  })
}

/** 更新任务调度 */
export function updateTaskSchedule(name: string, params: Api.TaskManager.UpdateScheduleParams) {
  return request.put<{ message: string }>({
    url: `/api/v1/tasks/${name}/schedule`,
    data: params,
    showErrorMessage: false
  })
}