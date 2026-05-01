import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info esi check page', () => {
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

  test('loads scopes and characters and allows character switching', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: [
              { module: 'wallet', scope: 'esi-wallet.read_character_wallet.v1', description: 'Wallet', required: true },
              { module: 'skills', scope: 'esi-skills.read_skills.v1', description: 'Skills', required: false },
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
            data: [
              {
                id: 1,
                character_id: 1001,
                character_name: 'Amiya',
                user_id: 1,
                scopes: 'esi-wallet.read_character_wallet.v1 esi-skills.read_skills.v1',
                token_expiry: '2026-12-31T00:00:00Z',
                token_invalid: false,
                corporation_id: 1,
                alliance_id: 0,
              },
              {
                id: 2,
                character_id: 1002,
                character_name: 'Beta',
                user_id: 1,
                scopes: 'esi-skills.read_skills.v1',
                token_expiry: '2026-12-31T00:00:00Z',
                token_invalid: false,
                corporation_id: 1,
                alliance_id: 0,
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/esi-check'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('授权总览')).toBeInTheDocument()
    })
    expect(screen.getByText('共 2 个人物')).toBeInTheDocument()
    expect(screen.getByText('1 个人物授权异常')).toBeInTheDocument()
    expect(screen.getByText('esi-wallet.read_character_wallet.v1')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Beta' }))

    await waitFor(() => {
      expect(screen.getByText('Beta')).toBeInTheDocument()
    })
    expect(screen.getByText('0/1 已授权')).toBeInTheDocument()
  })
})
