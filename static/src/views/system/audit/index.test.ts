import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('audit admin page gates export entry behind system.audit.export capability', () => {
  assert.match(source, /useCorpCapability/)
  assert.match(source, /hasCapability\('system\.audit\.export'\)/)
  assert.match(source, /canExport/)
  assert.match(source, /v-if="!canExport"/)
})
