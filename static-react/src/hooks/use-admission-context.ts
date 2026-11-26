import { useEffect } from 'react'
import { create } from 'zustand'
import { getAdmissionContext } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { useSessionStore } from '@/stores'
import type { AdmissionContext } from '@/api/generated/client'

interface AdmissionContextState {
  key: string
  generation: number
  context: AdmissionContext | null
  loading: boolean
  error: boolean
  load: (key: string, force?: boolean) => Promise<void>
}
const useAdmissionState = create<AdmissionContextState>((set, get) => ({
  key: '',
  generation: 0,
  context: null,
  loading: false,
  error: false,
  load: async (key, force = false) => {
    if (!force && get().key === key) return
    const generation = get().generation + 1
    set({ key, generation, context: null, loading: true, error: false })
    if (!key) {
      set({ loading: false })
      return
    }
    try {
      const context = await unwrapContract(getAdmissionContext(), 'admissions.errors.internal')
      if (
        !context ||
        !Array.isArray(context.characters) ||
        !Array.isArray(context.targets) ||
        !Array.isArray(context.reviewable_corporation_ids)
      ) {
        throw new Error('admissions.errors.internal')
      }
      if (get().generation === generation) set({ context, loading: false })
    } catch {
      if (get().generation === generation) set({ context: null, loading: false, error: true })
    }
  },
}))

export function useAdmissionContext() {
  const key = useSessionStore((state) =>
    state.isLoggedIn ? `${state.accessToken}:${state.hydratedAt}` : ''
  )
  const state = useAdmissionState()
  const load = state.load
  useEffect(() => {
    void load(key)
  }, [key, load])
  return {
    context: key && state.key === key ? state.context : null,
    loading: Boolean(key) && (state.key !== key || state.loading),
    error: state.key === key && state.error,
    reload: () => state.load(key, true),
  }
}
