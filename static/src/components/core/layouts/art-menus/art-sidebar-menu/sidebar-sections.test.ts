import assert from 'node:assert/strict'
import test from 'node:test'
import { routeModules } from '@/router/modules'
import { applyMenuAccessFilter, pruneEmptyMenus } from '@/router/core/menuAccess'
import { admissionMenuVisibility } from '@/store/modules/admissions.helpers'
import { applyBadgeCountsToMenu } from '@/store/modules/badge.helpers'
import en from '@/locales/langs/en.json'
import zh from '@/locales/langs/zh.json'
import type { AppRouteRecord } from '@/types/router'
import { buildSidebarSections, findSidebarSelection } from './sidebar-sections'

function menuFor(roles: string[], capabilities: string[] = []) {
  function normalize(route: AppRouteRecord, parentPath = ''): AppRouteRecord {
    const path = route.path.startsWith('/') ? route.path : `${parentPath}/${route.path}`
    return {
      ...route,
      path,
      meta: { ...route.meta },
      ...(route.children ? { children: route.children.map((child) => normalize(child, path)) } : {})
    }
  }

  return pruneEmptyMenus(applyMenuAccessFilter(routeModules, roles, capabilities, true, true)).map(
    (route) => normalize(route)
  )
}

function visiblePages(routes: AppRouteRecord[]): AppRouteRecord[] {
  return routes.flatMap((route) =>
    route.meta.isHide ? [] : route.children ? visiblePages(route.children) : [route]
  )
}

test('left navigation projects the complete visible menu into ordered sections without changing route records', () => {
  const menu = menuFor(['super_admin', 'captain'])
  const snapshot = structuredClone(menu)
  const sections = buildSidebarSections(menu)
  const childrenOf = (path: string) =>
    sections.flatMap((section) => section.items).find((item) => item.path === path)?.children

  assert.deepEqual(
    sections.map(({ title, items }) => [title, items.map((item) => item.path)]),
    [
      ['menus.sidebar.home', ['/dashboard/console', '/announcements']],
      ['menus.sidebar.fleetReadiness', ['sidebar:operation', '/srp', 'sidebar:skill-planning']],
      ['menus.sidebar.characterTools', ['sidebar:info', 'sidebar:tools']],
      [
        'menus.sidebar.memberServices',
        ['/admissions', 'sidebar:newbro', '/welfare', 'sidebar:shop', '/ticket', '/fuxi-hall']
      ],
      [
        'menus.sidebar.corporationOperations',
        ['sidebar:corporation-data', 'sidebar:territory-facilities', 'sidebar:finance']
      ],
      [
        'menus.sidebar.systemManagement',
        [
          '/system/user',
          '/system/blacklist',
          '/system/auto-role',
          '/system/task-manager',
          '/system/audit',
          '/system/webhook',
          '/system/basic-config'
        ]
      ]
    ]
  )
  assert.deepEqual(
    childrenOf('sidebar:operation')?.map(({ path }) => path),
    [
      '/operation/pap',
      '/operation/corporation-pap',
      '/operation/fleets',
      '/operation/fleet-configs'
    ]
  )
  assert.deepEqual(
    childrenOf('sidebar:skill-planning')?.map(({ path }) => path),
    [
      '/skill-planning/personal-skill-plans',
      '/skill-planning/skill-plans',
      '/skill-planning/completion-check'
    ]
  )
  assert.deepEqual(
    childrenOf('sidebar:tools')?.map(({ path }) => path),
    ['/mumble', '/info/esi-check', '/info/tool-bookmarks']
  )
  assert.deepEqual(
    childrenOf('sidebar:newbro')?.map(({ path }) => path),
    [
      '/newbro/select-mentor',
      '/newbro/captain',
      '/newbro/mentor',
      '/newbro/recruit-link',
      '/newbro/manage',
      '/newbro/mentor-manage',
      '/system/newbro-settings'
    ]
  )
  assert.deepEqual(
    childrenOf('sidebar:shop')?.map(({ path }) => path),
    ['/shop/browse', '/shop/wallet', '/shop/manage', '/shop/order-manage']
  )
  assert.deepEqual(
    childrenOf('sidebar:finance')?.map(({ path }) => path),
    ['/system/salary', '/system/wallet', '/system/pap', '/system/pap-exchange']
  )
  assert.deepEqual(
    visiblePages(sections.flatMap((section) => section.items))
      .map(({ path }) => path)
      .sort(),
    visiblePages(menu)
      .map(({ path }) => path)
      .sort()
  )
  assert.deepEqual(menu, snapshot)
})

test('only accessible, non-hidden entries create visible groups and sections', () => {
  const guest = buildSidebarSections(admissionMenuVisibility(menuFor(['guest']), null))
  assert.deepEqual(
    guest.map(({ title }) => title),
    ['menus.sidebar.characterTools', 'menus.sidebar.memberServices']
  )
  assert.deepEqual(
    visiblePages(guest.flatMap(({ items }) => items)).map(({ path }) => path),
    ['/mumble', '/admissions/history']
  )
  assert.deepEqual(buildSidebarSections([]), [])
  assert.deepEqual(
    buildSidebarSections([
      { path: '/welfare', meta: { title: 'menus.welfare.title' }, children: [] }
    ]),
    []
  )

  const fuelOfficer = buildSidebarSections(menuFor(['fuel_officer'], ['menu.dashboard']))
  assert.deepEqual(findSidebarSelection(fuelOfficer, '/dashboard/fuel-officer-structures'), {
    path: '/dashboard/fuel-officer-structures',
    groups: ['sidebar:territory-facilities']
  })
  assert.equal(findSidebarSelection(fuelOfficer, '/dashboard/corporation-structures'), undefined)

  const settingsOnly = buildSidebarSections(menuFor(['admin'], ['system.manage']))
  assert.deepEqual(findSidebarSelection(settingsOnly, '/system/newbro-settings'), {
    path: '/system/newbro-settings',
    groups: ['sidebar:newbro']
  })
  assert.equal(findSidebarSelection(settingsOnly, '/newbro/mentor-manage'), undefined)
})

test('selection opens the displayed parent instead of the route parent, with direct entries at level zero', () => {
  const sections = buildSidebarSections(menuFor(['super_admin']))
  assert.deepEqual(findSidebarSelection(sections, '/dashboard/console'), {
    path: '/dashboard/console',
    groups: []
  })
  assert.deepEqual(findSidebarSelection(sections, '/info/esi-check'), {
    path: '/info/esi-check',
    groups: ['sidebar:tools']
  })
  assert.deepEqual(findSidebarSelection(sections, '/system/wallet'), {
    path: '/system/wallet',
    groups: ['sidebar:finance']
  })
  assert.deepEqual(findSidebarSelection(sections, '/system/user'), {
    path: '/system/user',
    groups: []
  })
  assert.equal(findSidebarSelection(sections, '/characters'), undefined)
  assert.equal(findSidebarSelection(sections, '/dashboard'), undefined)
})

test('existing leaf badges move to their new group and hidden pages never contribute', () => {
  const menu = admissionMenuVisibility(menuFor(['super_admin']), null)
  const badgeCounts = {
    announcement_unread: 2,
    admission_pending: 9,
    corporation_structures_attention: 4,
    welfare_eligible: 2,
    welfare_pending: 3,
    order_pending: 5,
    mentor_pending_applications: 1
  }
  applyBadgeCountsToMenu(menu, badgeCounts)
  const projectedBadges = () =>
    Object.fromEntries(
      buildSidebarSections(menu)
        .flatMap(({ items }) => items.flatMap((item) => [item, ...(item.children ?? [])]))
        .filter(({ meta }) => meta.showTextBadge)
        .map(({ path, meta }) => [path, meta.showTextBadge])
    )

  assert.deepEqual(projectedBadges(), {
    '/announcements': '2',
    'sidebar:newbro': '1',
    '/newbro/mentor': '1',
    '/welfare': '5',
    '/welfare/my': '2',
    '/welfare/approval': '3',
    'sidebar:shop': '5',
    '/shop/order-manage': '5',
    'sidebar:territory-facilities': '4',
    '/dashboard/corporation-structures': '4'
  })
  assert.equal(menu.find(({ path }) => path === '/admissions')?.meta.showTextBadge, '9')
  assert.equal(menu.find(({ path }) => path === '/dashboard')?.meta.showTextBadge, '4')
  applyBadgeCountsToMenu(menu, {})
  assert.deepEqual(projectedBadges(), {})
})

test('wallet titles distinguish currencies in both locales and only override the sidebar', () => {
  const menu = menuFor(['super_admin'])
  const pages = visiblePages(buildSidebarSections(menu).flatMap(({ items }) => items))
  const isk = pages.find(({ path }) => path === '/info/wallet')?.meta.title
  const fuxi = pages.find(({ path }) => path === '/shop/wallet')?.meta.title
  assert.equal(isk, 'menus.sidebar.iskWallet')
  assert.equal(fuxi, 'menus.sidebar.fuxiWallet')
  assert.equal(
    visiblePages(menu).find(({ path }) => path === '/shop/wallet')?.meta.title,
    'menus.shop.wallet'
  )
  assert.match(en.menus.sidebar.iskWallet, /ISK/u)
  assert.match(en.menus.sidebar.fuxiWallet, /Fuxi Coin/u)
  assert.match(zh.menus.sidebar.iskWallet, /ISK/u)
  assert.match(zh.menus.sidebar.fuxiWallet, /伏羲币/u)
})
