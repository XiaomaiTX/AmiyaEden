import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('dashboard characters page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
    })
  })

  test('loads profile data and renders character controls', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              user: {
                id: 1,
                nickname: 'Amiya',
                qq: '123456',
                discord_id: 'amiya#0001',
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
                  scopes: 'esi-killmails.read_corporation_killmails.v1 esi-skills.read_skills.v1',
                  token_expiry: '2026-06-01T00:00:00Z',
                  token_invalid: false,
                  corporation_id: 1,
                  alliance_id: 1,
                },
                {
                  id: 2,
                  character_id: 1002,
                  character_name: 'Miya',
                  user_id: 1,
                  scopes: 'esi-skills.read_skills.v1',
                  token_expiry: '2026-06-01T00:00:00Z',
                  token_invalid: true,
                  corporation_id: 1,
                  alliance_id: 1,
                },
              ],
              roles: ['admin'],
              permissions: [],
              profile_complete: false,
              enforce_character_esi_restriction: true,
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
              show_card: true,
              needs_profile_qq: false,
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/characters'],
    })

    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByDisplayValue('Amiya')).toBeInTheDocument()
      expect(screen.getByText('Miya')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '补录推荐人' })).toBeInTheDocument()
    })
  })

  test('refreshes the session profile lock after saving complete contact details', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      profileComplete: false,
      primaryCharacterId: 1001,
      characters: [{ characterId: 1001, tokenInvalid: false }],
    })

    let meRequestCount = 0
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      if (url === '/api/v1/me' && init?.method === 'PUT') {
        return new Response(JSON.stringify({ code: 0, msg: 'ok', data: null }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      if (url === '/api/v1/me') {
        meRequestCount += 1
        const complete = meRequestCount > 1
        return new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              user: {
                id: 1,
                nickname: complete ? 'Amiya' : '',
                qq: complete ? '123456' : '',
                discord_id: '',
                status: 1,
                role: 'admin',
                primary_character_id: 1001,
                last_login_at: null,
                last_login_ip: '',
              },
              characters: [
                {
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
              corp_capabilities: [],
              permissions: [],
              enforce_character_esi_restriction: true,
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      }

      if (url.includes('/api/v1/newbro/direct-referral/status')) {
        return new Response(JSON.stringify({ code: 0, msg: 'ok', data: { show_card: false, needs_profile_qq: false } }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      throw new Error(`Unexpected request: ${url}`)
    })

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/dashboard/characters'],
    })
    render(<RouterProvider router={router} />)

    await screen.findByRole('button', { name: '保存资料' })
    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[0], { target: { value: 'Amiya' } })
    fireEvent.change(inputs[1], { target: { value: '123456' } })
    fireEvent.click(screen.getByRole('button', { name: '保存资料' }))

    await waitFor(() => {
      expect(useSessionStore.getState().profileComplete).toBe(true)
    })
  })
})
