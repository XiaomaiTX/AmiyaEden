import { useCallback, useEffect, useRef } from 'react'
import { fetchGalaxyRegistrySystems } from '@/api/galaxy-registry'
import { confirmAction } from '@/feedback'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'

const timeoutMs = 2 * 60 * 60 * 1000
const notifiedPrefix = 'amiya-eden:galaxy-registry-timeout-notified:'
const permissionDismissedKey = 'amiya-eden:galaxy-registry-timeout-permission-dismissed'

function storage() { try { return window.localStorage } catch { return null } }
function parseTime(value: string) { const time = new Date(value.includes('T') ? value : value.replace(' ', 'T')).valueOf(); return Number.isNaN(time) ? null : time }

export function useGalaxyRegistryTimeoutNotification() {
  const { t } = useI18n()
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const roles = useSessionStore((state) => state.roles)
  const refreshing = useRef(false)
  const enabled = isLoggedIn && (roles.includes('captain') || roles.includes('admin') || roles.includes('super_admin'))
  const refresh = useCallback(async () => {
    if (refreshing.current || typeof Notification === 'undefined') return
    refreshing.current = true
    try {
      const systems = await fetchGalaxyRegistrySystems()
      const candidates = systems.items.filter((system) => {
        const entry = system.active_entry
        const startedAt = entry ? parseTime(entry.actual_start_at) : null
        return Boolean(entry?.is_mine && (entry.is_overdue || (startedAt !== null && Date.now() >= startedAt + timeoutMs)))
      })
      if (!candidates.length) return
      const local = storage()
      const unseen = candidates.filter((system) => local?.getItem(`${notifiedPrefix}${system.active_entry?.entry_id}`) !== '1')
      if (!unseen.length || Notification.permission === 'denied') return
      if (Notification.permission !== 'granted') {
        if (local?.getItem(permissionDismissedKey) === '1') return
        const accepted = await confirmAction({ title: t('browserNotification.title'), message: t('browserNotification.message'), confirmText: t('browserNotification.confirm'), cancelText: t('common.cancel') })
        if (!accepted || await Notification.requestPermission() !== 'granted') { local?.setItem(permissionDismissedKey, '1'); return }
      }
      unseen.forEach((system) => {
        const entry = system.active_entry
        if (!entry) return
        const notification = new Notification(t('browserNotification.timeoutTitle'), { body: t('browserNotification.timeoutBody', { system: system.solar_system_name }), tag: `galaxy-registry-timeout-${entry.entry_id}` })
        notification.onclick = () => { window.focus(); window.location.hash = '#/dashboard/galaxy-registry' }
        local?.setItem(`${notifiedPrefix}${entry.entry_id}`, '1')
      })
    } catch { /* Notifications must not interrupt normal application loading. */ } finally { refreshing.current = false }
  }, [t])
  useEffect(() => {
    if (!enabled) return undefined
    void refresh()
    const timer = window.setInterval(() => void refresh(), 60_000)
    const onVisible = () => { if (document.visibilityState === 'visible') void refresh() }
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('focus', refresh)
    return () => { window.clearInterval(timer); document.removeEventListener('visibilitychange', onVisible); window.removeEventListener('focus', refresh) }
  }, [enabled, refresh])
}
