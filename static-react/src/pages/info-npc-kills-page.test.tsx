import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info npc kills page', () => {
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

  test('loads personal npc kill report with characters, journal rows and trends', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: [
              { character_id: 1001, character_name: 'Amiya', corporation_id: 1 },
              { character_id: 1002, character_name: 'Miya', corporation_id: 1 },
            ],
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
              summary: {
                total_bounty: 654321,
                total_ess: 0,
                total_incursion: 0,
                total_mission: 0,
                total_tax: 1234,
                actual_income: 653087,
                total_records: 2,
                estimated_hours: 4,
              },
              by_npc: [
                {
                  npc_id: 1,
                  npc_name: 'Pirate Drone',
                  count: 2,
                  amount: 500,
                },
              ],
              by_system: [
                {
                  solar_system_id: 30000142,
                  solar_system_name: 'Jita',
                  count: 2,
                  amount: 500,
                },
              ],
              trend: [
                {
                  date: '2026-05-01',
                  amount: 500,
                  count: 2,
                },
              ],
              journals: [
                {
                  id: 1,
                  character_id: 1001,
                  character_name: 'Amiya',
                  amount: 500,
                  tax: 50,
                  date: '2026-05-01 12:00:00',
                  ref_type: 'bounty_prizes',
                  solar_system_id: 30000142,
                  solar_system_name: 'Jita',
                  reason: 'Mission clear',
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
      initialEntries: ['/info/npc-kills'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Amiya')).toBeInTheDocument()
    })

    expect(screen.getByText('Pirate Drone')).toBeInTheDocument()
    expect(screen.getByText('Jita')).toBeInTheDocument()
    expect(screen.getByText('Mission clear')).toBeInTheDocument()
  })
})
