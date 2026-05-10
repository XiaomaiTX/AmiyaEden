import { useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { subscribeUnauthorized } from '@/auth/unauthorized'
import { notifyError } from '@/feedback'
import { resolveLocaleText } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'

export function UnauthorizedBridge() {
  const navigate = useNavigate()
  const location = useLocation()
  const locale = usePreferenceStore((state) => state.locale)
  const clearSession = useSessionStore((state) => state.clearSession)

  useEffect(() => {
    return subscribeUnauthorized((event) => {
      if (location.pathname === '/auth/login' || location.pathname === '/login') {
        return
      }

      clearSession()
      notifyError(resolveLocaleText(locale, 'feedback.unauthorized'))

      const currentPath = `${location.pathname}${location.search}${location.hash}`
      const redirect = event.redirectTo ?? currentPath
      navigate(`/auth/login?redirect=${encodeURIComponent(redirect)}`, { replace: true })
    })
  }, [clearSession, locale, location.hash, location.pathname, location.search, navigate])

  return null
}

