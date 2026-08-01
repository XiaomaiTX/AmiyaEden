import { beforeEach, describe, expect, test } from 'vitest'
import { SIDEBAR_NAVIGATION_STORE_KEY } from '@/stores/persistence-keys'
import { useSidebarNavigationStore } from '@/stores/sidebar-navigation-store'

describe('sidebar navigation store', () => {
  beforeEach(() => {
    localStorage.removeItem(SIDEBAR_NAVIGATION_STORE_KEY)
    useSidebarNavigationStore.getState().clear()
  })

  test('expands and collapses menu groups idempotently', () => {
    const store = useSidebarNavigationStore.getState()

    store.expandMenuGroup('nav.group.info')
    store.expandMenuGroup('nav.group.info')
    expect(useSidebarNavigationStore.getState().expandedMenuGroupKeys).toEqual(['nav.group.info'])

    store.collapseMenuGroup('nav.group.info')
    store.collapseMenuGroup('nav.group.info')
    expect(useSidebarNavigationStore.getState().expandedMenuGroupKeys).toEqual([])
  })

  test('preserves groups for the same character and clears them for another character', () => {
    const store = useSidebarNavigationStore.getState()

    store.resetForCharacter(1)
    store.expandMenuGroup('nav.group.info')
    store.resetForCharacter(1)
    expect(useSidebarNavigationStore.getState().expandedMenuGroupKeys).toEqual(['nav.group.info'])

    store.resetForCharacter(2)
    expect(useSidebarNavigationStore.getState()).toMatchObject({
      ownerCharacterId: 2,
      expandedMenuGroupKeys: [],
    })
  })

  test('clears character-owned navigation state', () => {
    const store = useSidebarNavigationStore.getState()

    store.resetForCharacter(1)
    store.expandMenuGroup('nav.group.info')
    store.clear()

    expect(useSidebarNavigationStore.getState()).toMatchObject({
      ownerCharacterId: null,
      expandedMenuGroupKeys: [],
    })
  })
})
