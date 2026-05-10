import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export interface TaskItem {
  name: string
  description: string
  category: 'esi' | 'operation' | 'system'
  type: 'recurring' | 'triggered'
  runnable: boolean
  cron_expr: string
  default_cron: string
  last_execution?: TaskLastExecution | null
}

export interface TaskLastExecution {
  status: 'running' | 'success' | 'failed'
  started_at: string
  finished_at?: string
  duration_ms?: number
  error?: string
  summary?: string
}

export interface TaskExecutionItem {
  id: number
  task_name: string
  trigger: 'cron' | 'manual'
  triggered_by?: number
  triggered_by_name?: string
  status: 'running' | 'success' | 'failed'
  started_at: string
  finished_at?: string
  duration_ms?: number
  error?: string
  summary?: string
}

export interface UpdateScheduleParams {
  cron_expr: string
}

export type TaskHistorySearchParams = Partial<{
  task_name: string
  status: string
}> &
  Partial<CommonSearchParams>

export type TaskHistoryList = PaginatedResponse<TaskExecutionItem>
