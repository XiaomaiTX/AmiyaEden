import { fetchBadgeCounts } from '@/api/badge'
import { useBadgeStore } from '@/stores/badge-store'

vi.mock('@/api/badge', () => ({
  fetchBadgeCounts: vi.fn(),
}))

describe('badge store', () => {
  beforeEach(() => {
    useBadgeStore.getState().clear()
    vi.mocked(fetchBadgeCounts).mockReset()
  })

  test('loads badge counts once for the active character', async () => {
    vi.mocked(fetchBadgeCounts).mockResolvedValue({ order_pending: 2 })

    await Promise.all([
      useBadgeStore.getState().load(1001),
      useBadgeStore.getState().load(1001),
    ])
    await useBadgeStore.getState().load(1001)

    expect(fetchBadgeCounts).toHaveBeenCalledTimes(1)
    expect(useBadgeStore.getState().counts).toEqual({ order_pending: 2 })
  })

  test('discards a response after the session badge state is cleared', async () => {
    let resolveRequest: (counts: { ticket_attention: number }) => void = () => undefined
    vi.mocked(fetchBadgeCounts).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      })
    )

    const request = useBadgeStore.getState().load(1001)
    useBadgeStore.getState().clear()
    resolveRequest({ ticket_attention: 3 })
    await request

    expect(useBadgeStore.getState().counts).toEqual({})
    expect(useBadgeStore.getState().loadedForCharacterId).toBeNull()
  })
})
