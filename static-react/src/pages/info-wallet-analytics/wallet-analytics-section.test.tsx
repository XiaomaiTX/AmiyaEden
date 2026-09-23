import { render, screen, waitFor } from '@testing-library/react'
import { fetchInfoWalletAnalytics } from '@/api/eve-info'
import { I18nProvider } from '@/i18n'
import type { WalletAnalyticsResponse } from '@/types/api/eve-info'
import { WalletAnalyticsSection } from './wallet-analytics-section'

vi.mock('@/api/eve-info', () => ({
  fetchInfoWalletAnalytics: vi.fn(),
}))

const mockedFetchAnalytics = vi.mocked(fetchInfoWalletAnalytics)

const analyticsPayload: WalletAnalyticsResponse = {
  summary: {
    available_from: '2026-08-24',
    available_to: '2026-09-23',
    opening_balance: 2_000_000_000,
    closing_balance: 3_100_000_000,
    total_income: 1_500_000_000,
    total_expense: 400_000_000,
    total_net: 1_100_000_000,
    total_tax: 0,
    entry_count: 3,
  },
  daily_series: [
    {
      date: '2026-08-24',
      income: 1_000_000_000,
      expense: 0,
      net: 1_000_000_000,
      tax: 0,
      balance: 3_000_000_000,
      count: 1,
    },
    {
      date: '2026-08-25',
      income: 0,
      expense: 200_000_000,
      net: -200_000_000,
      tax: 0,
      balance: 2_800_000_000,
      count: 1,
    },
    {
      date: '2026-09-23',
      income: 500_000_000,
      expense: 200_000_000,
      net: 300_000_000,
      tax: 0,
      balance: 3_100_000_000,
      count: 1,
    },
  ],
  ref_type_breakdown: [
    { ref_type: 'player_donation', income: 1_500_000_000, expense: 0, count: 1 },
    { ref_type: 'market_transaction', income: 0, expense: 400_000_000, count: 2 },
  ],
}

function renderSection(characterId: number | null) {
  return render(
    <I18nProvider>
      <WalletAnalyticsSection characterId={characterId} />
    </I18nProvider>
  )
}

describe('wallet analytics section', () => {
  beforeEach(() => {
    mockedFetchAnalytics.mockReset()
  })

  test('renders the summary, resolved range and per-day charts for the selected character', async () => {
    mockedFetchAnalytics.mockResolvedValue(analyticsPayload)

    renderSection(1001)

    await waitFor(() => {
      expect(screen.getByText('收支分析')).toBeInTheDocument()
    })

    // 未指定区间时由后端返回的可得范围推导，默认展示全部可得流水
    expect(mockedFetchAnalytics).toHaveBeenCalledWith({ character_id: 1001 })
    expect(screen.getByText(/共 3 笔流水/)).toBeInTheDocument()
    expect(screen.getByText('净收支')).toBeInTheDocument()
    expect(screen.getByText('每日余额')).toBeInTheDocument()
    expect(screen.getByText('支出类型构成')).toBeInTheDocument()
  })

  test('prompts for a character before any request is made', async () => {
    renderSection(null)

    expect(screen.getByText('请先选择人物')).toBeInTheDocument()
    expect(mockedFetchAnalytics).not.toHaveBeenCalled()
  })

  test('falls back to an empty state when the character has no collected entries', async () => {
    mockedFetchAnalytics.mockResolvedValue({
      summary: {
        available_from: '',
        available_to: '',
        opening_balance: null,
        closing_balance: null,
        total_income: 0,
        total_expense: 0,
        total_net: 0,
        total_tax: 0,
        entry_count: 0,
      },
      daily_series: [],
      ref_type_breakdown: [],
    })

    renderSection(1001)

    await waitFor(() => {
      expect(screen.getByText('暂无可分析的流水数据')).toBeInTheDocument()
    })
  })
})
