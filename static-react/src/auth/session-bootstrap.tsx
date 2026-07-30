import { useEffect } from 'react'
import { fetchGetUserInfo } from '@/api/auth'
import { toSessionSnapshot } from '@/auth/session-hydration'
import { useSessionStore } from '@/stores'

let sessionRefreshInFlight: ReturnType<typeof fetchGetUserInfo> | null = null

function refreshSessionOnce() {
  if (!sessionRefreshInFlight) {
    sessionRefreshInFlight = fetchGetUserInfo().finally(() => {
      sessionRefreshInFlight = null
    })
  }
  return sessionRefreshInFlight
}

export function SessionBootstrap() {
  const accessToken = useSessionStore((state) => state.accessToken)
  const bootstrapRequired = useSessionStore((state) => state.bootstrapRequired)
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)
  const clearSession = useSessionStore((state) => state.clearSession)

  useEffect(() => {
    if (!accessToken || !bootstrapRequired) {
      return
    }

    let cancelled = false

    void refreshSessionOnce()
      .then((userInfo) => {
        if (!cancelled) {
          setSessionSnapshot(toSessionSnapshot(userInfo))
        }
      })
      .catch(() => {
        if (!cancelled) {
          clearSession()
        }
      })

    return () => {
      cancelled = true
    }
  }, [accessToken, bootstrapRequired, clearSession, setSessionSnapshot])

  return null
}
