import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import type { DashboardFleetItem } from '@/types/api/dashboard'
import { cn } from '@/lib/utils'

interface DashboardFleetListProps {
  fleets: DashboardFleetItem[]
  className?: string
}

function resolveImportanceKey(importance: string | undefined): string | null {
  if (!importance) {
    return null
  }

  const known = new Set(['strat_op', 'cta', 'other'])
  return known.has(importance) ? importance : null
}

export function DashboardFleetList({ fleets, className }: DashboardFleetListProps) {
  const { t } = useI18n()

  return (
    <section className={cn('flex h-128 flex-col rounded-lg border bg-card p-5', className)}>
      <header className="mb-2 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-semibold">{t('dashboardConsole.fleetList.title')}</h4>
          <p className="text-xs text-muted-foreground">
            {t('dashboardConsole.fleetList.records', { count: fleets.length })}
          </p>
        </div>
      </header>

      <div className="mt-2 flex-1 overflow-auto">
        {fleets.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            {t('dashboardConsole.fleetList.empty')}
          </div>
        ) : (
          <ul className="divide-y divide-border">
            {fleets.map((item, index) => {
              const importanceKey = resolveImportanceKey(item.importance)
              return (
                <li
                  key={`${item.source}-${item.id}-${index}`}
                  className="flex items-center gap-3 py-3 text-sm"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span
                        className={cn(
                          'inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-medium uppercase',
                          item.source === 'alliance'
                            ? 'border-amber-500/40 text-amber-600 dark:text-amber-400'
                            : 'border-primary/40 text-primary'
                        )}
                      >
                        {item.source === 'alliance'
                          ? t('dashboardConsole.fleetList.source.alliance')
                          : t('dashboardConsole.fleetList.source.internal')}
                      </span>
                      <span className="truncate font-medium">{item.title}</span>
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span>{formatTime(item.start_at)}</span>
                      {item.character_name ? <span>{item.character_name}</span> : null}
                      {item.ship_type_name ? <span>{item.ship_type_name}</span> : null}
                      {importanceKey ? (
                        <span>{t(`dashboardConsole.fleetList.importance.${importanceKey}`)}</span>
                      ) : null}
                    </div>
                  </div>
                  <div className="shrink-0 text-right">
                    <span className="font-medium text-primary">{item.pap_count}</span>
                    <span className="ml-0.5 text-xs text-muted-foreground">{t('common.pap')}</span>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </section>
  )
}
