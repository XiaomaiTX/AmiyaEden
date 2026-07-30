import { useSessionStore } from '@/stores'

/**
 * React capability predicates for button-level and component-level gating.
 *
 * Reads the corp capability set from the session store (populated from
 * `/me.corp_capabilities`) and exposes three predicates used by button-level
 * and component-level gating. The backend remains the final authorization
 * authority; this hook only prevents "click button → 403" UX surprises.
 *
 * - `hasCapability(key)` — true if the user holds a single capability.
 * - `hasAllCapabilities(keys)` — true if every capability is present (AND).
 * - `hasAnyCapability(keys)` — true if at least one capability is present (OR).
 *
 * `super_admin` short-circuits to true to mirror the backend RequireCorpCapability
 * middleware.
 */
export function useCorpCapability() {
  const roles = useSessionStore((state) => state.roles)
  const corpCapabilities = useSessionStore((state) => state.corpCapabilities)

  const isSuperAdmin = roles.includes('super_admin')

  const hasCapability = (key: string) => {
    if (isSuperAdmin) return true
    return corpCapabilities.includes(key)
  }

  const hasAllCapabilities = (keys: string[]) => {
    if (keys.length === 0) return true
    if (isSuperAdmin) return true
    return keys.every((key) => corpCapabilities.includes(key))
  }

  const hasAnyCapability = (keys: string[]) => {
    if (keys.length === 0) return true
    if (isSuperAdmin) return true
    return keys.some((key) => corpCapabilities.includes(key))
  }

  return { hasCapability, hasAllCapabilities, hasAnyCapability }
}
