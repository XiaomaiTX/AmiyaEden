import { describe, expect, it } from 'vitest'
import { PAP_TREND_MONTHS, buildPapTrendSeries } from './pap-trend'

describe('buildPapTrendSeries', () => {
  it('returns empty array when input is empty', () => {
    expect(buildPapTrendSeries([])).toEqual([])
  })

  it('sorts months chronologically before building the series', () => {
    const series = buildPapTrendSeries([
      { year: 2026, month: 7, total_pap: 5 },
      { year: 2025, month: 12, total_pap: 1 },
      { year: 2026, month: 1, total_pap: 2 },
    ])

    expect(series).toHaveLength(PAP_TREND_MONTHS)
    expect(series[0]).toEqual({ year: 2025, month: 8, total_pap: 0 })
    expect(series.at(-1)).toEqual({ year: 2026, month: 7, total_pap: 5 })
    expect(series.find((item) => item.year === 2025 && item.month === 12)).toEqual({
      year: 2025,
      month: 12,
      total_pap: 1,
    })
    expect(series.find((item) => item.year === 2026 && item.month === 1)).toEqual({
      year: 2026,
      month: 1,
      total_pap: 2,
    })
  })

  it('fills missing months with zero values', () => {
    const series = buildPapTrendSeries([{ year: 2026, month: 7, total_pap: 4 }])

    expect(series).toHaveLength(PAP_TREND_MONTHS)
    expect(series.at(-1)).toEqual({ year: 2026, month: 7, total_pap: 4 })
    const zeroItems = series.filter((item) => item.total_pap === 0)
    expect(zeroItems).toHaveLength(PAP_TREND_MONTHS - 1)
  })

  it('handles cross-year month arithmetic correctly', () => {
    const series = buildPapTrendSeries([{ year: 2026, month: 2, total_pap: 1 }])

    expect(series[0]).toEqual({ year: 2025, month: 3, total_pap: 0 })
    expect(series.at(-1)).toEqual({ year: 2026, month: 2, total_pap: 1 })
    expect(series.filter((item) => item.year === 2025)).toHaveLength(10)
    expect(series.filter((item) => item.year === 2026)).toHaveLength(2)
  })
})
