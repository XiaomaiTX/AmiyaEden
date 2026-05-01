import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  TaskHistoryList,
  TaskHistorySearchParams,
  TaskItem,
  UpdateScheduleParams,
} from '@/types/api/task-manager'
import type {
  RunTaskByNameParams,
  RunTaskParams,
  TaskInfo,
  TaskStatusList,
  TaskStatusSearchParams,
} from '@/types/api/esi-refresh'

export async function fetchTasks() {
  const response = await requestJson<ApiResponse<TaskItem[]>>('/api/v1/tasks')
  return assertSuccess(response, 'fetch tasks failed')
}

export async function fetchTaskHistory(params?: TaskHistorySearchParams) {
  const query = params ? `?${new URLSearchParams(serializeQuery(params)).toString()}` : ''
  const response = await requestJson<ApiResponse<TaskHistoryList>>(`/api/v1/tasks/history${query}`)
  return assertSuccess(response, 'fetch task history failed')
}

export async function runTask(name: string) {
  const response = await requestJson<ApiResponse<{ message: string }>>(
    `/api/v1/tasks/${name}/run`,
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'run task failed')
}

export async function updateTaskSchedule(name: string, params: UpdateScheduleParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>(
    `/api/v1/tasks/${name}/schedule`,
    {
      method: 'PUT',
      body: JSON.stringify(params),
    }
  )
  return assertSuccess(response, 'update task schedule failed')
}

export async function fetchESIRefreshTasks() {
  const response = await requestJson<ApiResponse<TaskInfo[]>>('/api/v1/tasks/esi/tasks')
  return assertSuccess(response, 'fetch esi tasks failed')
}

export async function fetchESIRefreshStatuses(params?: TaskStatusSearchParams) {
  const query = params ? `?${new URLSearchParams(serializeQuery(params)).toString()}` : ''
  const response = await requestJson<ApiResponse<TaskStatusList>>(`/api/v1/tasks/esi/statuses${query}`)
  return assertSuccess(response, 'fetch esi statuses failed')
}

export async function runESIRefreshTask(params: RunTaskParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>('/api/v1/tasks/esi/run', {
    method: 'POST',
    body: JSON.stringify(params),
  })
  return assertSuccess(response, 'run esi task failed')
}

export async function runESIRefreshTaskByName(params: RunTaskByNameParams) {
  const response = await requestJson<ApiResponse<{ message: string }>>(
    '/api/v1/tasks/esi/run-task',
    {
      method: 'POST',
      body: JSON.stringify(params),
    }
  )
  return assertSuccess(response, 'run esi task by name failed')
}

export async function runESIRefreshAll() {
  const response = await requestJson<ApiResponse<{ message: string }>>('/api/v1/tasks/esi/run-all', {
    method: 'POST',
  })
  return assertSuccess(response, 'run all esi refresh failed')
}

function serializeQuery(params: Record<string, unknown>) {
  return Object.fromEntries(
    Object.entries(params).flatMap(([key, value]) => {
      if (value === undefined || value === null || value === '') {
        return []
      }

      if (Array.isArray(value)) {
        return value.length > 0 ? [[key, value.join(',')]] : []
      }

      return [[key, String(value)]]
    })
  )
}
