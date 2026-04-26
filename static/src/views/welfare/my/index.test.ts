import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('welfare my tables use ArtTable emptyText instead of standalone ElEmpty blocks', () => {
  assert.match(source, /<ArtTable[\s\S]*?:empty-text="t\('welfareMy\.noEligibleWelfares'\)"/)
  assert.match(source, /<ArtTable[\s\S]*?:empty-text="t\('welfareMy\.noApplications'\)"/)
  assert.doesNotMatch(source, /<ElEmpty/)
})

test('welfare my page keeps the tab content full-height so tables can scroll instead of clipping', () => {
  assert.match(source, /:deep\(\.el-card__body\)\s*\{[\s\S]*display:\s*flex/)
  assert.match(source, /:deep\(\.el-tabs\)\s*\{[\s\S]*flex:\s*1/)
  assert.match(source, /:deep\(\.el-tabs__content\)\s*\{[\s\S]*overflow:\s*hidden/)
  assert.match(source, /:deep\(\.el-tab-pane\)\s*\{[\s\S]*height:\s*100%/)
})

test('welfare my apply tab provides local filter controls for role, natural person, and welfare name', () => {
  assert.match(source, /v-model="roleFilter"/)
  assert.match(source, /v-model="naturalPersonFilter"/)
  assert.match(source, /v-model="welfareNameFilter"/)
  assert.match(source, /handleEligibleFilterChange/)
  assert.match(source, /handleEligibleFilterReset/)
})
