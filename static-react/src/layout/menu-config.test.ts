import { buildShellMenuGroups } from '@/layout/menu-config'

describe('menu config', () => {
  test('hides protected routes for anonymous visitor', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: false,
      roles: [],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(groups.length).toBe(0)
  })

  test('shows dashboard and info menus for logged-in member', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['member'],
      corpCapabilities: ['menu.dashboard', 'menu.info'],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    const dashboard = groups.find((group) => group.labelKey === 'nav.group.dashboard')
    const info = groups.find((group) => group.labelKey === 'nav.group.info')
    expect(dashboard).toBeDefined()
    expect(info).toBeDefined()
    expect(groups.find((group) => group.labelKey === 'nav.group.system')).toBeUndefined()
  })

  test('does not expose hidden detail routes in menu', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const allItems = groups.flatMap((group) => group.items)
    expect(allItems.some((item) => item.to.includes('/ticket/detail/'))).toBe(false)
    expect(allItems.some((item) => item.to.includes('/operation/fleet-detail/'))).toBe(false)
  })

  test('hides system routes when required corp capability is missing', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    const system = groups.find((group) => group.labelKey === 'nav.group.system')
    // Other system routes may not require corp capabilities, but the
    // user/task-manager/wallet/audit entries must be gated.
    if (system) {
      expect(system.items.some((item) => item.to.endsWith('/system/user'))).toBe(false)
      expect(system.items.some((item) => item.to.endsWith('/system/task-manager'))).toBe(false)
      expect(system.items.some((item) => item.to.endsWith('/system/wallet'))).toBe(false)
      expect(system.items.some((item) => item.to.endsWith('/system/audit'))).toBe(false)
    }
  })

  test('shows system routes when required corp capability is present', () => {
    const groups = buildShellMenuGroups({
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

    const system = groups.find((group) => group.labelKey === 'nav.group.system')
    expect(system).toBeDefined()
    expect(system?.items.some((item) => item.to.endsWith('/system/user'))).toBe(true)
    expect(system?.items.some((item) => item.to.endsWith('/system/task-manager'))).toBe(true)
    expect(system?.items.some((item) => item.to.endsWith('/system/wallet'))).toBe(true)
    expect(system?.items.some((item) => item.to.endsWith('/system/audit'))).toBe(true)
  })

  test('shows representative capability-gated routes across every migrated domain', () => {
    const groups = buildShellMenuGroups({
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
    const items = groups.flatMap((group) => group.items.map((item) => item.to))

    expect(items).toEqual(expect.arrayContaining([
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
    ]))
  })

  test('super_admin bypasses corp capability gates for menu visibility', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    const system = groups.find((group) => group.labelKey === 'nav.group.system')
    expect(system).toBeDefined()
  })

  test('shows Captain and Mentor entries to super_admin without specialist roles', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: [],
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const newbro = groups.find((group) => group.labelKey === 'nav.group.newbro')
    const items = newbro?.items.map((item) => item.to) ?? []

    expect(items).toContain('/newbro/captain')
    expect(items).toContain('/newbro/mentor')
  })
})
