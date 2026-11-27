import { useEffect, useReducer, useRef, useState } from 'react'
import { HttpError } from '@/api/http-error'
import { unwrapContract } from '@/api/response'
import type { QQOfficialTaskAccepted } from '@/api/generated/client'
import { useCorpCapability } from '@/hooks/use-corp-capability'
import { useI18n } from '@/i18n'
import { notifyError, notifySuccess } from '@/feedback'
import { createQQSubmissionKeys, qqErrorKey } from '@/lib/qq-official'

type Reader<T> = (signal: AbortSignal) => Promise<T>

/** Coalesce reads; abort on scope changes and never poll a hidden page. */
export function useQQOfficialRead<T>(
  read: Reader<T>,
  pollMs?: number,
  shouldPoll?: (value: T) => boolean
) {
  const [version, reload] = useReducer((n: number) => n + 1, 0)
  const [view, setView] = useState<{ reader: Reader<T>; value: T | null; failed: boolean }>()
  const current = view?.reader === read ? view : undefined
  const inFlight = useRef<AbortController | null>(null)

  useEffect(() => {
    let last: T | undefined
    const refresh = async () => {
      if (inFlight.current) return
      const controller = new AbortController()
      inFlight.current = controller
      try {
        const value = await read(controller.signal)
        if (!controller.signal.aborted) {
          last = value
          setView({ reader: read, value, failed: false })
        }
      } catch {
        if (!controller.signal.aborted) setView({ reader: read, value: null, failed: true })
      } finally {
        if (inFlight.current === controller) inFlight.current = null
      }
    }
    const poll = () => {
      if (
        document.visibilityState !== 'hidden' &&
        (last === undefined || !shouldPoll || shouldPoll(last))
      )
        void refresh()
    }
    void refresh()
    const timer = pollMs ? window.setInterval(poll, pollMs) : undefined
    if (pollMs) document.addEventListener('visibilitychange', poll)
    return () => {
      inFlight.current?.abort()
      inFlight.current = null
      window.clearInterval(timer)
      if (pollMs) document.removeEventListener('visibilitychange', poll)
    }
  }, [pollMs, read, shouldPoll, version])

  const replace = (value: T) => {
    inFlight.current?.abort()
    inFlight.current = null
    setView({ reader: read, value, failed: false })
  }

  return {
    data: current?.value ?? null,
    error: current?.failed ?? false,
    loading: !current,
    reload,
    replace,
  }
}

export function useQQOfficialActions() {
  const { t } = useI18n()
  const { hasCapability } = useCorpCapability()
  const canManage = hasCapability('system.manage')
  const [busy, setBusy] = useState(false)
  const pending = useRef(false)
  const [keys] = useState(createQQSubmissionKeys)

  const run = async <T>(action: () => Promise<T>) => {
    if (pending.current || !canManage) return
    pending.current = true
    setBusy(true)
    try {
      return await action()
    } catch (error) {
      notifyError(t(qqErrorKey(error instanceof HttpError ? error.status : undefined)))
    } finally {
      pending.current = false
      setBusy(false)
    }
  }

  const enqueue = (action: string, request: (key: string) => Promise<QQOfficialTaskAccepted>) =>
    run(async () => {
      const data = await unwrapContract(request(keys.get(action)))
      keys.clear(action)
      notifySuccess(t('qqOfficial.receipt', { id: data.task_id }))
      return data
    })

  return { canManage, busy, run, enqueue, clearKey: keys.clear }
}
