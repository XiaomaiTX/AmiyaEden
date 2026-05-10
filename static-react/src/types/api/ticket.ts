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

export interface CreateTicketParams {
  category_id: number
  title: string
  description: string
  priority?: TicketPriority
}

export interface AddReplyParams {
  content: string
}

export type TicketListResponse = PaginatedResponse<TicketItem>
