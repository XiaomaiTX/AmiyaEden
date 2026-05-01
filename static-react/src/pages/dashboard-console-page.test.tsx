import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('dashboard console page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: [],
    })
  })

  test('renders dashboard cards from api data', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            cards: {
              online_count: 12,
              total_assets_count: 34,
              total_assets_price: 5600,
              my_pap_count: 8,
            },
            fleets: [],
            pap_stats: { alliance: [], internal: [] },
            srp_list: [],
          },
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/console'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('工作台')).toBeInTheDocument()
    })

    expect(screen.getByText('12')).toBeInTheDocument()
  })
})

