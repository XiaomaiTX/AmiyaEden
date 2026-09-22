import assert from 'node:assert/strict'
import test from 'node:test'
import { routeModules } from './index'

test('routeModules keeps Characters as the first top-level route', () => {
  assert.equal(routeModules[0]?.name, 'Characters')
  assert.equal(routeModules[0]?.path, '/characters')
})

test('routeModules registers MumbleCredentials right after Characters', () => {
  assert.equal(routeModules[1]?.name, 'MumbleCredentials')
  assert.equal(routeModules[1]?.path, '/mumble')
  assert.equal(routeModules[1]?.meta?.title, 'menus.mumble.title')
})
