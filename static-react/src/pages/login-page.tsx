import { useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { getEveSSOLoginURL } from '@/api/auth'
import { useI18n } from '@/i18n'

const SSO_REDIRECT_STORAGE_KEY = 'auth:sso:redirect'

function useRedirectPath() {
  const location = useLocation()

  return useMemo(() => {
    const query = new URLSearchParams(location.search)
    const redirect = query.get('redirect')

    if (!redirect || redirect === '/auth/login' || redirect === '/login') {
      return '/'
    }

    return redirect
  }, [location.search])
}

export function LoginPage() {
  const { t } = useI18n()
  const redirectPath = useRedirectPath()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleEveLogin = async () => {
    setLoading(true)
    setError(null)
    try {
      sessionStorage.setItem(SSO_REDIRECT_STORAGE_KEY, redirectPath)
      const loginURL = await getEveSSOLoginURL()
      window.location.href = loginURL
    } catch (error) {
      setError(error instanceof Error ? error.message : t('auth.loginStartFailed'))
      setLoading(false)
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-4 px-6">
      <section className="rounded-lg border bg-card p-6">
        <h1 className="text-xl font-semibold">{t('auth.loginTitle')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('auth.loginDescription')}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t('auth.loginRedirectTo')}: {redirectPath}
        </p>

        <div className="mt-4 flex gap-2">
          <Button type="button" onClick={handleEveLogin} disabled={loading}>
            {loading ? t('auth.loginStarting') : t('auth.loginWithEve')}
          </Button>
        </div>

        {error ? <p className="mt-3 text-xs text-destructive">{error}</p> : null}
      </section>
    </main>
  )
}

