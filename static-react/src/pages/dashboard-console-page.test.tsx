import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

function mockDashboardResponse(payload: unknown) {
  vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
    new Response(JSON.stringify({ code: 0, msg: 'ok', data: payload }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  )
}

describe('dashboard console page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      corpCapabilities: ['menu.dashboard'],
    })
  })

  test('renders cards, fleets, PAP charts and SRP table from api data', async () => {
    mockDashboardResponse({
      cards: {
        eve_wallet_balance: 1_250_000_000,
        eve_skill_points: 87_500_000,
        system_wallet_balance: 1234.5,
        alliance_pap: 6,
      },
      fleets: [
        {
          source: 'alliance',
          id: 'fleet-1',
          title: 'Moon Operation',
          start_at: '2026-07-22T10:00:00Z',
          character_name: 'Amiya',
          ship_type_name: 'Ferox',
          importance: 'cta',
          pap_count: 2,
        },
      ],
      pap_stats: {
        alliance: [{ year: 2026, month: 6, total_pap: 3 }],
        internal: [],
      },
      srp_list: [
        {
          id: 100,
          character_name: 'Amiya',
          ship_name: 'Ferox',
          solar_system_name: 'Jita',
          killmail_time: '2026-07-20T08:30:00Z',
          recommended_amount: 100_000_000,
          final_amount: 90_000_000,
          review_status: 'approved',
          payout_status: 'notpaid',
          created_at: '2026-07-20T09:00:00Z',
        },
      ],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/console'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('1.25 B')).toBeInTheDocument()
    })

    expect(screen.getByText('Moon Operation')).toBeInTheDocument()
    expect(screen.getByText('Jita')).toBeInTheDocument()
    expect(screen.getByText(/90.00 M/)).toBeInTheDocument()
    expect(screen.getByText('已批准')).toBeInTheDocument()
  })

  test('renders empty placeholders when no data is returned', async () => {
    mockDashboardResponse({
      cards: {
        eve_wallet_balance: 0,
        eve_skill_points: 0,
        system_wallet_balance: 0,
        alliance_pap: 0,
      },
      fleets: [],
      pap_stats: { alliance: [], internal: [] },
      srp_list: [],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/console'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getAllByText('暂无舰队参与记录').length).toBeGreaterThan(0)
    })
    expect(screen.getAllByText('暂无 PAP 数据').length).toBe(2)
    expect(screen.getByText('暂无补损申请记录')).toBeInTheDocument()
  })
})
