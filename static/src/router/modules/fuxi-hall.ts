import { AppRouteRecord } from '@/types/router'

export const fuxiHallRoutes: AppRouteRecord = {
  path: '/fuxi-hall',
  name: 'FuxiHallRoot',
  component: '/index/index',
  meta: {
    title: 'menus.fuxiHall.title',
    icon: 'ri:group-line',
    login: true,
    corpCapabilitiesAny: ['menu.fuxi_hall']
  },
  children: [
    {
      path: 'hall-of-fame',
      name: 'FuxiHallFame',
      component: '/fuxi-hall/hall-of-fame',
      meta: { title: 'menus.fuxiHall.fame', keepAlive: true, login: true }
    },
    {
      path: 'manage',
      name: 'FuxiHallManage',
      component: '/fuxi-hall/manage',
      meta: {
        title: 'menus.fuxiHall.manage',
        keepAlive: false,
        roles: ['super_admin', 'admin']
      }
    }
  ]
}
