import { render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

function jsonResponse(data: unknown) {
  return new Response(
    JSON.stringify({
      code: 0,
      msg: 'ok',
      data,
    }),
    {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )
}

function resetSession() {
  useSessionStore.setState({
    isLoggedIn: true,
    accessToken: 'token-123',
    characterId: 1001,
    characterName: 'Amiya',
    roles: ['super_admin'],
    authList: [],
    isCurrentlyNewbro: false,
    isMentorMenteeEligible: false,
    hydratedAt: null,
  })
}

describe('system migration pages', () => {
  beforeEach(() => {
    resetSession()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('system user page loads the restriction card and user table', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)

      if (url.includes('/api/v1/system/role/definitions')) {
        return jsonResponse([])
      }

      if (url.includes('/api/v1/system/basic-config/character-esi-restriction')) {
        return jsonResponse({ enforce_character_esi_restriction: true })
      }

      if (url.includes('/api/v1/system/user') && !url.includes('/roles')) {
        return jsonResponse({
          list: [
            {
              id: 1,
              nickname: 'Amiya',
              qq: '123456',
              discord_id: 'discord-1',
              primary_character_id: 1001,
              status: 1,
              roles: ['super_admin'],
              characters: [],
              last_login_at: '2026-05-01T00:00:00Z',
              last_login_ip: '127.0.0.1',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      }

      throw new Error(`Unexpected request: ${url}`)
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/system/user'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('用户管理')
    await waitFor(() => {
      expect(screen.getByText('Character ESI Restriction')).toBeInTheDocument()
      expect(screen.getByText('Amiya')).toBeInTheDocument()
    })
  })

  test('system task manager page loads the task list', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)

      if (url.includes('/api/v1/tasks/esi/tasks')) {
        return jsonResponse([
          {
            name: 'esi-refresh-character',
            description: 'Character Refresh',
            priority: 50,
            active_interval: '0 */5 * * * *',
            inactive_interval: '0 0 * * * *',
            required_scopes: [],
          },
        ])
      }

      if (url.includes('/api/v1/tasks')) {
        return jsonResponse([
          {
            name: 'esi-refresh-character',
            description: 'Character Refresh',
            category: 'esi',
            type: 'recurring',
            runnable: true,
            cron_expr: '0 */5 * * * *',
            default_cron: '0 */5 * * * *',
            last_execution: null,
          },
        ])
      }

      return jsonResponse({ list: [], total: 0, page: 1, pageSize: 20 })
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/system/task-manager'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('任务管理')
    await waitFor(() => {
      expect(screen.getByText('Character Refresh')).toBeInTheDocument()
    })
  })

  test('system wallet page loads the wallet list tab', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)

      if (url.includes('/api/v1/system/wallet/list')) {
        return jsonResponse({
          list: [
            {
              id: 1,
              user_id: 1001,
              balance: 123456,
              updated_at: '2026-05-01T00:00:00Z',
              character_name: 'Amiya',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      }

      return jsonResponse({ list: [], total: 0, page: 1, pageSize: 20 })
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/system/wallet'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('钱包管理')
    await waitFor(() => {
      expect(screen.getByText('Amiya')).toBeInTheDocument()
      expect(screen.getByText('123,456')).toBeInTheDocument()
    })
  })
})
