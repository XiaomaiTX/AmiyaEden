import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info wallet page', () => {
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

  test('loads characters then renders wallet journal', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: [{ character_id: 1001, character_name: 'Amiya', corporation_id: 1 }],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
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
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/wallet'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getAllByText('player_donation').length).toBeGreaterThan(0)
    })

    expect(screen.getByText(/123,456\s*ISK/)).toBeInTheDocument()
  })
})

