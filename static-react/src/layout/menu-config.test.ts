import { appRouteSpecs } from '@/app/migration-routes'
import { buildShellMenuSections, type ShellMenuEntry } from '@/layout/menu-config'
import type { AdmissionContext } from '@/api/generated/client'

function buildMenuEntries(...args: Parameters<typeof buildShellMenuSections>) {
  return buildShellMenuSections(...args).flatMap((section) => section.entries)
}

function menuItems(entries: ShellMenuEntry[]) {
  return entries.flatMap((entry) => (entry.kind === 'group' ? entry.items : [entry]))
}

function findGroup(entries: ShellMenuEntry[], labelKey: string) {
  return entries
    .filter((entry) => entry.kind === 'group')
    .find((entry) => entry.labelKey === labelKey)
}

test('admission pending count appears only on visible review navigation and its parent', () => {
  const session = {
    isLoggedIn: true,
    roles: ['user'],
    corpCapabilities: [],
    isCurrentlyNewbro: false,
    isMentorMenteeEligible: false,
  }
  const context: AdmissionContext = {
    is_member: true,
    can_create_new_member: false,
    can_create_alt: true,
    qq: '',
    discord_id: '',
    characters: [],
    targets: [],
    reviewable_corporation_ids: [98824507],
  }
  const entries = buildMenuEntries(session, { admission_pending: 2 }, context)
  const admissions = findGroup(entries, 'admissions.title')
  expect(admissions?.badge).toBe(2)
  expect(admissions?.items.find((item) => item.to === '/admissions/review')?.badge).toBe(2)
  expect(admissions?.items.find((item) => item.to === '/admissions/history')?.badge).toBeUndefined()
  const noAuthority = buildMenuEntries(
    session,
    { admission_pending: 2 },
    { ...context, reviewable_corporation_ids: [] }
  )
  expect(findGroup(noAuthority, 'admissions.title')?.badge).toBeUndefined()
})

describe('menu config', () => {
  test('lists each visible route once, in the intended sections', () => {
    const sections = buildShellMenuSections(
      {
        isLoggedIn: true,
        roles: ['super_admin', 'captain'],
        corpCapabilities: [],
        isCurrentlyNewbro: true,
        isMentorMenteeEligible: true,
      },
      { corporation_structures_attention: 5 },
      {
        is_member: true,
        can_create_new_member: true,
        can_create_alt: true,
        qq: '',
        discord_id: '',
        characters: [],
        targets: [],
        reviewable_corporation_ids: [98824507],
      }
    )
    expect(sections.map((section) => section.labelKey)).toEqual([
      'nav.home',
      'nav.section.fleet',
      'nav.section.charactersAndTools',
      'nav.section.memberServices',
      'nav.section.corpOperations',
      'nav.group.system',
    ])
    const entries = sections.flatMap((section) => section.entries)
    expect(
      menuItems(entries)
        .map((item) => item.to)
        .sort()
    ).toEqual(
      appRouteSpecs
        .filter((route) => !route.menuHidden)
        .map((route) => `/${route.path}`)
        .sort()
    )
    expect(findGroup(entries, 'nav.group.operation')?.items.map((item) => item.to)).toEqual([
      '/operation/pap',
      '/operation/corporation-pap',
      '/operation/fleets',
      '/operation/fleet-configs',
    ])
    expect(findGroup(entries, 'nav.group.newbro')?.items.at(-1)?.to).toBe('/system/newbro-settings')
    expect(findGroup(entries, 'nav.group.territoryFacilities')?.badge).toBe(5)
  })

  test('shows the QQ Official console only to super administrators', () => {
    const session = {
      isLoggedIn: true,
      roles: ['admin'],
      corpCapabilities: ['system.manage'],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    }
    expect(menuItems(buildMenuEntries(session)).some(({ to }) => to === '/system/qq-official')).toBe(false)
    expect(buildMenuEntries({ ...session, roles: ['super_admin'] })).toContainEqual(
      expect.objectContaining({ kind: 'link', to: '/system/qq-official', labelKey: 'nav.system.qqOfficial' })
    )
  })

  test('drops sections and groups with no visible pages', () => {
    const sections = buildShellMenuSections({
      isLoggedIn: true,
      roles: ['guest'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })
    expect(
      sections.map((section) => [section.labelKey, section.entries.map((entry) => entry.labelKey)])
    ).toEqual([
      ['nav.section.charactersAndTools', ['nav.group.tools']],
      ['nav.section.memberServices', ['admissions.title']],
    ])
  })
  test.each(['user', 'admin', 'super_admin'])(
    'shows Fuxi Hall as one direct entry for %s',
    (role) => {
      const entries = buildMenuEntries({
        isLoggedIn: true,
        roles: [role],
        corpCapabilities: role === 'super_admin' ? [] : ['menu.fuxi_hall'],
        isCurrentlyNewbro: false,
        isMentorMenteeEligible: false,
      })
      const hall = entries.find((entry) => entry.labelKey === 'nav.fuxiHall.fame')
      expect(hall).toMatchObject({ kind: 'link', to: '/fuxi-hall' })
      expect(menuItems(entries).some((item) => item.to.startsWith('/fuxi-hall/'))).toBe(false)
    }
  )

  test('hides Fuxi Hall when the menu capability is missing', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })
    expect(menuItems(entries).some((item) => item.to === '/fuxi-hall')).toBe(false)
  })

  test('hides protected routes for anonymous visitor', () => {
    const entries = buildMenuEntries({
      isLoggedIn: false,
      roles: [],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(entries.length).toBe(0)
  })

  test('shows console, character information and utilities for a logged-in member', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['member'],
      corpCapabilities: ['menu.dashboard', 'menu.info'],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(entries).toContainEqual(
      expect.objectContaining({ kind: 'link', to: '/dashboard/console' })
    )
    expect(menuItems(entries).map((item) => item.to)).not.toContain('/characters')
    const tools = findGroup(entries, 'nav.group.tools')
    expect(tools?.items.map((item) => item.to)).toEqual([
      '/mumble',
      '/info/esi-check',
      '/info/tool-bookmarks',
    ])
    expect(findGroup(entries, 'nav.group.info')).toBeDefined()
    expect(menuItems(entries).some((item) => item.to.startsWith('/system/'))).toBe(false)
  })

  test('does not expose hidden detail routes in menu', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const allItems = menuItems(entries)
    expect(allItems.some((item) => item.to.includes('/ticket/detail/'))).toBe(false)
    expect(allItems.some((item) => item.to.includes('/operation/fleet-detail/'))).toBe(false)
  })

  test('hides system routes when required corp capability is missing', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    const paths = menuItems(entries).map((item) => item.to)
    expect(paths).not.toContain('/system/user')
    expect(paths).not.toContain('/system/task-manager')
    expect(paths).not.toContain('/system/wallet')
    expect(paths).not.toContain('/system/audit')
  })

  test('shows system routes when required corp capability is present', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['admin'],
      corpCapabilities: [
        'system.manage',
        'system.task.read',
        'system.wallet.read',
        'system.audit.read',
        'system.basic_config.read',
      ],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(entries.filter((entry) => entry.kind === 'link').map((entry) => entry.to)).toEqual(
      expect.arrayContaining(['/system/user', '/system/task-manager', '/system/audit'])
    )
    expect(findGroup(entries, 'nav.group.finance')?.items.map((item) => item.to)).toContain(
      '/system/wallet'
    )
  })

  test('shows representative capability-gated routes across every migrated domain', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['admin', 'welfare', 'srp', 'captain'],
      corpCapabilities: [
        'menu.dashboard',
        'menu.info',
        'info.wallet.read',
        'menu.operation',
        'menu.skill_planning',
        'welfare.approval',
        'menu.newbro',
        'menu.shop',
        'shop.manage',
        'shop.admin.product.manage',
        'srp.manage',
        'menu.ticket',
        'ticket.manage',
        'ticket.admin.read',
        'system.manage',
      ],
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const items = menuItems(entries).map((item) => item.to)

    expect(items).toEqual(
      expect.arrayContaining([
        '/dashboard/console',
        '/info/wallet',
        '/operation/fleet-configs',
        '/skill-planning/skill-plans',
        '/welfare/approval',
        '/newbro/captain',
        '/shop/manage',
        '/srp/srp-manage',
        '/ticket/management',
        '/system/user',
      ])
    )
  })

  test('super_admin bypasses corp capability gates for menu visibility', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(entries).toContainEqual(expect.objectContaining({ kind: 'link', to: '/system/user' }))
  })

  test('requires real captain role for refund menu and preserves mentor access', () => {
    const entries = buildMenuEntries({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const newbro = findGroup(entries, 'nav.group.newbro')
    const items = newbro?.items.map((item) => item.to) ?? []

    expect(items).not.toContain('/newbro/captain')
    expect(items).toContain('/newbro/mentor')
  })

  test('maps leaf badge counts and aggregates them into the parent menu', () => {
    const entries = buildMenuEntries(
      {
        isLoggedIn: true,
        roles: ['mentor'],
        corpCapabilities: ['menu.newbro'],
        isCurrentlyNewbro: false,
        isMentorMenteeEligible: false,
      },
      { mentor_pending_applications: 4 }
    )
    const newbro = findGroup(entries, 'nav.group.newbro')

    expect(newbro?.items.find((item) => item.to === '/newbro/mentor')?.badge).toBe(4)
    expect(newbro?.badge).toBe(4)
  })
})
