import { beforeEach, describe, expect, test } from 'vitest'
import {
  PREFERENCE_STORE_KEY,
  SESSION_STORE_KEY,
} from '@/stores/persistence-keys'
import { usePreferenceStore } from '@/stores/preference-store'
import { useSessionStore } from '@/stores/session-store'

describe('store boundaries', () => {
  beforeEach(() => {
    localStorage.removeItem(PREFERENCE_STORE_KEY)
    localStorage.removeItem(SESSION_STORE_KEY)
    usePreferenceStore.setState({ locale: 'zh-CN', sidebarCollapsed: false, theme: 'system' })
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

  test('preference store updates locale and layout preference', () => {
    usePreferenceStore.getState().setLocale('en-US')
    usePreferenceStore.getState().toggleSidebar()
    usePreferenceStore.getState().setTheme('dark')

    const state = usePreferenceStore.getState()

    expect(state.locale).toBe('en-US')
    expect(state.sidebarCollapsed).toBe(true)
    expect(state.theme).toBe('dark')
  })

  test('session store updates and clears auth snapshot', () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: ['route:dashboard:view'],
    })

    let state = useSessionStore.getState()
    expect(state.isLoggedIn).toBe(true)
    expect(state.characterId).toBe(1001)
    expect(state.roles).toEqual(['admin'])

    useSessionStore.getState().clearSession()
    state = useSessionStore.getState()
    expect(state.isLoggedIn).toBe(false)
    expect(state.authList).toEqual([])
  })
})

