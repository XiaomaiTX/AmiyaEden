import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('fuel officer structures page fetches assigned list endpoint', () => {
  assert.match(source, /fetchMyAssignedCorporationStructures/)
  assert.match(source, /fuel_remaining/)
})
