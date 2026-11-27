import { computed, onBeforeUnmount, onMounted, shallowRef } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { QQOfficialTaskAccepted } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { useCorpCapability } from '@/hooks/core/useCorpCapability'
import { HttpError } from '@/utils/http/http-error'
import { createQQSubmissionKeys, qqErrorKey, shouldPollQQTask } from './qq-official-helpers'

export const qqReadOptions = { showErrorMessage: false } as const

export function useQQOfficialRead<T>(
  read: (signal: AbortSignal) => Promise<T>,
  intervalMs?: number,
  shouldPoll?: (value: T | null) => boolean
) {
  const data = shallowRef<T | null>(null)
  const error = shallowRef(false)
  const loading = shallowRef(false)
  let controller: AbortController | undefined
  let timer: ReturnType<typeof setInterval> | undefined

  const refresh = async () => {
    if (controller) return
    const active = new AbortController()
    controller = active
    if (!data.value) loading.value = true
    try {
      const value = await read(active.signal)
      if (!active.signal.aborted) {
        data.value = value
        error.value = false
      }
    } catch {
      if (!active.signal.aborted) {
        data.value = null
        error.value = true
      }
    } finally {
      if (controller === active) {
        loading.value = false
        controller = undefined
      }
    }
  }
  const replace = (value: T | null) => {
    controller?.abort()
    controller = undefined
    data.value = value
    loading.value = false
    error.value = false
  }
  const poll = () => {
    if (document.visibilityState !== 'hidden' && (!shouldPoll || shouldPoll(data.value)))
      void refresh()
  }
  onMounted(() => {
    void refresh()
    if (intervalMs) {
      timer = setInterval(poll, intervalMs)
      document.addEventListener('visibilitychange', poll)
    }
  })
  onBeforeUnmount(() => {
    controller?.abort()
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', poll)
  })
  return { data, error, loading, refresh, replace }
}

export function useQQOfficialActions() {
  const { t } = useI18n()
  const { hasCapability } = useCorpCapability()
  const canManage = computed(() => hasCapability('system.manage'))
  const busy = shallowRef(false)
  const keys = createQQSubmissionKeys()
  const run = async <T>(action: () => Promise<T>) => {
    if (busy.value || !canManage.value) return
    busy.value = true
    try {
      return await action()
    } catch (error) {
      ElMessage.error(t(qqErrorKey(error instanceof HttpError ? error.code : undefined)))
    } finally {
      busy.value = false
    }
  }
  const enqueue = (action: string, request: (key: string) => Promise<QQOfficialTaskAccepted>) =>
    run(async () => {
      const data = await unwrapContract(request(keys.get(action)))
      keys.clear(action)
      ElMessage.success(t('qqOfficial.receipt', { id: data.task_id }))
      return data
    })
  return { canManage, busy, run, enqueue, clearKey: keys.clear }
}

export function qqTaskPolling(task: { status: string } | null) {
  return shouldPollQQTask(task?.status)
}
