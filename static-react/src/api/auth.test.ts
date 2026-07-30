import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/api/http-client', () => ({
  requestJson: vi.fn(),
}))

import { requestJson } from '@/api/http-client'
import { fetchGetUserInfo } from '@/api/auth'

const baseMeResponse = {
  user: {
    id: 1,
    nickname: 'Amiya',
    qq: '123456',
    discord_id: '',
    status: 1,
    role: 'user',
    primary_character_id: 1001,
    last_login_at: null,
    last_login_ip: '',
  },
  characters: [],
  roles: ['user'],
  corp_capabilities: [],
  permissions: [],
  enforce_character_esi_restriction: true,
}

describe('auth api', () => {
  beforeEach(() => {
    vi.mocked(requestJson).mockReset()
  })

  test('derives a completed profile when a rolling backend response omits the flag', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 0,
      msg: 'ok',
      data: baseMeResponse,
    })

    const userInfo = await fetchGetUserInfo()

    expect(userInfo.profileComplete).toBe(true)
  })

  test('honors an explicit backend profile flag', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 0,
      msg: 'ok',
      data: { ...baseMeResponse, profile_complete: false },
    })

    const userInfo = await fetchGetUserInfo()

    expect(userInfo.profileComplete).toBe(false)
  })
})
