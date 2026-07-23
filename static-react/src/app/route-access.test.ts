import { describe, expect, test } from 'vitest'
import { appRouteSpecs } from '@/app/migration-routes'
import { hasCorpCapabilityPermission, hasRouteRolePermission } from '@/app/route-access'

describe('hasRouteRolePermission', () => {
  test('returns true when no roles are required', () => {
    expect(hasRouteRolePermission(['user'], [])).toBe(true)
  })

  test('returns true when user has at least one required role', () => {
    expect(hasRouteRolePermission(['user', 'admin'], ['admin'])).toBe(true)
  })

  test('returns false when user has none of the required roles', () => {
    expect(hasRouteRolePermission(['user'], ['admin'])).toBe(false)
  })
})

describe('hasCorpCapabilityPermission', () => {
  test('super_admin short-circuits to true', () => {
    expect(
      hasCorpCapabilityPermission(['super_admin'], [], {
        corpCapabilitiesAll: ['shop.manage', 'shop.admin.product.manage'],
        corpCapabilitiesAny: ['srp.user'],
      })
    ).toBe(true)
  })

  test('corpCapabilitiesAll enforces AND semantics', () => {
    const meta = {
      corpCapabilitiesAll: ['shop.manage', 'shop.admin.product.manage'],
    }

    expect(
      hasCorpCapabilityPermission(['admin'], ['shop.manage', 'shop.admin.product.manage'], meta)
    ).toBe(true)
    expect(hasCorpCapabilityPermission(['admin'], ['shop.manage'], meta)).toBe(false)
    expect(hasCorpCapabilityPermission(['admin'], [], meta)).toBe(false)
  })

  test('corpCapabilitiesAny enforces OR semantics', () => {
    const meta = { corpCapabilitiesAny: ['srp.user', 'srp.manage'] }

    expect(hasCorpCapabilityPermission(['admin'], ['srp.user'], meta)).toBe(true)
    expect(hasCorpCapabilityPermission(['admin'], ['srp.manage'], meta)).toBe(true)
    expect(hasCorpCapabilityPermission(['admin'], ['menu.shop'], meta)).toBe(false)
  })

  test('returns true when meta has no corp capability requirements', () => {
    expect(hasCorpCapabilityPermission(['admin'], [], {})).toBe(true)
  })

  test('combination of All and Any requires both gates to pass', () => {
    const meta = {
      corpCapabilitiesAll: ['ticket.manage'],
      corpCapabilitiesAny: ['ticket.admin.read', 'ticket.admin.manage'],
    }

    expect(
      hasCorpCapabilityPermission(['admin'], ['ticket.manage', 'ticket.admin.read'], meta)
    ).toBe(true)
    expect(hasCorpCapabilityPermission(['admin'], ['ticket.manage'], meta)).toBe(false)
    expect(
      hasCorpCapabilityPermission(['admin'], ['ticket.admin.read', 'ticket.admin.manage'], meta)
    ).toBe(false)
  })

  test('matches every protected React route with its enforced capability requirements', () => {
    const cases = [
      ['dashboard/npc-kills', ['admin'], ['menu.dashboard', 'dashboard.npc_kills.corp', 'info.npc_kills.corp']],
      ['info/wallet', ['user'], ['menu.info', 'info.wallet.read']],
      ['operation/fleet-configs', ['user'], ['menu.operation']],
      ['skill-planning/skill-plans', ['user'], ['menu.skill_planning']],
      ['welfare/approval', ['welfare'], ['welfare.approval']],
      ['newbro/select-captain', ['user'], ['menu.newbro']],
      ['shop/manage', ['admin'], ['menu.shop', 'shop.manage', 'shop.admin.product.manage']],
      ['srp/srp-manage', ['srp'], ['srp.manage']],
      ['ticket/management', ['admin'], ['ticket.manage', 'ticket.admin.read']],
      ['system/pap', ['admin'], ['system.manage']],
    ] as const

    for (const [path, roles, capabilities] of cases) {
      const route = appRouteSpecs.find((item) => item.path === path)
      expect(route).toBeDefined()
      expect(hasCorpCapabilityPermission([...roles], [...capabilities], route?.meta ?? {})).toBe(true)
    }

    const npcKills = appRouteSpecs.find((item) => item.path === 'dashboard/npc-kills')
    expect(
      hasCorpCapabilityPermission(['admin'], ['menu.dashboard', 'dashboard.npc_kills.corp'], npcKills?.meta ?? {})
    ).toBe(false)
  })

  test('super_admin has Captain and Mentor route entries without the specialist roles', () => {
    for (const path of ['newbro/captain', 'newbro/mentor']) {
      const route = appRouteSpecs.find((item) => item.path === path)
      expect(route?.meta?.roles).toContain('super_admin')
      expect(hasRouteRolePermission(['super_admin'], route?.meta?.roles ?? [])).toBe(true)
      expect(hasCorpCapabilityPermission(['super_admin'], [], route?.meta ?? {})).toBe(true)
    }
  })
})
