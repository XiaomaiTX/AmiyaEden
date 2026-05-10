import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { dispatchUnauthorized } from '@/auth'
import { useSessionStore } from '@/stores'

describe('router auth and route meta access flow', () => {
  beforeEach(() => {
    useSessionStore.setState({
      isLoggedIn: false,
      characterId: null,
      characterName: null,
      roles: [],
      authList: [],
      hydratedAt: null,
    })
  })

  test('redirects to login when visiting protected route without session', () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/'],
    })

    render(<RouterProvider router={router} />)

    expect(screen.getByText(/EVE SSO/)).toBeInTheDocument()
  })

  test('renders home page when session is logged in', () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
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

  test('consumes route authList meta on admin route', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
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

  test('navigates to login when unauthorized event is dispatched', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
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
      expect(screen.getByText(/EVE SSO/)).toBeInTheDocument()
    })
  })

  test('renders 404 for unknown public route', () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/unknown-route'],
    })

    render(<RouterProvider router={router} />)

    expect(screen.getByText('404 Not Found')).toBeInTheDocument()
  })
})

