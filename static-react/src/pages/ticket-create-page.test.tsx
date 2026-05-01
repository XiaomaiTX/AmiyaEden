import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { usePreferenceStore } from '@/stores'
import { useSessionStore } from '@/stores'

describe('ticket create page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: [],
    })
    usePreferenceStore.setState({
      locale: 'zh-CN',
      sidebarCollapsed: false,
      theme: 'system',
    })
  })

  test('loads categories, submits a ticket, and returns to my tickets', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: [
              {
                id: 10,
                name: '技术支持',
                name_en: 'Support',
                description: 'Support issues',
                sort_order: 1,
                enabled: true,
                created_at: '2026-05-01T00:00:00Z',
                updated_at: '2026-05-01T00:00:00Z',
              },
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
              id: 99,
              user_id: 1001,
              category_id: 10,
              title: '需要帮助',
              description: '请帮我确认工单流程。',
              status: 'pending',
              priority: 'medium',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
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
                  id: 99,
                  user_id: 1001,
                  category_id: 10,
                  title: '需要帮助',
                  description: '请帮我确认工单流程。',
                  status: 'pending',
                  priority: 'medium',
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
      initialEntries: ['/ticket/create'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('技术支持')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText('请输入工单标题'), {
      target: { value: '需要帮助' },
    })
    fireEvent.change(screen.getByPlaceholderText('请详细描述问题、需求或建议'), {
      target: { value: '请帮我确认工单流程。' },
    })
    fireEvent.click(screen.getByRole('button', { name: '提交工单' }))

    await waitFor(() => {
      expect(screen.getByText('我的工单')).toBeInTheDocument()
    })
    expect(screen.getByText('需要帮助')).toBeInTheDocument()
  })

  test('shows localized category names in English locale', async () => {
    usePreferenceStore.setState({
      locale: 'en-US',
      sidebarCollapsed: false,
      theme: 'system',
    })

    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: [
            {
              id: 10,
              name: '技术支持',
              name_en: 'Support',
              description: 'Support issues',
              sort_order: 1,
              enabled: true,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/ticket/create'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Support')).toBeInTheDocument()
    })
    expect(screen.getByPlaceholderText('Enter ticket title')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Submit Ticket' })).toBeInTheDocument()
  })
})
