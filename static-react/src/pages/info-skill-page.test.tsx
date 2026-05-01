import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info skill page', () => {
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

  test('loads skills and queue', async () => {
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
              total_sp: 1000,
              unallocated_sp: 0,
              skills: [
                {
                  skill_id: 1,
                  skill_name: 'Gunnery',
                  group_name: 'Combat',
                  active_level: 3,
                  trained_level: 3,
                  skill_points: 100,
                  learned: true,
                },
              ],
              skill_queue: [
                {
                  queue_position: 1,
                  skill_id: 2,
                  skill_name: 'Missiles',
                  finished_level: 2,
                  level_start_sp: 0,
                  level_end_sp: 100,
                  training_start_sp: 0,
                  start_date: 0,
                  finish_date: 9999999999,
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/skill'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Gunnery (Combat)')).toBeInTheDocument()
    })
    expect(screen.getByText('Missiles L2')).toBeInTheDocument()
  })
})
