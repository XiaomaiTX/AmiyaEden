export const GALAXY_REGISTRY_TIMEOUT_MS = 2 * 60 * 60 * 1000

export const GALAXY_REGISTRY_TIMEOUT_NOTIFIED_PREFIX =
  'amiya-eden:galaxy-registry-timeout-notified:'
export const GALAXY_REGISTRY_TIMEOUT_PERMISSION_DISMISSED_KEY =
  'amiya-eden:galaxy-registry-timeout-permission-dismissed'

export interface GalaxyRegistryTimeoutActiveEntry {
  entry_id: number
  actual_start_at?: string | null
  is_overdue?: boolean
  is_mine?: boolean
}

export interface GalaxyRegistryTimeoutSystemItem {
  solar_system_name: string
  active_entry?: GalaxyRegistryTimeoutActiveEntry | null
}

export interface GalaxyRegistryTimeoutCandidate {
  entryId: number
  systemName: string
  actualStartAt: string | null
}

export interface NotificationStorage {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
}

export const canMonitorGalaxyRegistryTimeout = (roles: readonly string[]) =>
  roles.includes('captain') || roles.includes('admin') || roles.includes('super_admin')

export const parseGalaxyRegistryTimestamp = (value?: string | null) => {
  if (!value) return null
  const normalized = value.includes('T') ? value : value.replace(' ', 'T')
  const parsed = new Date(normalized)
  return Number.isNaN(parsed.getTime()) ? null : parsed.getTime()
}

export const findGalaxyRegistryTimeoutCandidates = (
  systems: readonly GalaxyRegistryTimeoutSystemItem[],
  nowMs: number = Date.now(),
  timeoutMs: number = GALAXY_REGISTRY_TIMEOUT_MS
) =>
  systems.reduce<GalaxyRegistryTimeoutCandidate[]>((candidates, system) => {
    const activeEntry = system.active_entry
    if (!activeEntry?.is_mine) {
      return candidates
    }

    const startMs = parseGalaxyRegistryTimestamp(activeEntry.actual_start_at)
    const isTimedOutByStart = startMs !== null && nowMs >= startMs + timeoutMs
    if (!activeEntry.is_overdue && !isTimedOutByStart) {
      return candidates
    }

    candidates.push({
      entryId: activeEntry.entry_id,
      systemName: system.solar_system_name,
      actualStartAt: activeEntry.actual_start_at ?? null
    })
    return candidates
  }, [])

export const getGalaxyRegistryTimeoutNotifiedKey = (entryId: number) =>
  `${GALAXY_REGISTRY_TIMEOUT_NOTIFIED_PREFIX}${entryId}`

export const hasGalaxyRegistryTimeoutNotified = (
  storage: NotificationStorage | null,
  entryId: number
) => {
  if (!storage) return false
  try {
    return storage.getItem(getGalaxyRegistryTimeoutNotifiedKey(entryId)) === '1'
  } catch {
    return false
  }
}

export const markGalaxyRegistryTimeoutNotified = (
  storage: NotificationStorage | null,
  entryId: number
) => {
  if (!storage) return
  try {
    storage.setItem(getGalaxyRegistryTimeoutNotifiedKey(entryId), '1')
  } catch {
    // 忽略浏览器存储异常，避免通知监控影响主流程。
  }
}

export const hasDismissedGalaxyRegistryTimeoutPermissionPrompt = (
  storage: NotificationStorage | null
) => {
  if (!storage) return false
  try {
    return storage.getItem(GALAXY_REGISTRY_TIMEOUT_PERMISSION_DISMISSED_KEY) === '1'
  } catch {
    return false
  }
}

export const markGalaxyRegistryTimeoutPermissionPromptDismissed = (
  storage: NotificationStorage | null
) => {
  if (!storage) return
  try {
    storage.setItem(GALAXY_REGISTRY_TIMEOUT_PERMISSION_DISMISSED_KEY, '1')
  } catch {
    // 忽略浏览器存储异常，避免权限提示影响主流程。
  }
}
