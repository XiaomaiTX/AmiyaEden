import { AppRouteRecord } from '@/types/router'

export const fuxiHallRoutes: AppRouteRecord = {
  path: '/fuxi-hall',
  name: 'FuxiHall',
  component: '/fuxi-hall/index',
  meta: {
    title: 'menus.fuxiHall.fame',
    icon: 'ri:group-line',
    keepAlive: true,
    login: true,
    corpCapabilitiesAny: ['menu.fuxi_hall']
  }
}
