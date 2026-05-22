import assert from 'node:assert/strict'
import test from 'node:test'
import { routeModules } from './index'

test('routeModules keeps Characters as the first top-level route', () => {
  assert.equal(routeModules[0]?.name, 'Characters')
  assert.equal(routeModules[0]?.path, '/characters')
})
