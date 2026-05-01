import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AdminAddReplyParams,
  AdminTicketListParams,
  AddReplyParams,
  CreateTicketParams,
  TicketCategory,
  TicketItem,
  TicketListParams,
  TicketListResponse,
  TicketReply,
  TicketStatusHistory,
  Statistics,
  UpdatePriorityParams,
  UpdateStatusParams,
  UpsertCategoryParams,
} from '@/types/api/ticket'
type ApiResult<T> = ApiResponse<T>

export async function createTicket(data: CreateTicketParams) {
  const response = await requestJson<ApiResponse<TicketItem>>('/api/v1/ticket/tickets', {
    method: 'POST',
    body: JSON.stringify(data),
  })

  return assertSuccess(response, 'create ticket failed')
}

export async function listMyTickets(params: TicketListParams) {
  const searchParams = new URLSearchParams({
    current: String(params.current ?? 1),
    size: String(params.size ?? 20),
  })
  if (params.status) {
    searchParams.set('status', params.status)
  }

  const response = await requestJson<ApiResponse<TicketListResponse>>(
    `/api/v1/ticket/tickets/me?${searchParams.toString()}`
  )

  return assertSuccess(response, 'list my tickets failed')
}

export async function listTicketCategories() {
  const response = await requestJson<ApiResponse<TicketCategory[]>>('/api/v1/ticket/categories')

  return assertSuccess(response, 'list ticket categories failed') ?? []
}

export async function getMyTicket(id: number) {
  const response = await requestJson<ApiResponse<TicketItem>>(`/api/v1/ticket/tickets/${id}`)

  return assertSuccess(response, 'get my ticket failed')
}

export async function listMyTicketReplies(id: number) {
  const response = await requestJson<ApiResponse<TicketReply[]>>(
    `/api/v1/ticket/tickets/${id}/replies`
  )

  return assertSuccess(response, 'list my ticket replies failed') ?? []
}

export async function addMyTicketReply(id: number, data: AddReplyParams) {
  const response = await requestJson<ApiResponse<TicketReply>>(
    `/api/v1/ticket/tickets/${id}/replies`,
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'add my ticket reply failed')
}

export async function adminListTickets(params: AdminTicketListParams) {
  const searchParams = new URLSearchParams()
  if (params.current) searchParams.set('current', String(params.current))
  if (params.size) searchParams.set('size', String(params.size))
  if (params.status) searchParams.set('status', params.status)
  if (params.keyword) searchParams.set('keyword', params.keyword)
  if (params.category_id) searchParams.set('category_id', String(params.category_id))
  if (params.user_id) searchParams.set('user_id', String(params.user_id))

  const response = await requestJson<ApiResult<TicketListResponse>>(
    `/api/v1/system/ticket/tickets${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'admin list tickets failed')
}

export async function adminGetTicket(id: number) {
  const response = await requestJson<ApiResult<TicketItem>>(`/api/v1/system/ticket/tickets/${id}`)
  return assertSuccess(response, 'admin get ticket failed')
}

export async function adminUpdateTicketStatus(id: number, data: UpdateStatusParams) {
  const response = await requestJson<ApiResult<TicketItem>>(
    `/api/v1/system/ticket/tickets/${id}/status`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'admin update ticket status failed')
}

export async function adminUpdateTicketPriority(id: number, data: UpdatePriorityParams) {
  const response = await requestJson<ApiResult<TicketItem>>(
    `/api/v1/system/ticket/tickets/${id}/priority`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'admin update ticket priority failed')
}

export async function adminAddTicketReply(id: number, data: AdminAddReplyParams) {
  const response = await requestJson<ApiResult<TicketReply>>(
    `/api/v1/system/ticket/tickets/${id}/replies`,
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'admin add ticket reply failed')
}

export async function adminListTicketReplies(id: number) {
  const response = await requestJson<ApiResult<TicketReply[]>>(
    `/api/v1/system/ticket/tickets/${id}/replies`
  )
  return assertSuccess(response, 'admin list ticket replies failed') ?? []
}

export async function adminListTicketStatusHistory(id: number) {
  const response = await requestJson<ApiResult<TicketStatusHistory[]>>(
    `/api/v1/system/ticket/tickets/${id}/status-history`
  )
  return assertSuccess(response, 'admin list ticket status history failed') ?? []
}

export async function adminListTicketCategories() {
  const response = await requestJson<ApiResult<TicketCategory[]>>('/api/v1/system/ticket/categories')
  return assertSuccess(response, 'admin list ticket categories failed') ?? []
}

export async function adminCreateTicketCategory(data: UpsertCategoryParams) {
  const response = await requestJson<ApiResult<TicketCategory>>('/api/v1/system/ticket/categories', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'admin create ticket category failed')
}

export async function adminUpdateTicketCategory(id: number, data: UpsertCategoryParams) {
  const response = await requestJson<ApiResult<TicketCategory>>(
    `/api/v1/system/ticket/categories/${id}`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'admin update ticket category failed')
}

export async function adminDeleteTicketCategory(id: number) {
  const response = await requestJson<ApiResult<null>>(
    `/api/v1/system/ticket/categories/${id}`,
    { method: 'DELETE' }
  )
  return assertSuccess(response, 'admin delete ticket category failed')
}

export async function adminTicketStatistics() {
  const response = await requestJson<ApiResult<Statistics>>('/api/v1/system/ticket/statistics')
  return assertSuccess(response, 'admin ticket statistics failed')
}
