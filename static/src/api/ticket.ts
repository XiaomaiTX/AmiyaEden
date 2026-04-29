import request from '@/utils/http'

export function createTicket(data: Api.Ticket.CreateTicketParams) {
  return request.post<Api.Ticket.TicketItem>({
    url: '/api/v1/ticket/tickets',
    data
  })
}

export function listMyTickets(params?: Api.Ticket.TicketListParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Ticket.TicketItem>>({
    url: '/api/v1/ticket/tickets/me',
    params
  })
}

export function getMyTicket(id: number) {
  return request.get<Api.Ticket.TicketItem>({
    url: `/api/v1/ticket/tickets/${id}`
  })
}

export function addMyTicketReply(id: number, data: Api.Ticket.AddReplyParams) {
  return request.post<Api.Ticket.TicketReply>({
    url: `/api/v1/ticket/tickets/${id}/replies`,
    data
  })
}

export function listMyTicketReplies(id: number) {
  return request.get<Api.Ticket.TicketReply[]>({
    url: `/api/v1/ticket/tickets/${id}/replies`
  })
}

export function listTicketCategories() {
  return request.get<Api.Ticket.TicketCategory[]>({
    url: '/api/v1/ticket/categories'
  })
}

export function adminListTickets(params?: Api.Ticket.AdminTicketListParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Ticket.TicketItem>>({
    url: '/api/v1/system/ticket/tickets',
    params
  })
}

export function adminGetTicket(id: number) {
  return request.get<Api.Ticket.TicketItem>({
    url: `/api/v1/system/ticket/tickets/${id}`
  })
}

export function adminUpdateTicketStatus(id: number, data: Api.Ticket.UpdateStatusParams) {
  return request.put<Api.Ticket.TicketItem>({
    url: `/api/v1/system/ticket/tickets/${id}/status`,
    data
  })
}

export function adminUpdateTicketPriority(id: number, data: Api.Ticket.UpdatePriorityParams) {
  return request.put<Api.Ticket.TicketItem>({
    url: `/api/v1/system/ticket/tickets/${id}/priority`,
    data
  })
}

export function adminAddTicketReply(id: number, data: Api.Ticket.AdminAddReplyParams) {
  return request.post<Api.Ticket.TicketReply>({
    url: `/api/v1/system/ticket/tickets/${id}/replies`,
    data
  })
}

export function adminListTicketReplies(id: number) {
  return request.get<Api.Ticket.TicketReply[]>({
    url: `/api/v1/system/ticket/tickets/${id}/replies`
  })
}

export function adminListTicketStatusHistory(id: number) {
  return request.get<Api.Ticket.TicketStatusHistory[]>({
    url: `/api/v1/system/ticket/tickets/${id}/status-history`
  })
}

export function adminListTicketCategories() {
  return request.get<Api.Ticket.TicketCategory[]>({
    url: '/api/v1/system/ticket/categories'
  })
}

export function adminCreateTicketCategory(data: Api.Ticket.UpsertCategoryParams) {
  return request.post<Api.Ticket.TicketCategory>({
    url: '/api/v1/system/ticket/categories',
    data
  })
}

export function adminUpdateTicketCategory(id: number, data: Api.Ticket.UpsertCategoryParams) {
  return request.put<Api.Ticket.TicketCategory>({
    url: `/api/v1/system/ticket/categories/${id}`,
    data
  })
}

export function adminDeleteTicketCategory(id: number) {
  return request.del({
    url: `/api/v1/system/ticket/categories/${id}`
  })
}

export function adminTicketStatistics() {
  return request.get<Api.Ticket.Statistics>({
    url: '/api/v1/system/ticket/statistics'
  })
}
