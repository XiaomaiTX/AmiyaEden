import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('auth callback page', () => {
  beforeEach(() => {
    useSessionStore.getState().clearSession()
    sessionStorage.clear()
  })

  test('hydrates session from callback token and /me', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            user: {
              id: 1,
              nickname: 'Amiya',
              qq: '',
              discord_id: '',
              status: 1,
              role: 'admin',
              primary_character_id: 1001,
              last_login_at: null,
              last_login_ip: '127.0.0.1',
            },
            characters: [
              {
                id: 1,
                character_id: 1001,
                character_name: 'Amiya',
                user_id: 1,
                scopes: '',
                token_expiry: '',
                token_invalid: false,
                corporation_id: 1,
                alliance_id: 1,
              },
            ],
            roles: ['admin'],
            permissions: [],
            profile_complete: true,
            enforce_character_esi_restriction: false,
          },
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/callback?token=test-token'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('登录成功')).toBeInTheDocument()
    })

    const state = useSessionStore.getState()
    expect(state.accessToken).toBe('test-token')
    expect(state.isLoggedIn).toBe(true)
    expect(state.characterId).toBe(1001)
    expect(state.roles).toEqual(['admin'])
  })

  test('shows error when token is missing', async () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/callback'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('登录失败')).toBeInTheDocument()
    })
  })

  test('shows callback error from query params', async () => {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/callback?error=access_denied&error_description=user_cancelled'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('登录失败')).toBeInTheDocument()
      expect(screen.getByText('user_cancelled')).toBeInTheDocument()
    })
  })

  test('clears session when /me verification fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 1001, msg: 'invalid token', data: null }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/callback?token=bad-token'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('登录失败')).toBeInTheDocument()
    })

    const state = useSessionStore.getState()
    expect(state.isLoggedIn).toBe(false)
    expect(state.accessToken).toBeNull()
  })

  test('uses remembered redirect from sessionStorage when query redirect is missing', async () => {
    sessionStorage.setItem('auth:sso:redirect', '/info/wallet')
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            user: {
              id: 1,
              nickname: 'Amiya',
              qq: '',
              discord_id: '',
              status: 1,
              role: 'admin',
              primary_character_id: 1001,
              last_login_at: null,
              last_login_ip: '127.0.0.1',
            },
            characters: [],
            roles: ['admin'],
            permissions: [],
            profile_complete: true,
            enforce_character_esi_restriction: false,
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/callback?token=test-token'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('登录成功')).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/info/wallet')
    })
    expect(sessionStorage.getItem('auth:sso:redirect')).toBeNull()
  })
})
