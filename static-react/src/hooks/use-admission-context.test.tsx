import { act, renderHook, waitFor } from '@testing-library/react'
import { getAdmissionContext } from '@/api/admissions'
import { useSessionStore } from '@/stores'
import { useAdmissionContext } from './use-admission-context'
import type { AdmissionContext } from '@/types/api/admissions'

vi.unmock('@/hooks/use-admission-context')
vi.mock('@/api/admissions', () => ({ getAdmissionContext: vi.fn() }))

const context: AdmissionContext = {
  is_member: true,
  can_create_new_member: false,
  can_create_alt: true,
  qq: '',
  discord_id: 'discord',
  characters: [],
  targets: [],
  reviewable_corporation_ids: [98185110],
}

afterEach(() => {
  act(() => useSessionStore.getState().clearSession())
  vi.resetAllMocks()
})

it('keeps eligibility closed on failure, retries, refreshes membership, and clears on logout', async () => {
  vi.mocked(getAdmissionContext)
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValueOnce(context)
  act(() =>
    useSessionStore.setState({ isLoggedIn: true, accessToken: 'context-retry', hydratedAt: '1' })
  )
  const { result } = renderHook(() => useAdmissionContext())
  expect(result.current.context).toBeNull()
  await waitFor(() => expect(result.current.error).toBe(true))
  await act(() => result.current.reload())
  expect(result.current.context?.can_create_alt).toBe(true)
  vi.mocked(getAdmissionContext).mockResolvedValueOnce({
    ...context,
    is_member: false,
    can_create_alt: false,
    can_create_new_member: true,
  })
  act(() => useSessionStore.setState({ hydratedAt: '2' }))
  await waitFor(() => expect(result.current.context?.can_create_new_member).toBe(true))
  act(() => useSessionStore.getState().clearSession())
  expect(result.current.context).toBeNull()
})

it('does not let an older session response restore permissions after logout', async () => {
  let finish!: (value: AdmissionContext) => void
  vi.mocked(getAdmissionContext).mockReturnValue(
    new Promise((resolve) => {
      finish = resolve
    })
  )
  act(() =>
    useSessionStore.setState({ isLoggedIn: true, accessToken: 'context-stale', hydratedAt: '3' })
  )
  const { result } = renderHook(() => useAdmissionContext())
  await waitFor(() => expect(getAdmissionContext).toHaveBeenCalled())
  act(() => useSessionStore.getState().clearSession())
  await act(async () => {
    finish(context)
  })
  expect(result.current.context).toBeNull()
  expect(result.current.loading).toBe(false)
})
