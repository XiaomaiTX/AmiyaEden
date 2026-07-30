import { useWorktabStore } from '@/stores/worktab-store'

describe('worktab store', () => {
  beforeEach(() => {
    useWorktabStore.getState().clear()
  })

  test('reuses a route id while preserving its latest URL state', () => {
    const store = useWorktabStore.getState()
    store.open({ routeId: 'tickets', path: '/tickets?page=1', titleKey: 'tickets' })
    store.open({ routeId: 'tickets', path: '/tickets?page=2', titleKey: 'tickets' })

    expect(useWorktabStore.getState().opened).toEqual([
      {
        routeId: 'tickets',
        path: '/tickets?page=2',
        titleKey: 'tickets',
        fixed: false,
      },
    ])
  })

  test('keeps fixed tabs during batch close and resets tabs for another character', () => {
    const store = useWorktabStore.getState()
    store.resetForCharacter(1)
    store.open({ routeId: 'a', path: '/a', titleKey: 'a' })
    store.open({ routeId: 'b', path: '/b', titleKey: 'b' })
    store.toggleFixed('/a')

    expect(store.closeAll()).toBe('/a')
    expect(useWorktabStore.getState().opened.map((tab) => tab.path)).toEqual(['/a'])

    useWorktabStore.getState().resetForCharacter(2)
    expect(useWorktabStore.getState().opened).toEqual([])
  })
})
