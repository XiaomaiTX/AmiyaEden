import { Menu, UserRound } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { shellMenuItems } from '@/layout/menu-config'
import { usePreferenceStore, useSessionStore } from '@/stores'

function getRouteTitle(pathname: string, t: (key: string) => string) {
  const found = shellMenuItems.find((item) => item.to === pathname)
  return found ? t(found.labelKey) : t('shell.unnamedPage')
}

export function HeaderBar() {
  const { t } = useI18n()
  const location = useLocation()
  const title = getRouteTitle(location.pathname, t)

  const locale = usePreferenceStore((state) => state.locale)
  const setLocale = usePreferenceStore((state) => state.setLocale)
  const toggleSidebar = usePreferenceStore((state) => state.toggleSidebar)

  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const characterName = useSessionStore((state) => state.characterName)
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)
  const clearSession = useSessionStore((state) => state.clearSession)

  return (
    <header className="flex min-h-14 items-center justify-between border-b bg-background/95 px-4 backdrop-blur">
      <div className="flex items-center gap-3">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="hidden md:inline-flex"
          onClick={toggleSidebar}
          aria-label="toggle-sidebar"
        >
          <Menu className="h-4 w-4" />
        </Button>
        <div className="flex flex-col">
          <span className="text-xs text-muted-foreground">{t('shell.runtime')}</span>
          <span className="text-sm font-medium">{title}</span>
        </div>
      </div>

      <div className="flex items-center gap-2">
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
        ) : (
          <Button
            type="button"
            size="sm"
            onClick={() =>
              setSessionSnapshot({
                isLoggedIn: true,
                characterId: 1001,
                characterName: 'Amiya',
                roles: ['admin'],
                authList: ['route:dashboard:view'],
              })
            }
          >
            {t('auth.mockLogin')}
          </Button>
        )}
      </div>
    </header>
  )
}
