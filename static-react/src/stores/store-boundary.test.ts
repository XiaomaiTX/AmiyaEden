import { beforeEach, describe, expect, test } from 'vitest'
import {
  PREFERENCE_STORE_KEY,
  SESSION_STORE_KEY,
  SIDEBAR_NAVIGATION_STORE_KEY,
} from '@/stores/persistence-keys'
import { usePreferenceStore } from '@/stores/preference-store'
import { useSessionStore } from '@/stores/session-store'
import { useSidebarNavigationStore } from '@/stores/sidebar-navigation-store'

describe('store boundaries', () => {
  beforeEach(() => {
    localStorage.removeItem(PREFERENCE_STORE_KEY)
    localStorage.removeItem(SESSION_STORE_KEY)
    localStorage.removeItem(SIDEBAR_NAVIGATION_STORE_KEY)
    usePreferenceStore.setState({ locale: 'zh-CN', sidebarCollapsed: false, theme: 'system' })
    useSidebarNavigationStore.getState().clear()
    useSessionStore.setState({
      isLoggedIn: false,
      accessToken: null,
      characterId: null,
      characterName: null,
      roles: [],
      corpCapabilities: [],
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
    })

    let state = useSessionStore.getState()
    expect(state.isLoggedIn).toBe(true)
    expect(state.characterId).toBe(1001)
    expect(state.roles).toEqual(['admin'])

    useSidebarNavigationStore.getState().resetForCharacter(1001)
    useSidebarNavigationStore.getState().expandMenuGroup('nav.group.info')
    useSessionStore.getState().clearSession()
    state = useSessionStore.getState()
    expect(state.isLoggedIn).toBe(false)
    expect(useSidebarNavigationStore.getState()).toMatchObject({
      ownerCharacterId: null,
      expandedMenuGroupKeys: [],
    })
  })
})

