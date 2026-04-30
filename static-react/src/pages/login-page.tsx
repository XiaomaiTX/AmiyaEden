import { useMemo } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'

function useRedirectPath() {
  const location = useLocation()

  return useMemo(() => {
    const query = new URLSearchParams(location.search)
    const redirect = query.get('redirect')

    if (!redirect || redirect === '/login') {
      return '/'
    }

    return redirect
  }, [location.search])
}

export function LoginPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const redirectPath = useRedirectPath()
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-4 px-6">
      <section className="rounded-lg border bg-card p-6">
        <h1 className="text-xl font-semibold">{t('auth.loginTitle')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('auth.loginDescription')}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t('auth.loginRedirectTo')}: {redirectPath}
        </p>

        <div className="mt-4 flex gap-2">
          <Button
            type="button"
            onClick={() => {
              setSessionSnapshot({
                isLoggedIn: true,
                characterId: 1001,
                characterName: 'Amiya',
                roles: ['admin'],
                authList: ['route:dashboard:view'],
              })
              navigate(redirectPath, { replace: true })
            }}
          >
            {t('auth.mockLogin')}
          </Button>
        </div>

        {isLoggedIn ? (
          <p className="mt-3 text-xs text-muted-foreground">{t('auth.alreadyLoggedIn')}</p>
        ) : null}
      </section>
    </main>
  )
}
