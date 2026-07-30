import { appRouteSpecs } from '@/app/migration-routes'
import {
  evaluateRouteAccess,
  hasCorpCapabilityPermission,
  hasRouteRolePermission,
} from '@/app/route-access'

const session = {
  isLoggedIn: true,
  roles: ['user'],
  corpCapabilities: ['menu.dashboard'],
  isCurrentlyNewbro: false,
  isMentorMenteeEligible: false,
}

describe('hasRouteRolePermission', () => {
  test('allows routes without roles and matches any required role', () => {
    expect(hasRouteRolePermission(['user'], [])).toBe(true)
    expect(hasRouteRolePermission(['user', 'admin'], ['admin'])).toBe(true)
    expect(hasRouteRolePermission(['user'], ['admin'])).toBe(false)
  })
})

describe('route access evaluation', () => {
  test('separates authentication, role, capability, and eligibility decisions', () => {
    expect(evaluateRouteAccess({ jwt: true }, { ...session, isLoggedIn: false })).toBe('login')
    expect(evaluateRouteAccess({ login: true }, { ...session, roles: ['guest'] })).toBe('forbidden')
    expect(evaluateRouteAccess({ roles: ['admin'] }, session)).toBe('forbidden')
    expect(
      evaluateRouteAccess({ corpCapabilitiesAll: ['menu.dashboard', 'dashboard.manage'] }, session)
    ).toBe('forbidden')
    expect(evaluateRouteAccess({ requiresNewbro: true }, session)).toBe('forbidden')
    expect(
      evaluateRouteAccess(
        { requiresMentorMenteeEligibility: true },
        { ...session, isMentorMenteeEligible: true }
      )
    ).toBe('allow')
  })

  test('supports capability AND/OR semantics and super-admin bypass', () => {
    expect(
      hasCorpCapabilityPermission(['user'], ['a', 'b'], {
        corpCapabilitiesAll: ['a', 'b'],
        corpCapabilitiesAny: ['b', 'c'],
      })
    ).toBe(true)
    expect(
      hasCorpCapabilityPermission(['user'], ['a'], {
        corpCapabilitiesAny: ['b', 'c'],
      })
    ).toBe(false)
    expect(
      hasCorpCapabilityPermission(['super_admin'], [], {
        corpCapabilitiesAll: ['missing'],
      })
    ).toBe(true)
  })

  test('keeps representative route declarations aligned with enforced capabilities', () => {
    const cases = [
      [
        'dashboard/npc-kills',
        ['admin'],
        ['menu.dashboard', 'dashboard.npc_kills.corp', 'info.npc_kills.corp'],
      ],
      ['info/wallet', ['user'], ['menu.info', 'info.wallet.read']],
      ['operation/fleet-configs', ['user'], ['menu.operation']],
      ['skill-planning/skill-plans', ['user'], ['menu.skill_planning']],
      ['welfare/approval', ['welfare'], ['welfare.approval']],
      ['shop/manage', ['admin'], ['menu.shop', 'shop.manage', 'shop.admin.product.manage']],
      ['ticket/management', ['admin'], ['menu.ticket', 'ticket.manage', 'ticket.admin.read']],
      ['ticket/categories', ['admin'], ['menu.ticket', 'ticket.manage', 'ticket.admin.read']],
      ['system/pap', ['admin'], ['system.manage']],
    ] as const

    for (const [path, roles, capabilities] of cases) {
      const route = appRouteSpecs.find((item) => item.path === path)
      expect(route).toBeDefined()
      expect(hasCorpCapabilityPermission([...roles], [...capabilities], route?.meta ?? {})).toBe(
        true
      )
    }
  })

  test('keeps ticket category reads separate from category mutations', () => {
    const route = appRouteSpecs.find((item) => item.path === 'ticket/categories')
    expect(route?.meta?.corpCapabilitiesAll).toEqual([
      'menu.ticket',
      'ticket.manage',
      'ticket.admin.read',
    ])
    expect(route?.meta?.corpCapabilitiesAll).not.toContain('ticket.admin.manage')
  })

  test('keeps auto-role access role-only because the backend enforces its role boundary', () => {
    const route = appRouteSpecs.find((item) => item.path === 'system/auto-role')
    expect(route?.meta?.roles).toEqual(['super_admin'])
    expect(route?.meta?.corpCapabilitiesAll).toBeUndefined()
    expect(route?.meta?.corpCapabilitiesAny).toBeUndefined()
  })

  test('keeps super-admin in specialist route declarations', () => {
    for (const path of ['newbro/captain', 'newbro/mentor']) {
      const route = appRouteSpecs.find((item) => item.path === path)
      expect(route?.meta?.roles).toContain('super_admin')
      expect(hasRouteRolePermission(['super_admin'], route?.meta?.roles ?? [])).toBe(true)
    }
  })
})
