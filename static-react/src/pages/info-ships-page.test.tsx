import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info ships page', () => {
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

  test('loads ships and renders summary and rows', async () => {
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
              total_ships: 2,
              flyable_ships: 1,
              ships: [
                {
                  type_id: 603,
                  type_name: 'Merlin',
                  group_id: 25,
                  group_name: 'Frigate',
                  market_group_id: 1,
                  market_group_name: 'Ships',
                  race_id: 1,
                  race_name: 'Caldari',
                  can_fly: true,
                  skill_reqs: [],
                },
                {
                  type_id: 593,
                  type_name: 'Punisher',
                  group_id: 25,
                  group_name: 'Frigate',
                  market_group_id: 1,
                  market_group_name: 'Ships',
                  race_id: 2,
                  race_name: 'Amarr',
                  can_fly: false,
                  skill_reqs: [],
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/ships'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Merlin')).toBeInTheDocument()
    })

    expect(screen.getByText('Punisher')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })
})
