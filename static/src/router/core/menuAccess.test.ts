import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import type { AppRouteRecord } from '../../types/router'
import { dashboardRoutes } from '../modules/dashboard'
import { newbroRoutes as actualNewbroRoutes } from '../modules/newbro'
import { skillPlanningRoutes } from '../modules/skill-planning'
import { shopRoutes } from '../modules/shop'
import { srpRoutes } from '../modules/srp'
import { systemRoutes } from '../modules/system'
import { ticketRoutes } from '../modules/ticket'
import { applyMenuAccessFilter, pruneEmptyMenus } from './menuAccess'

const newbroRoutes: AppRouteRecord[] = [
  {
    path: '/newbro',
    name: 'NewbroRoot',
    component: '/index/index',
    meta: {
      title: 'menus.newbro.title',
      login: true
    },
    children: [
      {
        path: 'select-captain',
        name: 'NewbroSelectCaptain',
        component: '/newbro/select-captain',
        meta: {
          title: 'menus.newbro.selectCaptain',
          login: true,
          requiresNewbro: true
        }
      },
      {
        path: 'select-mentor',
        name: 'NewbroSelectMentor',
        component: '/newbro/select-mentor',
        meta: {
          title: 'menus.newbro.selectMentor',
          login: true,
          requiresMentorMenteeEligibility: true
        }
      }
    ]
  }
]

const zhMessages = JSON.parse(
  readFileSync(new URL('../../locales/langs/zh.json', import.meta.url), 'utf8')
) as Record<string, unknown>

const enMessages = JSON.parse(
  readFileSync(new URL('../../locales/langs/en.json', import.meta.url), 'utf8')
) as Record<string, unknown>

function getLocaleValue(messages: Record<string, unknown>, key: string) {
  return key.split('.').reduce<unknown>((value, part) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[part]
  }, messages)
}

function collectAuthListTitles(routes: AppRouteRecord[], parentPath = '') {
  const titles: Array<{ path: string; title: string }> = []

  for (const route of routes) {
    const path = `${parentPath}/${route.path}`.replace(/\/+/g, '/')
    for (const action of route.meta?.authList ?? []) {
      titles.push({ path, title: action.title })
    }
    if (route.children?.length) {
      titles.push(...collectAuthListTitles(route.children, path))
    }
  }

  return titles
}

test('router permission action titles use locale keys', () => {
  const titles = collectAuthListTitles([shopRoutes, srpRoutes, systemRoutes])

  assert.deepEqual(
    titles.filter(({ title }) => /[\u4e00-\u9fff]/u.test(title)),
    []
  )
  assert.deepEqual(
    titles.filter(({ title }) => !title.startsWith('authActions.')),
    []
  )
  assert.deepEqual(
    titles.filter(
      ({ title }) =>
        typeof getLocaleValue(zhMessages, title) !== 'string' ||
        typeof getLocaleValue(enMessages, title) !== 'string'
    ),
    []
  )
})

test('applyMenuAccessFilter hides requiresNewbro routes when status is unknown', () => {
  const filtered = applyMenuAccessFilter(newbroRoutes, ['user'], [], undefined)

  assert.deepEqual(filtered, [
    {
      path: '/newbro',
      name: 'NewbroRoot',
      component: '/index/index',
      meta: {
        title: 'menus.newbro.title',
        login: true
      },
      children: []
    }
  ])
})

test('pruneEmptyMenus removes directories whose children were fully filtered out', () => {
  const filtered = applyMenuAccessFilter(newbroRoutes, ['user'], [], undefined)
  const pruned = pruneEmptyMenus(filtered)

  assert.deepEqual(pruned, [])
})

test('applyMenuAccessFilter keeps SkillPlans for logged-in ordinary users', () => {
  const filtered = applyMenuAccessFilter(
    [skillPlanningRoutes],
    ['user'],
    ['menu.skill_planning'],
    undefined
  )
  const skillPlanning = filtered[0]

  assert.equal(
    skillPlanning.children?.some((route) => route.name === 'SkillPlans'),
    true
  )
})

test('applyMenuAccessFilter hides mentor selection routes when mentor eligibility is unknown', () => {
  const filtered = applyMenuAccessFilter(newbroRoutes, ['user'], [], true, undefined)

  assert.equal(
    filtered[0].children?.some((route) => route.name === 'NewbroSelectMentor'),
    false
  )
})

test('newbro mentor selection route requires mentor mentee eligibility', () => {
  const mentorRoute = actualNewbroRoutes.children?.find(
    (route) => route.name === 'NewbroSelectMentor'
  )

  assert.equal(mentorRoute?.meta.requiresMentorMenteeEligibility, true)
})

test('newbro manage route keeps readonly menu access for captains', () => {
  const manageRoute = actualNewbroRoutes.children?.find((route) => route.name === 'NewbroManage')

  assert.deepEqual(manageRoute?.meta.roles, ['super_admin', 'admin', 'captain'])
})

test('newbro recruit link route is the last child under Newbro', () => {
  const childNames = actualNewbroRoutes.children?.map((route) => route.name)

  assert.deepEqual(childNames?.at(-1), 'NewbroRecruitLink')
})

test('applyMenuAccessFilter hides AutoRole from admins but keeps it for super admins', () => {
  const adminFiltered = applyMenuAccessFilter(
    [systemRoutes],
    ['admin'],
    ['menu.system', 'system.manage']
  )
  const adminSystemMenu = adminFiltered[0]

  assert.equal(
    adminSystemMenu.children?.some((route) => route.name === 'AutoRole'),
    false
  )

  const superAdminFiltered = applyMenuAccessFilter([systemRoutes], ['super_admin'])
  const superAdminSystemMenu = superAdminFiltered[0]

  assert.equal(
    superAdminSystemMenu.children?.some((route) => route.name === 'AutoRole'),
    true
  )
})

test('applyMenuAccessFilter enforces corp capability gates on SRP routes', () => {
  const withoutCaps = applyMenuAccessFilter([srpRoutes], ['admin'], [])
  const withCaps = applyMenuAccessFilter([srpRoutes], ['admin'], ['menu.srp', 'srp.manage'])

  assert.equal(withoutCaps.length, 0)
  assert.equal(withCaps.length, 1)
  assert.equal(
    withCaps[0].children?.some((route) => route.name === 'SrpManage'),
    true
  )
})

test('applyMenuAccessFilter hides BasicConfig from admins but keeps it for super admins', () => {
  const adminFiltered = applyMenuAccessFilter(
    [systemRoutes],
    ['admin'],
    ['menu.system', 'system.manage']
  )
  const adminSystemMenu = adminFiltered[0]

  assert.equal(
    adminSystemMenu.children?.some((route) => route.name === 'BasicConfig'),
    false
  )

  const superAdminFiltered = applyMenuAccessFilter([systemRoutes], ['super_admin'])
  const superAdminSystemMenu = superAdminFiltered[0]

  assert.equal(
    superAdminSystemMenu.children?.some((route) => route.name === 'BasicConfig'),
    true
  )
})

test('CorpNpcKillReport lives under Dashboard for admins only', () => {
  const adminDashboard = applyMenuAccessFilter(
    [dashboardRoutes],
    ['admin'],
    ['menu.dashboard', 'dashboard.npc_kills.corp', 'info.npc_kills.corp']
  )[0]
  const userDashboard = applyMenuAccessFilter(
    [dashboardRoutes],
    ['user'],
    ['menu.dashboard', 'dashboard.npc_kills.corp', 'info.npc_kills.corp']
  )[0]
  const adminSystem = applyMenuAccessFilter(
    [systemRoutes],
    ['admin'],
    ['menu.system', 'system.manage']
  )[0]

  const adminNpcKillsRoute = adminDashboard.children?.find(
    (route) => route.name === 'CorpNpcKillReport'
  )

  assert.equal(adminNpcKillsRoute?.path, 'npc-kills')
  assert.deepEqual(adminNpcKillsRoute?.meta.roles, ['super_admin', 'admin'])
  assert.equal(
    userDashboard.children?.some((route) => route.name === 'CorpNpcKillReport'),
    false
  )
  assert.equal(
    adminSystem.children?.some((route) => route.name === 'CorpNpcKillReport'),
    false
  )
})

test('DashboardCorporationStructures lives under Dashboard for admins only', () => {
  const adminDashboard = applyMenuAccessFilter(
    [dashboardRoutes],
    ['admin'],
    ['menu.dashboard', 'dashboard.corp_structures.read']
  )[0]
  const superAdminDashboard = applyMenuAccessFilter([dashboardRoutes], ['super_admin'])[0]
  const userDashboard = applyMenuAccessFilter(
    [dashboardRoutes],
    ['user'],
    ['menu.dashboard', 'dashboard.corp_structures.read']
  )[0]

  const adminRoute = adminDashboard.children?.find(
    (route) => route.name === 'DashboardCorporationStructures'
  )

  assert.equal(adminRoute?.path, 'corporation-structures')
  assert.deepEqual(adminRoute?.meta.roles, ['super_admin', 'admin'])
  assert.equal(
    superAdminDashboard.children?.some((route) => route.name === 'DashboardCorporationStructures'),
    true
  )
  assert.equal(
    userDashboard.children?.some((route) => route.name === 'DashboardCorporationStructures'),
    false
  )
})

test('system menu no longer includes ticket admin pages', () => {
  const adminSystem = applyMenuAccessFilter(
    [systemRoutes],
    ['admin'],
    ['menu.system', 'system.manage']
  )[0]

  assert.equal(
    adminSystem.children?.some((route) => route.name === 'TicketManagement'),
    false
  )
  assert.equal(
    adminSystem.children?.some((route) => route.name === 'TicketCategories'),
    false
  )
  assert.equal(
    adminSystem.children?.some((route) => route.name === 'TicketStatistics'),
    false
  )
})

test('ticket center includes admin pages for admin roles', () => {
  const adminTicket = applyMenuAccessFilter(
    [ticketRoutes],
    ['admin'],
    ['menu.ticket', 'ticket.manage']
  )[0]

  assert.equal(
    adminTicket.children?.some((route) => route.name === 'TicketManagement'),
    true
  )
  assert.equal(
    adminTicket.children?.some((route) => route.name === 'TicketCategories'),
    true
  )
  assert.equal(
    adminTicket.children?.some((route) => route.name === 'TicketStatistics'),
    true
  )
})

test('applyMenuAccessFilter keeps SRP prices for SRP, admin, senior fc, and super admins', () => {
  const requiredCaps = ['menu.srp', 'srp.manage']
  const adminSrpMenu = applyMenuAccessFilter([srpRoutes], ['admin'], requiredCaps)[0]
  const seniorFCSrpMenu = applyMenuAccessFilter([srpRoutes], ['senior_fc'], requiredCaps)[0]
  const srpOfficerMenu = applyMenuAccessFilter([srpRoutes], ['srp'], requiredCaps)[0]
  const superAdminSrpMenu = applyMenuAccessFilter([srpRoutes], ['super_admin'], requiredCaps)[0]

  assert.equal(
    adminSrpMenu.children?.some((route) => route.name === 'SrpPrices'),
    true
  )
  assert.equal(
    seniorFCSrpMenu.children?.some((route) => route.name === 'SrpPrices'),
    true
  )
  assert.equal(
    srpOfficerMenu.children?.some((route) => route.name === 'SrpPrices'),
    true
  )
  assert.equal(
    superAdminSrpMenu.children?.some((route) => route.name === 'SrpPrices'),
    true
  )
})

test('applyMenuAccessFilter hides ShopOrderManage from welfare officers', () => {
  const welfareShopMenu = applyMenuAccessFilter([shopRoutes], ['welfare'], ['menu.shop'])[0]

  assert.equal(
    welfareShopMenu.children?.some((route) => route.name === 'ShopOrderManage'),
    false
  )
})

test('applyMenuAccessFilter keeps ShopOrderManage for shop order officers', () => {
  const shopOrderShopMenu = applyMenuAccessFilter(
    [shopRoutes],
    ['shop_order_manage'],
    ['menu.shop', 'shop.manage']
  )[0]

  assert.equal(
    shopOrderShopMenu.children?.some((route) => route.name === 'ShopOrderManage'),
    true
  )
})

test('applyMenuAccessFilter hides non-SRP domains when only SRP capabilities are granted', () => {
  const filtered = applyMenuAccessFilter(
    [
      dashboardRoutes,
      skillPlanningRoutes,
      shopRoutes,
      systemRoutes,
      ticketRoutes,
      srpRoutes,
      actualNewbroRoutes
    ],
    ['admin'],
    ['menu.srp', 'srp.user']
  )

  assert.deepEqual(
    filtered.map((route) => route.name),
    ['SRP']
  )
})
