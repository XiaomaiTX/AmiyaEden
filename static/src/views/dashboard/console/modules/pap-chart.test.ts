import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./pap-chart.vue', import.meta.url), 'utf8')

test('dashboard console PAP chart uses the shared 12-month trend helper', () => {
  assert.match(source, /buildPapTrendSeries\(props\.data\)/)
  assert.match(
    source,
    /t\('console\.papChart\.monthLabel', \{\s*year:\s*d\.year,\s*month:\s*String\(d\.month\)\.padStart\(2, '0'\)\s*\}\)/
  )
  assert.match(source, /chartData\.value\.map\(\(d\) => d\.total_pap\)/)
  assert.match(source, /chartData\.length/)
})

test('dashboard console PAP chart still renders as a line chart', () => {
  assert.match(source, /<ArtLineChart[\s>]/)
  assert.doesNotMatch(source, /<ArtBarChart[\s>]/)
  assert.doesNotMatch(source, /barWidth="/)
})
