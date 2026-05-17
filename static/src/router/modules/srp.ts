import { AppRouteRecord } from '@/types/router'

export const srpRoutes: AppRouteRecord = {
  path: '/srp',
  name: 'SRP',
  component: '/index/index',
  meta: {
    title: 'menus.srp.title',
    icon: 'ri:shield-user-line',
    login: true,
    corpCapabilities: ['menu.srp']
  },
  children: [
    {
      path: 'srp-apply',
      name: 'SrpApply',
      component: '/srp/apply',
      meta: {
        title: 'menus.srp.srpApply',
        keepAlive: true,
        login: true,
        corpCapabilities: ['srp.user']
      }
    },
    {
      path: 'srp-manage',
      name: 'SrpManage',
      component: '/srp/manage',
      meta: {
        title: 'menus.srp.srpManage',
        keepAlive: true,
        roles: ['super_admin', 'admin', 'senior_fc', 'srp'],
        corpCapabilities: ['srp.manage'],
        authList: [{ title: 'authActions.srp.approve', authMark: 'approve' }]
      }
    },
    {
      path: 'srp-prices',
      name: 'SrpPrices',
      component: '/srp/prices',
      meta: {
        title: 'menus.srp.srpPrices',
        keepAlive: true,
        roles: ['super_admin', 'admin', 'senior_fc', 'srp'],
        corpCapabilities: ['srp.manage']
      }
    }
  ]
}
