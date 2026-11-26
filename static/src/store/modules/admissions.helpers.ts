import { canAccessAdmission } from '../../utils/admissions'
import type { AdmissionContext } from '../../api/generated/client'
import type { AppRouteRecord } from '../../types/router'

export interface AdmissionLoaderTarget {
  setContext: (value: AdmissionContext | null) => void
  setLoading: (value: boolean) => void
  setLoaded: (value: boolean) => void
  setError: (value: boolean) => void
  onSettled: (context: AdmissionContext | null) => void
}

export function createAdmissionLoader(
  fetchContext: () => Promise<AdmissionContext>,
  target: AdmissionLoaderTarget
) {
  let context: AdmissionContext | null = null
  let loaded = false
  let generation = 0
  let pending: Promise<void> | null = null

  function refresh(): Promise<void> {
    if (pending) return pending
    const current = ++generation
    target.setLoading(true)
    target.setError(false)
    // A refresh is not a loss of permission. Keep the current menu and form mounted.
    pending = (async () => {
      try {
        const value = await fetchContext()
        if (generation === current) {
          context = value
          loaded = true
          target.setContext(value)
          target.setLoaded(true)
        }
      } catch {
        if (generation === current) {
          context = null
          loaded = false
          target.setContext(null)
          target.setLoaded(false)
          target.setError(true)
        }
      } finally {
        if (generation === current) {
          target.setLoading(false)
          pending = null
          target.onSettled(context)
        }
      }
    })()
    return pending
  }

  function ensureLoaded(): Promise<void> {
    if (pending) return pending
    if (loaded) return Promise.resolve()
    return refresh()
  }

  function clear() {
    generation++
    pending = null
    context = null
    loaded = false
    target.setContext(null)
    target.setLoading(false)
    target.setLoaded(false)
    target.setError(false)
    target.onSettled(null)
  }

  return { refresh, ensureLoaded, clear }
}

export function admissionMenuVisibility(
  menu: AppRouteRecord[],
  context: AdmissionContext | null
): AppRouteRecord[] {
  let changed = false
  const result = menu.map((item) => {
    const children = item.children ? admissionMenuVisibility(item.children, context) : undefined
    const isHide = item.meta.admission
      ? !canAccessAdmission(context, item.meta.admission)
      : item.meta.isHide
    if (isHide === item.meta.isHide && children === item.children) return item
    changed = true
    return { ...item, meta: { ...item.meta, isHide }, ...(children ? { children } : {}) }
  })
  return changed ? result : menu
}
