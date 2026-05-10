import { Link } from 'react-router-dom'
import { useI18n } from '@/i18n'

export function NotFoundPage() {
  const { t } = useI18n()

  return (
    <section className="rounded-lg border bg-card p-6">
      <h1 className="text-xl font-semibold">{t('errors.notFoundTitle')}</h1>
      <p className="mt-2 text-sm text-muted-foreground">{t('errors.notFoundDesc')}</p>
      <Link className="mt-3 inline-block text-sm underline" to="/">
        {t('common.backHome')}
      </Link>
    </section>
  )
}
