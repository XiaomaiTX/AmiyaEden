import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info implants page', () => {
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

  test('loads implants and renders active implants and jump clones', async () => {
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
              home_location: { location_id: 1, location_type: 'station', location_name: 'Jita IV - Moon 4' },
              last_clone_jump_date: '2026-05-01T00:00:00Z',
              last_station_change_date: null,
              jump_fatigue_expire: null,
              last_jump_date: '2026-05-01T01:00:00Z',
              active_implants: [{ implant_id: 111, implant_name: 'Ocular Filter - Basic' }],
              jump_clones: [
                {
                  jump_clone_id: 10,
                  location: { location_id: 2, location_type: 'station', location_name: 'Amarr VIII' },
                  implants: [{ implant_id: 222, implant_name: 'Memory Augmentation - Basic' }],
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/implants'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Ocular Filter - Basic')).toBeInTheDocument()
    })

    expect(screen.getByText('Amarr VIII')).toBeInTheDocument()
    expect(screen.getByText('Memory Augmentation - Basic')).toBeInTheDocument()
  })
})
