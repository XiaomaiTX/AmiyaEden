import { AppRouteRecord } from '@/types/router'

export const charactersRoutes: AppRouteRecord = {
  name: 'Characters',
  path: '/characters',
  component: '/dashboard/characters',
  meta: {
    title: 'menus.characters.title',
    icon: 'ri:user-3-line',
    keepAlive: true
  }
}
