import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export interface TaskInfo {
  name: string
  description: string
  priority: number
  active_interval: string
  inactive_interval: string
  required_scopes: string[]
}

export interface TaskStatus {
  task_name: string
  description: string
  character_id: number
  character_name?: string
  priority: number
  last_run?: string | null
  next_run?: string | null
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  error?: string
}

export interface RunTaskParams {
  task_name: string
  character_id: number
}

export interface RunTaskByNameParams {
  task_name: string
}

export type TaskStatusSearchParams = Partial<{
  character: string
  task_name: string
  status: string
}> &
  Partial<CommonSearchParams>

export type TaskStatusList = PaginatedResponse<TaskStatus>
