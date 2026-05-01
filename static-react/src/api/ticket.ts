import { requestJson } from '@/api/http-client'
import type {
  AddReplyParams,
  CreateTicketParams,
  TicketCategory,
  TicketItem,
  TicketListParams,
  TicketListResponse,
  TicketReply,
} from '@/types/api/ticket'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || fallbackMessage)
  }
  return response.data
}

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
