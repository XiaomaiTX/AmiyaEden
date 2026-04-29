import { AppRouteRecord } from '@/types/router'

export const ticketRoutes: AppRouteRecord = {
  path: '/ticket',
  name: 'TicketRoot',
  component: '/index/index',
  meta: {
    title: 'menus.ticket.title',
    icon: 'ri:question-answer-line',
    login: true
  },
  children: [
    {
      path: 'my-tickets',
      name: 'TicketMyList',
      component: '/ticket/my-tickets',
      meta: {
        title: 'menus.ticket.myTickets',
        keepAlive: true,
        login: true
      }
    },
    {
      path: 'create',
      name: 'TicketCreate',
      component: '/ticket/create',
      meta: {
        title: 'menus.ticket.create',
        keepAlive: true,
        login: true
      }
    },
    {
      path: 'detail/:id',
      name: 'TicketDetail',
      component: '/ticket/detail',
      meta: {
        title: 'menus.ticket.detail',
        isHide: true,
        isHideTab: true,
        login: true
      }
    }
  ]
}

