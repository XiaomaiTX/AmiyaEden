import { useEffect, useState } from 'react'
import { adminTicketStatistics } from '@/api/ticket'
import { useI18n } from '@/i18n'
import type { Statistics } from '@/types/api/ticket'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <article className="rounded-lg border bg-card p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{Intl.NumberFormat().format(value)}</p>
    </article>
  )
}

export function TicketStatisticsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [stats, setStats] = useState<Statistics | null>(null)

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await adminTicketStatistics()
        if (!cancelled) setStats(response)
      } catch (caughtError) {
        if (!cancelled) {
          setStats(null)
          setError(getErrorMessage(caughtError, t('ticketStatistics.loadFailed')))
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadData()
    return () => {
      cancelled = true
    }
  }, [t])

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('ticketStatistics.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('ticketStatistics.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('ticketStatistics.loading')}</p> : null}

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={t('ticketStatistics.total')} value={stats?.total ?? 0} />
        <StatCard label={t('ticketStatistics.status.pending')} value={stats?.status.pending ?? 0} />
        <StatCard
          label={t('ticketStatistics.status.in_progress')}
          value={stats?.status.in_progress ?? 0}
        />
        <StatCard
          label={t('ticketStatistics.status.completed')}
          value={stats?.status.completed ?? 0}
        />
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-wrap gap-4 text-sm">
          <span>
            {t('ticketStatistics.recent7d')}: {stats?.recent_7d ?? 0}
          </span>
          <span>
            {t('ticketStatistics.recent30d')}: {stats?.recent_30d ?? 0}
          </span>
          <span>
            {t('ticketStatistics.pendingCount')}: {stats?.pendingCount ?? 0}
          </span>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <h2 className="text-base font-semibold">{t('ticketStatistics.byCategory')}</h2>
        <div className="mt-3 space-y-2">
          {Object.entries(stats?.category ?? {}).map(([name, count]) => (
            <div key={name} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
              <span>{name}</span>
              <span className="font-medium">{count}</span>
            </div>
          ))}
          {!loading && Object.keys(stats?.category ?? {}).length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('ticketStatistics.empty')}</p>
          ) : null}
        </div>
      </div>
    </section>
  )
}
