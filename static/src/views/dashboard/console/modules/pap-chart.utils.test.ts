import assert from 'node:assert/strict'
import test from 'node:test'

import { buildPapTrendSeries, formatPapTrendMonthLabel, PAP_TREND_MONTHS } from './pap-chart.utils'

test('buildPapTrendSeries returns a fixed 12-month window and fills gaps with zero', () => {
  const series = buildPapTrendSeries([
    { year: 2025, month: 4, total_pap: 10 },
    { year: 2025, month: 6, total_pap: 30 }
  ])

  assert.equal(series.length, PAP_TREND_MONTHS)
  assert.deepEqual(series.at(-1), { year: 2025, month: 6, total_pap: 30 })
  assert.equal(
    series.some((item) => item.year === 2025 && item.month === 5 && item.total_pap === 0),
    true
  )
})

test('buildPapTrendSeries keeps month order stable across year boundaries', () => {
  const series = buildPapTrendSeries([
    { year: 2024, month: 12, total_pap: 10 },
    { year: 2025, month: 2, total_pap: 20 }
  ])

  assert.deepEqual(series.slice(-3).map(formatPapTrendMonthLabel), [
    '2024-12',
    '2025-01',
    '2025-02'
  ])
  assert.deepEqual(series.at(-2), { year: 2025, month: 1, total_pap: 0 })
  assert.deepEqual(series.at(-1), { year: 2025, month: 2, total_pap: 20 })
})
