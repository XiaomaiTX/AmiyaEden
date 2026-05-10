import { UserRound } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { appRouteSpecs } from '@/app/migration-routes'
import { ModeToggle } from '@/components/mode-toggle'
import { Button } from '@/components/ui/button'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { useI18n } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'

function getRouteTitle(pathname: string, t: (key: string) => string) {
  const found = appRouteSpecs.find((item) => {
    if (!item.path) return pathname === '/'
    const pattern = `^/${item.path.replace(/:[^/]+/g, '[^/]+')}$`
    return new RegExp(pattern).test(pathname)
  })
  return found ? t(found.titleKey) : t('shell.unnamedPage')
}

export function HeaderBar() {
  const { t } = useI18n()
  const location = useLocation()
  const title = getRouteTitle(location.pathname, t)

  const locale = usePreferenceStore((state) => state.locale)
  const setLocale = usePreferenceStore((state) => state.setLocale)
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const characterName = useSessionStore((state) => state.characterName)
  const clearSession = useSessionStore((state) => state.clearSession)

  return (
    <header className="flex min-h-14 items-center justify-between border-b bg-background/95 px-4 backdrop-blur">
      <div className="flex items-center gap-3">
        <SidebarTrigger />
        <div className="flex flex-col">
          <span className="text-xs text-muted-foreground">{t('shell.runtime')}</span>
          <span className="text-sm font-medium">{title}</span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <ModeToggle />
        <Button
          type="button"
          variant="outline"
          size="sm"
          title={t('common.switchLocale')}
          onClick={() => setLocale(locale === 'zh-CN' ? 'en-US' : 'zh-CN')}
        >
          {locale}
        </Button>

        {isLoggedIn ? (
          <Button type="button" variant="secondary" size="sm" onClick={clearSession}>
            <UserRound className="mr-1 h-4 w-4" />
            {characterName ?? 'User'}
          </Button>
        ) : null}
      </div>
    </header>
  )
}
