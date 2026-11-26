import userEvent from '@testing-library/user-event'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'
function jsonResponse(data: unknown) {
  return new Response(
    JSON.stringify({
      code: 200,
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

    await screen.findAllByText('用户管理')
    await waitFor(() => {
      expect(screen.getByText('人物 ESI 限制')).toBeInTheDocument()
      expect(screen.getAllByText('Amiya').length).toBeGreaterThan(0)
    })
  })

  test('ordinary task uses its localized confirmation and runs only after confirmation', async () => {
    const user = userEvent.setup()
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const runTask = vi.fn(async () => jsonResponse(null))
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
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

      if (url === '/api/v1/tasks/esi-refresh-character/run' && init?.method === 'POST') {
        return runTask()
      }

      if (url === '/api/v1/tasks') {
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

    await screen.findAllByText('任务管理')
    expect(await screen.findByText('Character Refresh')).toBeInTheDocument()
    const button = await screen.findByRole('button', { name: /^执行$/ })
    await user.click(button)
    expect(confirm).toHaveBeenCalledWith('确定要立即执行任务「esi-refresh-character」吗？')
    expect(runTask).not.toHaveBeenCalled()
    confirm.mockReturnValue(true)
    await user.click(button)
    expect(runTask).toHaveBeenCalledTimes(1)
  })

  test('ESI rows run directly, while both all-character actions require confirmation', async () => {
    useSessionStore.setState({ roles: ['admin'], corpCapabilities: ['system.task.read'] })
    const user = userEvent.setup()
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    let resolveCharacter!: (response: Response) => void
    const characterRequest = new Promise<Response>((resolve) => {
      resolveCharacter = resolve
    })
    let resolveByName!: (response: Response) => void
    const byNameRequest = new Promise<Response>((resolve) => {
      resolveByName = resolve
    })
    const runCharacter = vi
      .fn<(body: BodyInit | null | undefined) => Promise<Response>>()
      .mockReturnValue(characterRequest)
    const runByName = vi
      .fn<(body: BodyInit | null | undefined) => Promise<Response>>()
      .mockReturnValue(byNameRequest)
    const runAll = vi.fn(async () => jsonResponse(null))

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const path = new URL(String(input), window.location.href).pathname
      switch (path) {
        case '/api/v1/tasks/esi/tasks':
          return jsonResponse([{ name: 'refresh-pilot' }])
        case '/api/v1/tasks/esi/statuses':
          return jsonResponse({
            list: [1001, 1002].map((character_id) => ({
              task_name: 'refresh-pilot',
              character_id,
              character_name: `Pilot ${character_id}`,
              priority: 50,
              status: 'pending',
            })),
            total: 2,
            page: 1,
            pageSize: 20,
          })
        case '/api/v1/tasks/esi/run':
          return runCharacter(init?.body)
        case '/api/v1/tasks/esi/run-task':
          return runByName(init?.body)
        case '/api/v1/tasks/esi/run-all':
          return runAll()
        default:
          return jsonResponse([])
      }
    })

    const router = createMemoryRouter(appRoutes, { initialEntries: ['/system/task-manager'] })
    render(<RouterProvider router={router} />)
    await user.click(await screen.findByRole('button', { name: 'ESI 状态' }))

    const runButtonFor = async (pilot: string) => {
      const row = (await screen.findByText(pilot)).closest<HTMLElement>('[role="row"], tr')
      if (!row) throw new Error(`Missing ESI status row for ${pilot}`)
      return within(row).getByRole('button', { name: '执行' })
    }
    const rowRun = await runButtonFor('Pilot 1001')
    const otherRun = await runButtonFor('Pilot 1002')
    const byName = await screen.findByRole('button', { name: 'refresh-pilot' })
    const all = screen.getByRole('button', { name: '执行全部 ESI' })
    for (const button of [rowRun, byName, all]) expect(button).toBeDisabled()
    act(() => {
      useSessionStore.setState({ corpCapabilities: ['system.task.read', 'system.task.run'] })
    })

    await user.click(rowRun)
    expect(confirm).not.toHaveBeenCalled()
    expect(runCharacter).toHaveBeenCalledExactlyOnceWith(
      JSON.stringify({ task_name: 'refresh-pilot', character_id: 1001 })
    )
    expect(rowRun).toBeDisabled()
    expect(otherRun).toBeEnabled()
    resolveCharacter(Response.json({ msg: 'Character refresh failed' }, { status: 500 }))
    expect(await screen.findByText('Character refresh failed')).toBeInTheDocument()
    expect(rowRun).toBeEnabled()

    await user.click(byName)
    await user.click(all)
    expect(confirm).toHaveBeenNthCalledWith(
      1,
      '确定要为所有人物在后台执行任务「refresh-pilot」吗？'
    )
    expect(confirm).toHaveBeenNthCalledWith(2, '确定要为所有人物在后台执行全部 ESI 刷新任务吗？')
    expect(runByName).not.toHaveBeenCalled()
    expect(runAll).not.toHaveBeenCalled()

    confirm.mockReturnValue(true)
    await user.click(byName)
    expect(runByName).toHaveBeenCalledExactlyOnceWith(JSON.stringify({ task_name: 'refresh-pilot' }))
    expect(byName).toBeDisabled()
    expect(all).toBeDisabled()
    expect(rowRun).toBeEnabled()
    resolveByName(jsonResponse(null))
    await waitFor(() => expect(all).toBeEnabled())
    await user.click(all)
    expect(runAll).toHaveBeenCalledTimes(1)
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

    await screen.findAllByText('钱包管理')
    await waitFor(() => {
      expect(screen.getAllByText('Amiya').length).toBeGreaterThan(0)
      expect(screen.getByText('123,456')).toBeInTheDocument()
    })
  })

  test('impersonation swaps the session token before loading the target user', async () => {
    const fetchCalls: { url: string; authHeader: string | null }[] = []
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      const headers = new Headers(init?.headers)
      fetchCalls.push({ url, authHeader: headers.get('Authorization') })

      if (url.includes('/api/v1/system/role/definitions')) {
        return jsonResponse([])
      }

      if (url.includes('/api/v1/system/basic-config/character-esi-restriction')) {
        return jsonResponse({ enforce_character_esi_restriction: true })
      }

      if (url.includes('/api/v1/system/user/') && url.includes('/impersonate')) {
        return jsonResponse({ token: 'target-user-token' })
      }

      if (url === '/api/v1/me') {
        return jsonResponse({
          user: {
            id: 42,
            nickname: 'TargetUser',
            qq: '',
            discord_id: '',
            status: 1,
            role: 'user',
            primary_character_id: 2002,
            last_login_at: null,
            last_login_ip: '',
          },
          characters: [
            {
              character_id: 2002,
              character_name: 'TargetUser',
              user_id: 42,
              scopes: '',
              token_expiry: '',
              token_invalid: false,
              corporation_id: 2,
              alliance_id: 2,
            },
          ],
          roles: ['user'],
          corp_capabilities: ['menu.dashboard'],
          permissions: [],
          profile_complete: true,
          enforce_character_esi_restriction: false,
          is_currently_newbro: true,
          is_mentor_mentee_eligible: false,
        })
      }

      if (url.includes('/api/v1/system/user') && !url.includes('/roles')) {
        return jsonResponse({
          list: [
            {
              id: 42,
              nickname: 'TargetUser',
              qq: '',
              discord_id: '',
              primary_character_id: 2002,
              status: 1,
              roles: ['user'],
              characters: [],
              last_login_at: null,
              last_login_ip: '',
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

    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const assignSpy = vi.fn()
    vi.spyOn(window, 'location', 'get').mockReturnValue({
      ...window.location,
      assign: assignSpy,
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/system/user'],
    })

    render(<RouterProvider router={router} />)

    const impersonateButtons = await screen.findAllByText('模拟登录')
    await userEvent.click(impersonateButtons[0])

    await waitFor(() => {
      expect(assignSpy).toHaveBeenCalledWith('/')
    })

    confirmSpy.mockRestore()

    const meCalls = fetchCalls.filter((call) => call.url === '/api/v1/me')
    expect(meCalls.length).toBe(1)
    expect(meCalls[0].authHeader).toBe('Bearer target-user-token')

    const state = useSessionStore.getState()
    expect(state.accessToken).toBe('target-user-token')
    expect(state.roles).toEqual(['user'])
    expect(state.characterId).toBe(2002)
    expect(state.characterName).toBe('TargetUser')
    expect(state.corpCapabilities).toEqual(['menu.dashboard'])
    expect(state.isCurrentlyNewbro).toBe(true)
    expect(state.profileComplete).toBe(true)
    expect(state.primaryCharacterId).toBe(2002)
    expect(state.characters).toEqual([{ characterId: 2002, tokenInvalid: false }])
  })
})
