import type { AppRouteRecord } from '@/types/router'
export const announcementRoutes: AppRouteRecord = {
  path: '/announcements',
  name: 'Announcements',
  component: '/announcements/index',
  meta: { title: 'menus.announcements.title', icon: 'ri:notification-3-line', login: true }
}
