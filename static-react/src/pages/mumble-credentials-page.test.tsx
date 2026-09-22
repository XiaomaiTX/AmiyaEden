import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

function mockMeResponse() {
  return {
    user: {
      id: 1,
      nickname: 'Amiya',
      qq: '123456',
      discord_id: 'amiya#0001',
      status: 1,
      role: 'user',
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
        scopes: 'esi-skills.read_skills.v1',
        token_expiry: '2026-06-01T00:00:00Z',
        token_invalid: false,
        corporation_id: 1,
        alliance_id: 1,
      },
    ],
    roles: ['user'],
    permissions: [],
    profile_complete: true,
    enforce_character_esi_restriction: true,
  }
}

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify({ code: 0, msg: 'ok', data }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('mumble credentials page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['user'],
    })
  })

  test('renders multi-node server list with copy buttons', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url === '/api/v1/me') {
        return jsonResponse(mockMeResponse())
      }
      if (url === '/api/v1/mumble/credential') {
        return jsonResponse({
          created: true,
          enabled: true,
          servers: [
            { name: '香港-1', description: '香港节点', address: 'hk.mumble.example.com', port: 64738 },
            { name: '法兰克福', description: '', address: 'fra.mumble.example.com', port: 64739 },
          ],
        })
      }
      throw new Error(`Unexpected request: ${url}`)
    })

    const router = createMemoryRouter(appRoutes, { initialEntries: ['/mumble'] })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Mumble 语音凭据' })).toBeInTheDocument()
      expect(screen.getAllByText('Amiya').length).toBeGreaterThan(0)
      expect(screen.getByText('香港-1')).toBeInTheDocument()
      expect(screen.getByText('香港节点')).toBeInTheDocument()
      expect(screen.getByText('法兰克福')).toBeInTheDocument()
      expect(screen.getByText('hk.mumble.example.com:64738')).toBeInTheDocument()
      expect(screen.getByText('fra.mumble.example.com:64739')).toBeInTheDocument()
      expect(screen.getAllByRole('button', { name: '复制' }).length).toBe(3)
    })
    expect(screen.queryByText('稳定用户 ID')).not.toBeInTheDocument()
  })

  test('shows placeholder when no server nodes are configured', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url === '/api/v1/me') {
        return jsonResponse(mockMeResponse())
      }
      if (url === '/api/v1/mumble/credential') {
        return jsonResponse({ created: false, enabled: false })
      }
      throw new Error(`Unexpected request: ${url}`)
    })

    const router = createMemoryRouter(appRoutes, { initialEntries: ['/mumble'] })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Mumble 语音凭据' })).toBeInTheDocument()
    })
    expect(screen.getByText('管理员尚未配置任何服务器节点')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '创建语音密码' })).toBeInTheDocument()
  })
})
