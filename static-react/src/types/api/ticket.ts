import type { CommonSearchParams, PaginatedResponse } from '@/types/api/common'

export type TicketStatus = 'pending' | 'in_progress' | 'completed'
export type TicketPriority = 'low' | 'medium' | 'high'

export interface TicketItem {
  id: number
  user_id: number
  category_id: number
  title: string
  description: string
  status: TicketStatus
  priority: TicketPriority
  handled_by?: number
  handled_at?: string
  closed_at?: string
  created_at: string
  updated_at: string
}

export interface TicketCategory {
  id: number
  name: string
  name_en: string
  description: string
  sort_order: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface TicketReply {
  id: number
  ticket_id: number
  user_id: number
  content: string
  is_internal: boolean
  created_at: string
  updated_at: string
}

export interface TicketListParams extends Partial<CommonSearchParams> {
  status?: TicketStatus | ''
}

export interface AdminTicketListParams extends Partial<CommonSearchParams> {
  status?: TicketStatus | ''
  keyword?: string
  category_id?: number
  user_id?: number
}

export interface CreateTicketParams {
  category_id: number
  title: string
  description: string
  priority?: TicketPriority
}

export interface AddReplyParams {
  content: string
}

export interface AdminAddReplyParams extends AddReplyParams {
  is_internal?: boolean
}

export interface UpdateStatusParams {
  status: TicketStatus
}

export interface UpdatePriorityParams {
  priority: TicketPriority
}

export interface UpsertCategoryParams {
  name: string
  name_en: string
  description?: string
  sort_order?: number
  enabled?: boolean
}

export interface TicketStatusHistory {
  id: number
  ticket_id: number
  from_status: string
  to_status: TicketStatus
  changed_by: number
  changed_at: string
}

export interface Statistics {
  total: number
  status: Record<TicketStatus, number>
  category: Record<string, number>
  recent_7d: number
  recent_30d: number
  pendingCount: number
}

export type TicketListResponse = PaginatedResponse<TicketItem>
