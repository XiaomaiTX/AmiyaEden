import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { usePreferenceStore, useSessionStore } from '@/stores'

describe('ticket detail page', () => {
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

  test('loads ticket detail and submits a reply', async () => {
    vi.spyOn(globalThis, 'fetch')
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
            data: [
              {
                id: 1,
                ticket_id: 99,
                user_id: 1001,
                content: '初始回复',
                is_internal: false,
                created_at: '2026-05-01T01:00:00Z',
                updated_at: '2026-05-01T01:00:00Z',
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
              id: 2,
              ticket_id: 99,
              user_id: 1001,
              content: '我来跟进',
              is_internal: false,
              created_at: '2026-05-01T02:00:00Z',
              updated_at: '2026-05-01T02:00:00Z',
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
            data: [
              {
                id: 1,
                ticket_id: 99,
                user_id: 1001,
                content: '初始回复',
                is_internal: false,
                created_at: '2026-05-01T01:00:00Z',
                updated_at: '2026-05-01T01:00:00Z',
              },
              {
                id: 2,
                ticket_id: 99,
                user_id: 1001,
                content: '我来跟进',
                is_internal: false,
                created_at: '2026-05-01T02:00:00Z',
                updated_at: '2026-05-01T02:00:00Z',
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/ticket/detail/99'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('需要帮助')).toBeInTheDocument()
    })
    expect(screen.getByText('初始回复')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('请输入回复内容'), {
      target: { value: '我来跟进' },
    })
    fireEvent.click(screen.getByRole('button', { name: '提交回复' }))

    await waitFor(() => {
      expect(screen.getByText('我来跟进')).toBeInTheDocument()
    })
  })
})
