import assert from 'node:assert/strict'
import test from 'node:test'
import { routeModules } from './index'

test('routeModules keeps Characters as the first top-level route', () => {
  assert.equal(routeModules[0]?.name, 'Characters')
  assert.equal(routeModules[0]?.path, '/characters')
  assert.equal(routeModules[0]?.meta?.isHide, true)
})

test('routeModules registers MumbleCredentials right after Characters', () => {
  assert.equal(routeModules[1]?.name, 'MumbleCredentials')
  assert.equal(routeModules[1]?.path, '/mumble')
  assert.equal(routeModules[1]?.meta?.title, 'menus.mumble.title')
})

test('Fuxi Hall is a single direct page with the existing member capability gate', () => {
  const hall = routeModules.find((route) => route.path === '/fuxi-hall')
  assert.equal(hall?.name, 'FuxiHall')
  assert.equal(hall?.component, '/fuxi-hall/index')
  assert.equal(hall?.meta?.title, 'menus.fuxiHall.fame')
  assert.equal(hall?.meta?.login, true)
  assert.deepEqual(hall?.meta?.corpCapabilitiesAny, ['menu.fuxi_hall'])
  assert.equal(hall?.meta?.roles, undefined)
  assert.equal(hall?.children, undefined)
})

test('dashboard exposes the admin-only skill readiness page', () => {
  const dashboard = routeModules.find((route) => route.name === 'Dashboard')
  const page = dashboard?.children?.find((route) => route.path === 'skill-readiness')
  assert.equal(page?.name, 'DashboardSkillReadiness')
  assert.equal(page?.component, '/dashboard/skill-readiness')
  assert.deepEqual(page?.meta?.roles, ['super_admin', 'admin'])
})

test('every route module title uses a menus.* localization key', () => {
  const visit = (routes: typeof routeModules) => {
    for (const route of routes) {
      assert.ok(
        typeof route.meta?.title === 'string' && route.meta.title.startsWith('menus.'),
        `route ${String(route.name ?? route.path)} has non-menus title: ${String(route.meta?.title)}`
      )
      if (route.children?.length) visit(route.children)
    }
  }
  visit(routeModules)
})
