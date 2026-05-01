import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('dashboard npc kills page', () => {
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

  test('loads corporation npc kill report', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            summary: {
              total_bounty: 1234567.89,
              total_ess: 0,
              total_incursion: 0,
              total_mission: 0,
              total_tax: 12345,
              actual_income: 1222222.89,
              total_records: 3,
              estimated_hours: 5,
            },
            members: [
              {
                character_id: 1001,
                character_name: 'Amiya',
                total_bounty: 1000,
                total_ess: 0,
                total_incursion: 0,
                total_mission: 0,
                total_tax: 100,
                actual_income: 900,
                record_count: 2,
              },
            ],
            by_system: [
              {
                solar_system_id: 30000142,
                solar_system_name: 'Jita',
                count: 2,
                amount: 1000,
              },
            ],
            trend: [
              {
                date: '2026-05-01',
                amount: 1000,
                count: 2,
              },
            ],
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/npc-kills'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('1,234,567.89')).toBeInTheDocument()
    })

    expect(screen.getByText('Amiya')).toBeInTheDocument()
    expect(screen.getByText('Jita')).toBeInTheDocument()
  })
})
