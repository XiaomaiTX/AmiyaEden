import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('galaxy registry page is organized into current, captain, and admin tabs', () => {
  assert.match(source, /galaxyRegistry\.tabs\.current/)
  assert.match(source, /galaxyRegistry\.tabs\.captain/)
  assert.match(source, /galaxyRegistry\.tabs\.admin/)
})

test('galaxy registry entry tables use ledger ArtTable patterns', () => {
  assert.match(source, /<ArtTableHeader[\s\S]*myColumnChecks/s)
  assert.match(source, /<ArtTable[\s\S]*visual-variant="ledger"/s)
  assert.match(source, /useTable\(\{[\s\S]*fetchMyGalaxyRegistryEntries/s)
  assert.match(source, /useTable\(\{[\s\S]*fetchAdminGalaxyRegistryEntries/s)
})

test('galaxy registry page supports expected-end editing and admin save-all flows', () => {
  assert.match(source, /galaxyRegistry\.actions\.editExpectedEnd/)
  assert.match(source, /updateGalaxyRegistryEntryExpectedEndAt/)
  assert.match(source, /galaxyRegistry\.admin\.saveAllSystems/)
  assert.match(source, /handleSaveAllSystems/)
  assert.match(source, /updateAdminGalaxyRegistryEntryValidation/)
})
