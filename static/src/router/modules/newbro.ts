import { AppRouteRecord } from '@/types/router'

export const newbroRoutes: AppRouteRecord = {
  path: '/newbro',
  name: 'NewbroRoot',
  component: '/index/index',
  meta: {
    title: 'menus.newbro.title',
    icon: 'ri:user-heart-line',
    login: true
  },
  children: [
    {
      path: 'select-captain',
      name: 'NewbroSelectCaptain',
      component: '/newbro/select-captain',
      meta: {
        title: 'menus.newbro.selectCaptain',
        keepAlive: true,
        login: true,
        requiresNewbro: true
      }
    },
    {
      path: 'captain',
      name: 'NewbroCaptainDashboard',
      component: '/newbro/captain',
      meta: {
        title: 'menus.newbro.captain',
        keepAlive: true,
        roles: ['captain']
      }
    },
    {
      path: 'manage',
      name: 'NewbroManage',
      component: '/newbro/manage',
      meta: {
        title: 'menus.newbro.manage',
        keepAlive: true,
        roles: ['super_admin', 'admin']
      }
    }
  ]
}
