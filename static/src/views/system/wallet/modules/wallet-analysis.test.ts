import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./wallet-analysis.vue', import.meta.url), 'utf8')

test('wallet analysis tables use ArtTable ledger variant instead of native ElTable', () => {
  assert.doesNotMatch(source, /<ElTable[\s>]/)
  assert.match(source, /visual-variant="ledger"/)
  assert.match(source, /const topUserColumns = computed<ColumnOption\[]>\(/)
  assert.match(source, /const largeTransactionColumns = computed<ColumnOption\[]>\(/)
  assert.match(source, /const frequentAdjustmentColumns = computed<ColumnOption\[]>\(/)
  assert.match(source, /const operatorConcentrationColumns = computed<ColumnOption\[]>\(/)
})

test('wallet analysis summary still uses totalBalance i18n key', () => {
  assert.match(source, /label:\s*t\('walletAdmin\.analysis\.totalBalance'\)/)
})

test('wallet analysis keeps date range picker visible in filter row', () => {
  assert.match(source, /<ElDatePicker[\s\S]*class="filter-date-range"/)
  assert.match(source, /\.filter-date-range\s*\{[\s\S]*min-width:\s*280px/)
})

test('wallet analysis grid items can shrink to prevent chart overflow clipping', () => {
  assert.match(source, /\.grid-two\s*\{[\s\S]*>\s*\*\s*\{[\s\S]*min-width:\s*0/)
})
