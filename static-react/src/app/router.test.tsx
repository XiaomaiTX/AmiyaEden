import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
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

describe('router auth and route meta access flow', () => {
  beforeEach(() => {
    useSessionStore.setState({
      isLoggedIn: false,
      accessToken: null,
      characterId: null,
      characterName: null,
      roles: [],
      authList: [],
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

  test('redirects /login to /auth/login', async () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/login'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'EVE SSO 登录' })).toBeInTheDocument()
    })
  })

  test('renders home page when session is logged in', () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: ['route:dashboard:view'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)

    expect(screen.getByText('AmiyaEden React Shell')).toBeInTheDocument()
  })

  test('redirects to 403 when role does not match route meta roles', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['guest'],
      authList: [],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/admin-demo'],
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
      authList: [],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/npc-kills'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('403 Forbidden')).toBeInTheDocument()
    })
  })

  test('consumes route authList meta on admin route', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: [],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/admin-demo'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Admin Route Demo')).toBeInTheDocument()
    })

    expect(useSessionStore.getState().authList).toEqual(['approve_order', 'edit_exchange_rate'])
  })

  test('navigates to auth login when unauthorized event is dispatched', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: ['route:dashboard:view'],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)
    dispatchUnauthorized({ reason: 'manual' })

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
      authList: [],
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/operation/fleets'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('舰队管理')).toBeInTheDocument()
    })
  })

  test('applies requiresNewbro constraint', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['member'],
      authList: [],
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
})

