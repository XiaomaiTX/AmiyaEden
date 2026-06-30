import { ElMessageBox } from 'element-plus'
import { computed, onMounted, onUnmounted, watch, type WatchStopHandle } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { fetchGalaxyRegistrySystems } from '@/api/galaxy-registry'
import {
  createBrowserNotificationClient,
  sendBrowserNotification
} from '@/hooks/core/browserNotification'
import { useUserStore } from '@/store/modules/user'
import {
  canMonitorGalaxyRegistryTimeout,
  findGalaxyRegistryTimeoutCandidates,
  hasDismissedGalaxyRegistryTimeoutPermissionPrompt,
  hasGalaxyRegistryTimeoutNotified,
  markGalaxyRegistryTimeoutNotified,
  markGalaxyRegistryTimeoutPermissionPromptDismissed,
  type NotificationStorage
} from './galaxyRegistryTimeoutNotification'

const POLL_INTERVAL_MS = 60 * 1000
const GALAXY_REGISTRY_TIMEOUT_ROUTE = '/dashboard/galaxy-registry'

const getLocalStorage = (): NotificationStorage | null => {
  if (typeof window === 'undefined') return null
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export const useGalaxyRegistryTimeoutNotification = () => {
  const userStore = useUserStore()
  const router = useRouter()
  const { t } = useI18n()
  const browserNotificationClient = createBrowserNotificationClient()

  let timer: number | null = null
  let refreshing = false
  let stopCanMonitorWatch: WatchStopHandle | null = null

  const canMonitor = computed(
    () =>
      userStore.isLogin &&
      canMonitorGalaxyRegistryTimeout((userStore.info.roles ?? []) as readonly string[])
  )

  const focusGalaxyRegistry = () => {
    if (typeof window !== 'undefined') {
      window.focus()
    }
    router.push(GALAXY_REGISTRY_TIMEOUT_ROUTE).catch(() => undefined)
  }

  const ensureBrowserNotificationPermission = async () => {
    if (!browserNotificationClient.isSupported()) {
      return false
    }

    const permission = browserNotificationClient.getPermission()
    if (permission === 'granted') {
      return true
    }
    if (permission === 'denied') {
      return false
    }

    const storage = getLocalStorage()
    if (hasDismissedGalaxyRegistryTimeoutPermissionPrompt(storage)) {
      return false
    }

    try {
      await ElMessageBox.confirm(
        t('browserNotifications.permission.message'),
        t('browserNotifications.permission.title'),
        {
          confirmButtonText: t('browserNotifications.permission.confirm'),
          cancelButtonText: t('browserNotifications.permission.cancel'),
          type: 'info'
        }
      )
    } catch {
      markGalaxyRegistryTimeoutPermissionPromptDismissed(storage)
      return false
    }

    const requestedPermission = await browserNotificationClient.requestPermission()
    if (requestedPermission !== 'granted') {
      markGalaxyRegistryTimeoutPermissionPromptDismissed(storage)
      return false
    }

    return true
  }

  const refresh = async () => {
    if (!canMonitor.value || refreshing) {
      return
    }

    refreshing = true
    try {
      const response = await fetchGalaxyRegistrySystems()
      const storage = getLocalStorage()
      const candidates = findGalaxyRegistryTimeoutCandidates(response.items || []).filter(
        (candidate) => !hasGalaxyRegistryTimeoutNotified(storage, candidate.entryId)
      )

      if (!candidates.length) {
        return
      }

      const canNotify = await ensureBrowserNotificationPermission()
      if (!canNotify) {
        return
      }

      candidates.forEach((candidate) => {
        const sent = sendBrowserNotification(browserNotificationClient, {
          title: t('browserNotifications.galaxyRegistryTimeout.title'),
          body: t('browserNotifications.galaxyRegistryTimeout.body', {
            system: candidate.systemName
          }),
          tag: `galaxy-registry-timeout-${candidate.entryId}`,
          data: {
            type: 'galaxy-registry-timeout',
            entryId: candidate.entryId
          },
          onClick: focusGalaxyRegistry
        })

        if (sent) {
          markGalaxyRegistryTimeoutNotified(storage, candidate.entryId)
        }
      })
    } catch {
      // 通知监控静默失败，避免影响全局布局和页面加载。
    } finally {
      refreshing = false
    }
  }

  const start = () => {
    if (timer !== null) {
      return
    }

    void refresh()
    timer = window.setInterval(() => {
      void refresh()
    }, POLL_INTERVAL_MS)
  }

  const stop = () => {
    if (timer === null) {
      return
    }
    window.clearInterval(timer)
    timer = null
  }

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      void refresh()
    }
  }

  onMounted(() => {
    stopCanMonitorWatch = watch(
      canMonitor,
      (enabled) => {
        if (enabled) {
          start()
          return
        }
        stop()
      },
      { immediate: true }
    )

    document.addEventListener('visibilitychange', handleVisibilityChange)
    window.addEventListener('focus', refresh)
  })

  onUnmounted(() => {
    stopCanMonitorWatch?.()
    stopCanMonitorWatch = null
    stop()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    window.removeEventListener('focus', refresh)
  })
}
