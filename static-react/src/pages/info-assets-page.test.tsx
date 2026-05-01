import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info assets page', () => {
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

  test('loads assets and renders grouped items', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: { character_id: 1001, character_name: 'Amiya', corporation_id: 1 },
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
              total_items: 2,
              locations: [
                {
                  location_id: 1,
                  location_type: 'station',
                  location_name: 'Jita',
                  items: [
                    {
                      item_id: 10,
                      type_id: 34,
                      type_name: 'Tritanium',
                      group_name: 'Mineral',
                      category_id: 1,
                      quantity: 100,
                      location_flag: 'Hangar',
                      is_singleton: false,
                      character_id: 1001,
                      character_name: 'Amiya',
                      children: [],
                    },
                  ],
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/assets'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Tritanium')).toBeInTheDocument()
    })
    expect(screen.getByText('Jita')).toBeInTheDocument()
  })
})
