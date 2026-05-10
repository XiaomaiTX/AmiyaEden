import { render, screen, waitFor } from '@testing-library/react'
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
      authList: [],
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
      expect(screen.getByRole('heading', { name: 'EVE 人物管理' })).toBeInTheDocument()
    })

    expect(screen.getByDisplayValue('Amiya')).toBeInTheDocument()
    expect(screen.getByText('补录推荐人')).toBeInTheDocument()
    expect(screen.getByText('绑定新人物')).toBeInTheDocument()
    expect(screen.getByText('主人物')).toBeInTheDocument()
  })
})
