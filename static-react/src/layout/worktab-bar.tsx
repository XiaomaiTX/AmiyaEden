import { Pin, PinOff, X } from 'lucide-react'
import { useEffect } from 'react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { appRouteSpecs } from '@/app/migration-routes'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { useSessionStore, useWorktabStore } from '@/stores'

function matchesRoute(pattern: string, pathname: string) {
  return new RegExp(`^/${pattern.replace(/:[^/]+/g, '[^/]+')}$`).test(pathname)
}

export function WorktabBar() {
  const { t } = useI18n()
  const location = useLocation()
  const navigate = useNavigate()
  const characterId = useSessionStore((state) => state.characterId)
  const opened = useWorktabStore((state) => state.opened)
  const open = useWorktabStore((state) => state.open)
  const close = useWorktabStore((state) => state.close)
  const closeOthers = useWorktabStore((state) => state.closeOthers)
  const closeAll = useWorktabStore((state) => state.closeAll)
  const toggleFixed = useWorktabStore((state) => state.toggleFixed)
  const resetForCharacter = useWorktabStore((state) => state.resetForCharacter)

  useEffect(() => {
    resetForCharacter(characterId)
  }, [characterId, resetForCharacter])

  useEffect(() => {
    const route = appRouteSpecs.find((item) => matchesRoute(item.path, location.pathname))
    if (!route) {
      return
    }
    open({
      routeId: route.pageType,
      path: `${location.pathname}${location.search}`,
      titleKey: route.titleKey,
      fixed: route.fixedTab,
    })
  }, [location.pathname, location.search, open])

  if (opened.length === 0) {
    return null
  }

  const active = opened.find((tab) => tab.path === `${location.pathname}${location.search}`)

  const handleClose = (path: string) => {
    const wasActive = path === `${location.pathname}${location.search}`
    const nextPath = close(path)
    if (wasActive) {
      navigate(nextPath ?? '/', { replace: true })
    }
  }

  return (
    <div className="flex min-h-10 items-center gap-1 overflow-x-auto border-b bg-muted/20 px-3">
      <div className="flex min-w-max flex-1 items-center gap-1">
        {opened.map((tab) => (
          <div
            key={tab.routeId}
            className={[
              'flex h-8 items-center rounded-md border px-2 text-xs',
              tab.path === `${location.pathname}${location.search}`
                ? 'bg-background text-foreground shadow-sm'
                : 'bg-transparent text-muted-foreground',
            ].join(' ')}
          >
            <NavLink to={tab.path} className="max-w-40 truncate">
              {t(tab.titleKey)}
            </NavLink>
            {tab.fixed ? <Pin className="ml-1 size-3" /> : null}
            {!tab.fixed ? (
              <button
                type="button"
                className="ml-1 rounded p-0.5 hover:bg-muted"
                title={t('worktab.close')}
                onClick={() => handleClose(tab.path)}
              >
                <X className="size-3" />
              </button>
            ) : null}
          </div>
        ))}
      </div>
      {active ? (
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="size-7 shrink-0"
          title={active.fixed ? t('worktab.unpin') : t('worktab.pin')}
          onClick={() => toggleFixed(active.path)}
        >
          {active.fixed ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}
        </Button>
      ) : null}
      <Button
        type="button"
        size="sm"
        variant="ghost"
        className="h-7 shrink-0 px-2 text-xs"
        onClick={() => active && closeOthers(active.path)}
      >
        {t('worktab.closeOthers')}
      </Button>
      <Button
        type="button"
        size="sm"
        variant="ghost"
        className="h-7 shrink-0 px-2 text-xs"
        onClick={() => navigate(closeAll() ?? '/', { replace: true })}
      >
        {t('worktab.closeAll')}
      </Button>
    </div>
  )
}
