import { AppRouteRecord } from '@/types/router'

export const dashboardRoutes: AppRouteRecord = {
  name: 'Dashboard',
  path: '/dashboard',
  component: '/index/index',
  meta: {
    title: 'menus.dashboard.title',
    icon: 'ri:pie-chart-line',
    corpCapabilitiesAny: ['menu.dashboard']
  },
  children: [
    {
      path: 'console',
      name: 'Console',
      component: '/dashboard/console',
      meta: {
        title: 'menus.dashboard.console',
        keepAlive: false,
        fixedTab: true
      }
    },
    {
      path: 'npc-kills',
      name: 'CorpNpcKillReport',
      component: '/dashboard/npc-kills',
      meta: {
        title: 'menus.dashboard.npcKills',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAll: ['dashboard.npc_kills.corp', 'info.npc_kills.corp']
      }
    },
    {
      path: 'corporation-structures',
      name: 'DashboardCorporationStructures',
      component: '/dashboard/corporation-structures',
      meta: {
        title: 'menus.dashboard.corporationStructures',
        keepAlive: true,
        roles: ['super_admin', 'admin']
      }
    },
    {
      path: 'fuel-officer-structures',
      name: 'DashboardFuelOfficerStructures',
      component: '/dashboard/fuel-officer-structures',
      meta: {
        title: 'menus.dashboard.fuelOfficerStructures',
        keepAlive: true,
        roles: ['super_admin', 'fuel_officer']
      }
    },
    {
      path: 'galaxy-registry',
      name: 'DashboardGalaxyRegistry',
      component: '/dashboard/galaxy-registry',
      meta: {
        title: 'menus.dashboard.galaxyRegistry',
        keepAlive: true,
        roles: ['super_admin', 'admin', 'captain', 'user']
      }
    }
  ]
}
