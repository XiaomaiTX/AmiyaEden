import { AppRouteRecord } from '@/types/router'

export const fuxiHallRoutes: AppRouteRecord = {
  path: '/fuxi-hall',
  name: 'FuxiHallRoot',
  component: '/index/index',
  meta: {
    title: 'menus.fuxiHall.title',
    icon: 'ri:group-line',
    login: true,
    corpCapabilities: ['menu.fuxi_hall']
  },
  children: [
    {
      path: 'leadership',
      name: 'FuxiHallLeadership',
      component: '/fuxi-hall/leadership',
      meta: {
        title: 'menus.fuxiHall.leadership',
        keepAlive: true,
        login: true
      }
    },
    {
      path: 'contributors',
      name: 'FuxiHallContributors',
      component: '/fuxi-hall/contributors',
      meta: {
        title: 'menus.fuxiHall.contributors',
        keepAlive: true,
        login: true
      }
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
