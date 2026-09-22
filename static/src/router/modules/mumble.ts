import { AppRouteRecord } from '@/types/router'

export const mumbleRoutes: AppRouteRecord = {
  name: 'MumbleCredentials',
  path: '/mumble',
  component: '/dashboard/mumble-credentials',
  meta: {
    title: 'menus.mumble.title',
    icon: 'ri:mic-line',
    keepAlive: true
  }
}
