import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(
  new URL('../../../api/corporation-structures.ts', import.meta.url),
  'utf8'
)

test('corporation structures page uses single-page tab query routing', () => {
  assert.match(source, /type StructureTab = 'list' \| 'settings'/)
  assert.match(source, /const activeTab = ref<StructureTab>\(normalizeTab\(route\.query\.tab\)\)/)
  assert.match(source, /watch\(\s*\(\) => route\.query\.tab,/)
  assert.match(source, /void router\.replace\(\{\s*query:\s*\{\s*\.\.\.route\.query,\s*tab/s)
})

test('corporation structures page wires settings and list tabs', () => {
  assert.match(source, /corporationStructures\.tabs\.list/)
  assert.match(source, /corporationStructures\.tabs\.settings/)
  assert.match(source, /saveAuthorizations/)
  assert.match(source, /handleRunTaskForSelectedCorporation/)
  assert.doesNotMatch(source, /handleRefreshCorporation/)
  assert.doesNotMatch(source, /refreshThisCorporation/)
})

test('corporation structures api module exposes all required endpoints', () => {
  assert.match(apiSource, /\/api\/v1\/dashboard\/corporation-structures\/settings/)
  assert.match(apiSource, /\/settings\/authorizations/)
  assert.match(apiSource, /\/corporation-structures\/list/)
  assert.match(apiSource, /\/corporation-structures\/run-task/)
  assert.doesNotMatch(apiSource, /\/corporation-structures\/refresh/)
})
