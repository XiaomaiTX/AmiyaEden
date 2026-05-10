import { buildShellMenuGroups } from '@/layout/menu-config'

describe('menu config', () => {
  test('hides protected routes for anonymous visitor', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: false,
      roles: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
    })

    expect(groups.length).toBe(0)
  })

  test('shows dashboard and info menus for logged-in member', () => {
    const groups = buildShellMenuGroups({
      isLoggedIn: true,
      roles: ['member'],
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
      isCurrentlyNewbro: true,
      isMentorMenteeEligible: true,
    })
    const allItems = groups.flatMap((group) => group.items)
    expect(allItems.some((item) => item.to.includes('/ticket/detail/'))).toBe(false)
    expect(allItems.some((item) => item.to.includes('/operation/fleet-detail/'))).toBe(false)
  })
})
