import {
  buildBalanceSeries,
  buildDailyFlowSeries,
  buildRefTypeSeries,
  formatDayLabel,
  parseIsoDay,
  resolveAnalyticsRange,
  resolveFlowAxisBound,
  shiftIsoDay,
} from './wallet-analytics-series'
import type { WalletAnalyticsDailyPoint, WalletAnalyticsRefTypeItem } from '@/types/api/eve-info'

function createPoint(
  date: string,
  income: number,
  expense: number,
  balance: number
): WalletAnalyticsDailyPoint {
  return {
    date,
    income,
    expense,
    net: income - expense,
    tax: 0,
    balance,
    count: income || expense ? 1 : 0,
  }
}

describe('wallet analytics series helpers', () => {
  test('resolveAnalyticsRange clamps presets to the available range instead of a fixed window', () => {
    expect(resolveAnalyticsRange('all', '2026-08-24', '2026-09-23')).toEqual({
      from: '2026-08-24',
      to: '2026-09-23',
    })

    // 单日预设：起点与终点重合为同一天
    expect(resolveAnalyticsRange('1d', '2026-08-24', '2026-09-23')).toEqual({
      from: '2026-09-23',
      to: '2026-09-23',
    })

    // 可得数据不足 7 天：范围收缩到可得起点，而不是越界到 30 天窗口之外
    expect(resolveAnalyticsRange('7d', '2026-09-20', '2026-09-23')).toEqual({
      from: '2026-09-20',
      to: '2026-09-23',
    })

    // 可得数据超过 90 天：固定窗口贴合可得终点
    expect(resolveAnalyticsRange('90d', '2026-01-01', '2026-09-23')).toEqual({
      from: '2026-06-26',
      to: '2026-09-23',
    })
  })

  test('resolveAnalyticsRange returns null when the character has no collected entries', () => {
    expect(resolveAnalyticsRange('all', '', '')).toBeNull()
    expect(resolveAnalyticsRange('30d', '', '2026-09-23')).toBeNull()
  })

  test('shiftIsoDay moves by UTC days and parseIsoDay rejects malformed input', () => {
    expect(shiftIsoDay('2026-09-23', -29)).toBe('2026-08-25')
    expect(shiftIsoDay('2026-03-01', -1)).toBe('2026-02-28')
    expect(shiftIsoDay('not-a-day', 1)).toBe('not-a-day')
    expect(parseIsoDay('2026-09-23')).toBe(Date.UTC(2026, 8, 23))
    expect(parseIsoDay('2026-09-23T00:00:00Z')).toBeNull()
  })

  test('formatDayLabel renders a chart friendly MM-DD label', () => {
    expect(formatDayLabel('2026-09-23')).toBe('09-23')
    expect(formatDayLabel('')).toBe('')
  })

  test('buildBalanceSeries and buildDailyFlowSeries keep an ascending UTC-day order', () => {
    const points = [
      createPoint('2026-09-23', 30, 40, 900),
      createPoint('2026-09-21', 100, 0, 1000),
      createPoint('2026-09-22', 0, 0, 1000),
    ]

    expect(buildBalanceSeries(points)).toEqual([1000, 1000, 900])
    expect(buildDailyFlowSeries(points)).toEqual({
      labels: ['09-21', '09-22', '09-23'],
      income: [100, 0, 30],
      expense: [0, 0, 40],
    })
  })

  test('resolveFlowAxisBound rounds the larger flow up to a readable axis bound', () => {
    expect(resolveFlowAxisBound([120000, 40000], [180000, 0])).toBe(200000)
    expect(resolveFlowAxisBound([4500000000], [])).toBe(5000000000)
    expect(resolveFlowAxisBound([], [])).toBe(0)
  })

  test('buildRefTypeSeries keeps only the requested direction, sorted and truncated', () => {
    const breakdown: WalletAnalyticsRefTypeItem[] = [
      { ref_type: 'player_donation', income: 0, expense: 200, count: 1 },
      { ref_type: 'bounty_prizes', income: 80, expense: 0, count: 2 },
      { ref_type: 'market_transaction', income: 0, expense: 20, count: 1 },
      { ref_type: 'corporate_reward_payout', income: 500, expense: 0, count: 3 },
    ]

    expect(buildRefTypeSeries(breakdown, 'income')).toEqual({
      refTypes: ['corporate_reward_payout', 'bounty_prizes'],
      values: [500, 80],
    })
    expect(buildRefTypeSeries(breakdown, 'expense', 1)).toEqual({
      refTypes: ['player_donation'],
      values: [200],
    })
  })
})
