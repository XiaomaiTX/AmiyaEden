import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('ticket my tickets page', () => {
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

  test('loads ticket list and filters by status', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              list: [
                {
                  id: 1,
                  user_id: 1001,
                  category_id: 10,
                  title: 'Need help',
                  description: 'Please review',
                  status: 'pending',
                  priority: 'high',
                  created_at: '2026-05-01T00:00:00Z',
                  updated_at: '2026-05-01T00:00:00Z',
                },
                {
                  id: 2,
                  user_id: 1001,
                  category_id: 11,
                  title: 'Resolved issue',
                  description: 'Done',
                  status: 'completed',
                  priority: 'low',
                  created_at: '2026-05-01T00:00:00Z',
                  updated_at: '2026-05-01T00:00:00Z',
                },
              ],
              total: 2,
              page: 1,
              pageSize: 20,
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
              list: [
                {
                  id: 1,
                  user_id: 1001,
                  category_id: 10,
                  title: 'Need help',
                  description: 'Please review',
                  status: 'pending',
                  priority: 'high',
                  created_at: '2026-05-01T00:00:00Z',
                  updated_at: '2026-05-01T00:00:00Z',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 20,
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/ticket/my-tickets'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Need help')).toBeInTheDocument()
    })
    expect(screen.getByText('Resolved issue')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'pending' } })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))

    await waitFor(() => {
      expect(screen.getAllByText('Need help')).toHaveLength(1)
    })
    expect(screen.queryByText('Resolved issue')).not.toBeInTheDocument()
  })
})
