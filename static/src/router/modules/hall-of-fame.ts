import { AppRouteRecord } from '@/types/router'

export const hallOfFameRoutes: AppRouteRecord = {
  path: '/hall-of-fame',
  name: 'HallOfFameRoot',
  component: '/index/index',
  meta: {
    title: 'menus.hallOfFame.title',
    icon: 'ri:trophy-line'
  },
  children: [
    {
      path: 'temple',
      name: 'HallOfFameTemple',
      component: '/hall-of-fame/temple',
      meta: {
        title: 'menus.hallOfFame.temple',
        keepAlive: true,
        login: true
      }
    },
    {
      path: 'manage',
      name: 'HallOfFameManage',
      component: '/hall-of-fame/manage',
      meta: {
        title: 'menus.hallOfFame.manage',
        keepAlive: false,
        roles: ['super_admin', 'admin']
      }
    },
    {
      path: 'current-manage',
      name: 'HallOfFameCurrentManage',
      component: '/hall-of-fame/current-manage',
      meta: {
        title: 'menus.hallOfFame.currentManage',
        keepAlive: true
      }
    }
  ]
}
