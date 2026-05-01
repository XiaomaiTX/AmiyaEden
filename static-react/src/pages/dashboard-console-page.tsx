import { useEffect, useState } from 'react'
import { fetchDashboard } from '@/api/dashboard'
import { useI18n } from '@/i18n'

export function DashboardConsolePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<Api.Dashboard.DashboardResult | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const result = await fetchDashboard()
        if (!cancelled) {
          setData(result)
        }
      } catch {
        if (!cancelled) {
          setError(t('dashboardConsole.loadFailed'))
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [t])

  if (loading) {
    return <section className="rounded-lg border bg-card p-5 text-sm">{t('dashboardConsole.loading')}</section>
  }

  if (error) {
    return (
      <section className="rounded-lg border bg-card p-5">
        <h1 className="text-lg font-semibold">{t('dashboardConsole.title')}</h1>
        <p className="mt-2 text-sm text-destructive">{error}</p>
      </section>
    )
  }

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('dashboardConsole.title')}</h1>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={t('dashboardConsole.cards.online')} value={data?.cards.online_count ?? 0} />
        <StatCard label={t('dashboardConsole.cards.totalAssets')} value={data?.cards.total_assets_count ?? 0} />
        <StatCard label={t('dashboardConsole.cards.totalIsk')} value={data?.cards.total_assets_price ?? 0} />
        <StatCard label={t('dashboardConsole.cards.myPap')} value={data?.cards.my_pap_count ?? 0} />
      </div>

      <div className="rounded-lg border bg-card p-4">
        <h2 className="text-sm font-medium">{t('dashboardConsole.fleets')}</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          {t('dashboardConsole.fleetCount')}: {data?.fleets.length ?? 0}
        </p>
      </div>

      <div className="rounded-lg border bg-card p-4">
        <h2 className="text-sm font-medium">{t('dashboardConsole.srp')}</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          {t('dashboardConsole.srpCount')}: {data?.srp_list.length ?? 0}
        </p>
      </div>
    </section>
  )
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <article className="rounded-lg border bg-card p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold">{Intl.NumberFormat().format(value)}</p>
    </article>
  )
}
