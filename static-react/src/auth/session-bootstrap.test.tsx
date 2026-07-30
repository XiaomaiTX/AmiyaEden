import { StrictMode } from 'react'
import { render, waitFor } from '@testing-library/react'
import { fetchGetUserInfo } from '@/api/auth'
import { SessionBootstrap } from '@/auth/session-bootstrap'
import { useSessionStore } from '@/stores'

vi.mock('@/api/auth', () => ({
  fetchGetUserInfo: vi.fn(),
}))

describe('SessionBootstrap', () => {
  test('refreshes a persisted token once under StrictMode before routes continue', async () => {
    vi.mocked(fetchGetUserInfo).mockResolvedValue({
      roles: ['user'],
      corpCapabilities: ['menu.dashboard'],
      userId: 1,
      userName: 'Amiya',
      nickname: 'Doctor',
      qq: '12345',
      discordId: '',
      profileComplete: true,
      enforceCharacterESIRestriction: true,
      isCurrentlyNewbro: undefined,
      isMentorMenteeEligible: undefined,
      characters: [
        {
          character_id: 1001,
          character_name: 'Amiya',
          user_id: 1,
          scopes: '',
          token_expiry: '',
          token_invalid: false,
          corporation_id: 2,
          alliance_id: 3,
        },
      ],
      primaryCharacterId: 1001,
    })
    useSessionStore.setState({
      isLoggedIn: true,
      accessToken: 'persisted-token',
      bootstrapRequired: true,
    })

    render(
      <StrictMode>
        <SessionBootstrap />
      </StrictMode>
    )

    await waitFor(() => {
      expect(useSessionStore.getState().bootstrapRequired).toBe(false)
      expect(useSessionStore.getState().characterName).toBe('Amiya')
    })
    expect(fetchGetUserInfo).toHaveBeenCalledOnce()
  })
})
