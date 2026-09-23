import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify({ code: 0, msg: 'ok', data }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

const charactersPayload = [{ character_id: 1001, character_name: 'Amiya', corporation_id: 1 }]

const walletPayload = {
  balance: 123456,
  journals: [
    {
      id: 1,
      amount: 99,
      balance: 123456,
      date: '2026-05-01',
      description: 'demo',
      first_party_id: 1,
      second_party_id: 2,
      ref_type: 'player_donation',
      reason: '',
    },
  ],
  ref_types: ['player_donation'],
  total: 1,
  page: 1,
  page_size: 50,
}

const walletAnalyticsPayload = {
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
      date: '2026-09-23',
      income: 500_000_000,
      expense: 400_000_000,
      net: 100_000_000,
      tax: 0,
      balance: 3_100_000_000,
      count: 2,
    },
  ],
  ref_type_breakdown: [
    { ref_type: 'player_donation', income: 1_500_000_000, expense: 0, count: 1 },
    { ref_type: 'market_transaction', income: 0, expense: 400_000_000, count: 2 },
  ],
}

function mockWalletApi() {
  // 按 URL 分发：子区块（分析）与页面各自请求，顺序不固定，用顺序式 mock 会互相消耗
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (...args: Parameters<typeof fetch>) => {
    const [input] = args
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
    if (url.includes('/info/wallet/analytics')) {
      return jsonResponse(walletAnalyticsPayload)
    }
    if (url.includes('/info/wallet')) {
      return jsonResponse(walletPayload)
    }
    return jsonResponse(charactersPayload)
  })
}

describe('info wallet page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      corpCapabilities: ['menu.info', 'info.wallet.read'],
    })
  })

  test('loads characters then renders wallet journal with the analytics section', async () => {
    mockWalletApi()

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/wallet'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getAllByText('player_donation').length).toBeGreaterThan(0)
    })

    expect(screen.getByText(/123,456\s*ISK/)).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('收支分析')).toBeInTheDocument()
    })
    expect(screen.getByText(/共 3 笔流水/)).toBeInTheDocument()
  })
})
