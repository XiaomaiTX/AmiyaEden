import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('galaxy registry page contains systems, captain, and admin sections', () => {
  assert.match(source, /galaxyRegistry\.systems\.title/)
  assert.match(source, /galaxyRegistry\.myEntries\.title/)
  assert.match(source, /galaxyRegistry\.admin\.systemsTab/)
  assert.match(source, /galaxyRegistry\.admin\.entriesTab/)
  assert.match(source, /galaxyRegistry\.admin\.analyticsTab/)
})

test('galaxy registry page wires create and end actions to member controls', () => {
  assert.match(source, /openCreateDialog\(row\)/)
  assert.match(source, /handleEndEntry\(row\)/)
  assert.match(source, /handleForceEndEntry\(row\)/)
})
