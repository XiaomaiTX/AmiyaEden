import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { fetchGetUserInfo } from '@/api/auth'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'

type CallbackStatus = 'loading' | 'success' | 'error'
const SSO_REDIRECT_STORAGE_KEY = 'auth:sso:redirect'

function useCallbackQuery(search: string) {
  const query = useMemo(() => new URLSearchParams(search), [search])
  return {
    token: query.get('token'),
    redirect: query.get('redirect'),
    error: query.get('error'),
    errorDescription: query.get('error_description'),
  }
}

export function AuthCallbackPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { token, redirect, error, errorDescription } = useCallbackQuery(location.search)
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)
  const clearSession = useSessionStore((state) => state.clearSession)

  const [status, setStatus] = useState<CallbackStatus>('loading')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    let cancelled = false

    const run = async () => {
      if (error) {
        setStatus('error')
        setErrorMessage(errorDescription || error)
        return
      }

      if (!token) {
        setStatus('error')
        setErrorMessage(t('auth.callbackMissingToken'))
        return
      }

      try {
        setSessionSnapshot({
          accessToken: token,
          isLoggedIn: true,
        })

        const userInfo = await fetchGetUserInfo()
        if (cancelled) return

        setSessionSnapshot({
          isLoggedIn: true,
          accessToken: token,
          characterId: userInfo.primaryCharacterId ?? null,
          characterName: userInfo.userName,
          roles: userInfo.roles,
          authList: [],
          isCurrentlyNewbro: userInfo.isCurrentlyNewbro === true,
          isMentorMenteeEligible: userInfo.isMentorMenteeEligible === true,
        })

        setStatus('success')

        const rememberedRedirect = sessionStorage.getItem(SSO_REDIRECT_STORAGE_KEY)
        sessionStorage.removeItem(SSO_REDIRECT_STORAGE_KEY)
        const target = redirect || rememberedRedirect || '/'
        window.setTimeout(() => {
          navigate(target, { replace: true })
        }, 800)
      } catch {
        if (cancelled) return
        clearSession()
        setStatus('error')
        setErrorMessage(t('auth.callbackVerifyFailed'))
      }
    }

    void run()

    return () => {
      cancelled = true
    }
  }, [clearSession, error, errorDescription, navigate, redirect, setSessionSnapshot, t, token])

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-4 px-6">
      <section className="rounded-lg border bg-card p-6">
        {status === 'loading' ? <h1 className="text-xl font-semibold">{t('auth.callbackLoading')}</h1> : null}
        {status === 'success' ? <h1 className="text-xl font-semibold">{t('auth.callbackSuccess')}</h1> : null}
        {status === 'error' ? <h1 className="text-xl font-semibold">{t('auth.callbackError')}</h1> : null}

        {status === 'loading' ? (
          <p className="mt-2 text-sm text-muted-foreground">{t('auth.callbackLoadingSub')}</p>
        ) : null}

        {status === 'success' ? (
          <p className="mt-2 text-sm text-muted-foreground">{t('auth.callbackSuccessSub')}</p>
        ) : null}

        {status === 'error' ? <p className="mt-2 text-sm text-destructive">{errorMessage}</p> : null}

        {status === 'error' ? (
          <div className="mt-4">
            <Button type="button" onClick={() => navigate('/auth/login', { replace: true })}>
              {t('auth.backToLogin')}
            </Button>
          </div>
        ) : null}
      </section>
    </main>
  )
}
