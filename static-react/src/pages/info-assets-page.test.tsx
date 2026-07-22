import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

  test('renders location summaries on load', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            total_locations: 1,
            total_items: 2,
            locations: [
              {
                location_id: 60003760,
                location_type: 'station',
                location_name: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
                top_level_count: 2,
                root_item_count: 2,
                character_count: 1,
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
      expect(screen.getByText('Jita IV - Moon 4 - Caldari Navy Assembly Plant')).toBeInTheDocument()
    })
  })

  test('renders location items when a location is expanded', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              total_locations: 1,
              total_items: 2,
              locations: [
                {
                  location_id: 60003760,
                  location_type: 'station',
                  location_name: 'Jita',
                  top_level_count: 2,
                  root_item_count: 2,
                  character_count: 1,
                },
              ],
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
              location_id: 60003760,
              location_name: 'Jita',
              total_root_items: 1,
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
                  has_children: false,
                  child_count: 0,
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
      expect(screen.getByText('Jita')).toBeInTheDocument()
    })

    const locationBtn = screen.getByText('Jita')
    await userEvent.click(locationBtn)

    await waitFor(() => {
      expect(screen.getByText('Tritanium')).toBeInTheDocument()
    })
  })

  test('shows error when location list request fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValueOnce(new Error('Network failure'))

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/assets'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('加载位置列表失败。')).toBeInTheDocument()
    })
  })

  test('shows stable total_items in stats bar after search', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              total_locations: 2,
              total_items: 3,
              locations: [
                {
                  location_id: 60003760,
                  location_type: 'station',
                  location_name: 'Jita',
                  top_level_count: 2,
                  root_item_count: 2,
                  character_count: 1,
                },
                {
                  location_id: 60003761,
                  location_type: 'station',
                  location_name: 'Amarr',
                  top_level_count: 1,
                  root_item_count: 1,
                  character_count: 1,
                },
              ],
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
              total_locations: 1,
              total_items: 3,
              locations: [
                {
                  location_id: 60003760,
                  location_type: 'station',
                  location_name: 'Jita',
                  top_level_count: 2,
                  root_item_count: 2,
                  character_count: 1,
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
      expect(screen.getByText('Jita')).toBeInTheDocument()
    })

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: 'Jita' } })

    // After search, Amarr location disappears but total_items stays as 3
    await waitFor(() => {
      expect(screen.queryByText('Amarr')).not.toBeInTheDocument()
    })
    const statsBar = screen.getByTestId('asset-stats')
    expect(statsBar.textContent).toMatch(/3/)
  })

  test('ignores stale response from slow search', async () => {
    let resolveSlow!: (value: Response) => void
    const slowPromise = new Promise<Response>((resolve) => {
      resolveSlow = resolve
    })

    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              total_locations: 1,
              total_items: 2,
              locations: [
                {
                  location_id: 1,
                  location_type: 'station',
                  location_name: 'Initial',
                  top_level_count: 2,
                  root_item_count: 2,
                  character_count: 1,
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      // First search (slow) — won't resolve until we manually resolve it
      .mockReturnValueOnce(slowPromise)
      // Second search (fast) — resolves immediately
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              total_locations: 1,
              total_items: 5,
              locations: [
                {
                  location_id: 2,
                  location_type: 'station',
                  location_name: 'FastResult',
                  top_level_count: 5,
                  root_item_count: 5,
                  character_count: 1,
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
      expect(screen.getByText('Initial')).toBeInTheDocument()
    })

    const input = screen.getByRole('textbox')

    // Fire first search (will be slow)
    fireEvent.change(input, { target: { value: 'slow' } })
    // Fire second search (will be fast, wins the race)
    fireEvent.change(input, { target: { value: 'fast' } })

    await waitFor(() => {
      expect(screen.getByText('FastResult')).toBeInTheDocument()
    })

    // Resolve the stale (slow) response — it must NOT overwrite current state
    resolveSlow(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            total_locations: 1,
            total_items: 99,
            locations: [
              {
                location_id: 99,
                location_type: 'station',
                location_name: 'StaleResult',
                top_level_count: 99,
                root_item_count: 99,
                character_count: 1,
              },
            ],
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    // Let pending microtasks flush
    await new Promise((r) => setTimeout(r, 50))

    // FastResult must still be visible; StaleResult must not appear
    expect(screen.getByText('FastResult')).toBeInTheDocument()
    expect(screen.queryByText('StaleResult')).not.toBeInTheDocument()

    // Stats bar must reflect the fast (latest) response, not the stale one
    const statsBar = screen.getByTestId('asset-stats')
    expect(statsBar.textContent).toContain('5')
    expect(statsBar.textContent).not.toContain('99')
  })
})
