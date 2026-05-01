import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('dashboard corporation structures page', () => {
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

  test('loads list and settings tabs', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              corporations: [
                {
                  corporation_id: 1,
                  corporation_name: 'Amiya Corp',
                  authorized_character_id: 1001,
                  director_characters: [
                    { user_id: 1, character_id: 1001, character_name: 'Amiya' },
                  ],
                },
              ],
              fuel_notice_threshold_days: 7,
              timer_notice_threshold_days: 5,
            },
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
              systems: [
                {
                  system_id: 1,
                  system_name: 'Jita',
                  region_id: 1,
                  region_name: 'The Forge',
                  security: 0.9,
                },
              ],
              types: [{ type_id: 1, type_name: 'Astrahus' }],
              services: [{ name: 'Clone Bay' }],
            },
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
              items: [
                {
                  corporation_id: 1,
                  corporation_name: 'Amiya Corp',
                  structure_id: 10,
                  name: 'HQ Astrahus',
                  type_id: 1,
                  type_name: 'Astrahus',
                  system_id: 1,
                  system_name: 'Jita',
                  region_id: 1,
                  region_name: 'The Forge',
                  security: 0.9,
                  state: 'shield_vulnerable',
                  services: [{ name: 'Clone Bay', state: 'online' }],
                  fuel_expires: '2026-05-01T00:00:00Z',
                  fuel_remaining: '12h',
                  fuel_remaining_hours: 12,
                  reinforce_hour: 18,
                  state_timer_start: '2026-05-01T00:00:00Z',
                  state_timer_end: '2026-05-02T00:00:00Z',
                  updated_at: 1710000000,
                },
              ],
              total: 1,
              page: 1,
              page_size: 20,
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/corporation-structures'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('HQ Astrahus')).toBeInTheDocument()
    })

    expect(screen.getByText('Amiya Corp')).toBeInTheDocument()
    expect(screen.getByText('Structure List')).toBeInTheDocument()
  })
})
