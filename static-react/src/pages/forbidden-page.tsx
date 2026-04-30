import { Link } from 'react-router-dom'
import { useI18n } from '@/i18n'

export function ForbiddenPage() {
  const { t } = useI18n()

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-xl flex-col justify-center gap-4 px-6">
      <section className="rounded-lg border bg-card p-6">
        <h1 className="text-xl font-semibold">{t('errors.forbiddenTitle')}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('errors.forbiddenDesc')}</p>
        <Link className="mt-3 inline-block text-sm underline" to="/">
          {t('common.backHome')}
        </Link>
      </section>
    </main>
  )
}
