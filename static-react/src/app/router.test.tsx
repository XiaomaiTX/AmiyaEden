import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { act } from '@testing-library/react'
import { appRoutes } from '@/app/router'
import { dispatchUnauthorized } from '@/auth'
import { useSessionStore } from '@/stores'

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    fetchMyCharacters: vi.fn().mockResolvedValue([]),
  }
})

vi.mock('@/api/fleet', async () => {
  const actual = await vi.importActual<typeof import('@/api/fleet')>('@/api/fleet')
  return {
    ...actual,
    fetchFleetList: vi.fn().mockResolvedValue({ list: [], total: 0, page: 1, pageSize: 20 }),
    fetchMyPapLogs: vi.fn().mockResolvedValue([]),
    fetchFleetInvites: vi.fn().mockResolvedValue([]),
    fetchMembersWithPap: vi.fn().mockResolvedValue({
      list: [],
      total: 0,
      page: 1,
      pageSize: 20,
    }),
    fetchCorporationPapSummary: vi.fn().mockResolvedValue({
      list: [],
      total: 0,
      page: 1,
      pageSize: 20,
      overview: {
        filtered_pap_total: 0,
        filtered_strat_op_total: 0,
        all_pap_total: 0,
        filtered_user_count: 0,
        period: 'last_month',
      },
    }),
  }
})

vi.mock('@/api/fleet-config', async () => {
  const actual = await vi.importActual<typeof import('@/api/fleet-config')>('@/api/fleet-config')
  return {
    ...actual,
    fetchFleetConfigList: vi.fn().mockResolvedValue({ list: [], total: 0, page: 1, pageSize: 20 }),
  }
})

vi.mock('@/api/tool-bookmark', async () => {
  const actual = await vi.importActual<typeof import('@/api/tool-bookmark')>('@/api/tool-bookmark')
  return {
    ...actual,
    fetchVisibleToolBookmarks: vi.fn().mockResolvedValue([]),
    fetchAdminToolBookmarks: vi.fn().mockResolvedValue([]),
  }
})

describe('router auth and route meta access flow', () => {
  beforeEach(() => {
    useSessionStore.setState({
      isLoggedIn: false,
      accessToken: null,
      characterId: null,
      characterName: null,
      roles: [],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
      hydratedAt: null,
    })
  })

  test('redirects to auth login when visiting protected route without session', () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)

    expect(screen.getByRole('heading', { name: 'EVE SSO 登录' })).toBeInTheDocument()
  })

  test('redirects to auth login when visiting tool bookmarks without a session', async () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/tool-bookmarks'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'EVE SSO 登录' })).toBeInTheDocument()
    })
  })

  test('renders tool bookmarks when the user has the menu.info capability', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['user'],
      corpCapabilities: ['menu.info'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/tool-bookmarks'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '工具书签' })).toBeInTheDocument()
    })
  })

  test('redirects /login to /auth/login', async () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/login'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'EVE SSO 登录' })).toBeInTheDocument()
    })
  })

  test('redirects the authenticated root route to the dashboard console', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      corpCapabilities: ['menu.dashboard'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/dashboard/console')
    })
  })

  test('redirects to 403 when role does not match route meta roles', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['guest'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/system/basic-config'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('403 Forbidden')).toBeInTheDocument()
    })
  })

  test('applies batch A role gate on /dashboard/npc-kills', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['member'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/npc-kills'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('403 Forbidden')).toBeInTheDocument()
    })
  })

  test('navigates to auth login when unauthorized event is dispatched', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)
    act(() => {
      dispatchUnauthorized({ reason: 'manual' })
    })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'EVE SSO 登录' })).toBeInTheDocument()
    })
  })

  test('renders 404 for unknown public route', () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/unknown-route'],
    })

    render(<RouterProvider router={router} />)

    expect(screen.getByText('404 Not Found')).toBeInTheDocument()
  })

  test('renders operation fleets page', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      corpCapabilities: ['menu.operation'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/operation/fleets'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getAllByText('舰队管理').length).toBeGreaterThan(0)
    })
  })

  test('applies requiresNewbro constraint', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['member'],
      corpCapabilities: [],
      isCurrentlyNewbro: false,
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/newbro/select-captain'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('403 Forbidden')).toBeInTheDocument()
    })
  })

  test('redirects locked profiles to the JWT characters route with localized reasons', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['user'],
      corpCapabilities: ['menu.info'],
      profileComplete: false,
      primaryCharacterId: 1001,
      characters: [{ characterId: 1001, tokenInvalid: false }],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/tool-bookmarks'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/characters')
      expect(router.state.location.state).toEqual({
        profileLockReasons: ['profile_incomplete'],
      })
    })
  })

  test('allows a completed profile to access a protected route', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['user'],
      corpCapabilities: ['menu.info'],
      profileComplete: true,
      primaryCharacterId: 1001,
      characters: [{ characterId: 1001, tokenInvalid: false }],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/tool-bookmarks'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/info/tool-bookmarks')
    })
  })

  describe('tail batch routes', () => {
    test('renders recruit landing page at /r/:code', () => {
      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/r/test-code'],
      })

      render(<RouterProvider router={router} />)

      expect(screen.getByRole('heading', { name: '加入我们' })).toBeInTheDocument()
      expect(screen.getByRole('textbox')).toBeInTheDocument()
    })

    test('renders iframe page at /outside/iframe/*', () => {
      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/outside/iframe/https://example.com'],
      })

      render(<RouterProvider router={router} />)

      const iframe = screen.getByTitle('External Content')
      expect(iframe).toBeInTheDocument()
      expect(iframe).toHaveAttribute('src', 'https://example.com/')
    })

    test('renders iframe page with no-src message when path is empty', () => {
      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/outside/iframe//'],
      })

      render(<RouterProvider router={router} />)

      expect(screen.getByText('Missing iframe target path.')).toBeInTheDocument()
    })

    test('renders auth callback page at /auth/callback', () => {
      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/auth/callback'],
      })

      render(<RouterProvider router={router} />)

      expect(screen.getByText('登录失败')).toBeInTheDocument()
      expect(screen.getByText('未收到登录令牌，请重新登录。')).toBeInTheDocument()
    })

    test('renders 404 for unknown route', () => {
      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/nonexistent-path'],
      })

      render(<RouterProvider router={router} />)

      expect(screen.getByText('404 Not Found')).toBeInTheDocument()
    })

    test('removed hall of fame routes render 404', () => {
      useSessionStore.getState().setSessionSnapshot({
        isLoggedIn: true,
        accessToken: 'token-123',
        characterId: 1001,
        characterName: 'Amiya',
        roles: ['admin'],
      })

      const router = createMemoryRouter(appRoutes, {
        initialEntries: ['/hall-of-fame/manage'],
      })

      render(<RouterProvider router={router} />)

      expect(screen.getByText('404 Not Found')).toBeInTheDocument()
    })
  })
})
