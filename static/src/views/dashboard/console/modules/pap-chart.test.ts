import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./pap-chart.vue', import.meta.url), 'utf8')

test('dashboard console PAP chart uses a line chart instead of a bar chart', () => {
  assert.match(source, /<ArtLineChart[\s>]/)
  assert.doesNotMatch(source, /<ArtBarChart[\s>]/)
  assert.doesNotMatch(source, /barWidth="/)
})

test('dashboard console PAP chart keeps the monthly trend data wiring intact', () => {
  assert.match(source, /const chartData = computed\(\(\) => \{/)
  assert.match(source, /chartData\.value\.map\(\(d\) => t\('console\.papChart\.monthLabel'/)
  assert.match(source, /chartData\.value\.map\(\(d\) => d\.total_pap\)/)
})
