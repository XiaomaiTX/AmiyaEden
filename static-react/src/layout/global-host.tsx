import { useI18n } from '@/i18n'

export function GlobalHost() {
  const { t } = useI18n()

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex justify-end p-4">
      <div className="rounded-md border bg-card/95 px-3 py-1 text-xs text-muted-foreground shadow-sm backdrop-blur">
        {t('shell.globalHostPlaceholder')}
      </div>
    </div>
  )
}
