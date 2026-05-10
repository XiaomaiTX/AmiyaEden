import { Link } from 'react-router-dom'
import { dispatchUnauthorized } from '@/auth'
import { Button } from '@/components/ui/button'
import { confirmAction, notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'

export function HomePage() {
  const { t } = useI18n()
  const locale = usePreferenceStore((state) => state.locale)
  const sidebarCollapsed = usePreferenceStore((state) => state.sidebarCollapsed)

  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const characterName = useSessionStore((state) => state.characterName)
  const roles = useSessionStore((state) => state.roles)
  const hydratedAt = useSessionStore((state) => state.hydratedAt)

  return (
    <div className="space-y-4">
      <section className="rounded-lg border bg-card p-4">
        <h1 className="text-xl font-semibold">{t('home.title')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('home.description')}</p>
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        <article className="rounded-lg border bg-card p-4 text-sm">
          <h2 className="font-semibold">Preference Snapshot</h2>
          <p className="mt-2 text-muted-foreground">locale: {locale}</p>
          <p className="text-muted-foreground">sidebarCollapsed: {String(sidebarCollapsed)}</p>
        </article>

        <article className="rounded-lg border bg-card p-4 text-sm">
          <h2 className="font-semibold">Session Snapshot</h2>
          <p className="mt-2 text-muted-foreground">loggedIn: {String(isLoggedIn)}</p>
          <p className="text-muted-foreground">character: {characterName ?? 'none'}</p>
          <p className="text-muted-foreground">
            roles: {roles.length > 0 ? roles.join(', ') : 'none'}
          </p>
          <p className="text-muted-foreground">hydratedAt: {hydratedAt ?? 'not yet'}</p>
        </article>
      </section>

      <section className="flex flex-wrap gap-3">
        <Button asChild>
          <Link to="/500">{t('home.open500')}</Link>
        </Button>
        <Button asChild variant="outline">
          <Link to="/admin-demo">{t('home.permissionRoute')}</Link>
        </Button>
        <Button asChild variant="outline">
          <Link to="/missing-route">{t('home.open404')}</Link>
        </Button>
        <Button type="button" variant="outline" onClick={() => notifySuccess(t('feedback.successDemo'))}>
          {t('home.showSuccessToast')}
        </Button>
        <Button type="button" variant="outline" onClick={() => notifyError(t('feedback.errorDemo'))}>
          {t('home.showErrorToast')}
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={async () => {
            const accepted = await confirmAction({
              title: t('feedback.confirmTitle'),
              message: t('feedback.confirmMessage'),
              confirmText: t('common.confirm'),
              cancelText: t('common.cancel'),
            })

            if (accepted) {
              notifySuccess(t('feedback.confirmed'))
              return
            }

            notifyError(t('feedback.cancelled'))
          }}
        >
          {t('home.openConfirmDialog')}
        </Button>
        <Button
          type="button"
          variant="destructive"
          onClick={() => dispatchUnauthorized({ reason: 'manual' })}
        >
          {t('home.mock401')}
        </Button>
      </section>
    </div>
  )
}
