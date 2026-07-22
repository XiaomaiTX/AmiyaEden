import { useEffect, useState } from 'react'
import { fetchDashboard } from '@/api/dashboard'
import { Skeleton } from '@/components/ui/skeleton'
import { useI18n } from '@/i18n'
import type { DashboardResult } from '@/types/api/dashboard'
import { DashboardCardList } from './dashboard-console/card-list'
import { DashboardFleetList } from './dashboard-console/fleet-list'
import { DashboardPapChart } from './dashboard-console/pap-chart'
import { DashboardSrpList } from './dashboard-console/srp-list'

export function DashboardConsolePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<DashboardResult | null>(null)

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
    return (
      <section className="space-y-4">
        <DashboardConsoleSkeleton />
      </section>
    )
  }

  if (error) {
    return (
      <section className="rounded-lg border bg-card p-5">
        <p className="text-sm text-destructive">{error}</p>
      </section>
    )
  }

  return (
    <section className="space-y-4">
      <DashboardCardList cards={data?.cards} />

      <div className="grid gap-4 lg:grid-cols-3">
        <DashboardFleetList fleets={data?.fleets ?? []} />
        <DashboardPapChart
          title={t('dashboardConsole.alliancePapTitle')}
          data={data?.pap_stats?.alliance ?? []}
        />
        <DashboardPapChart
          title={t('dashboardConsole.internalPapTitle')}
          data={data?.pap_stats?.internal ?? []}
        />
      </div>

      <DashboardSrpList list={data?.srp_list ?? []} />
    </section>
  )
}

function DashboardConsoleSkeleton() {
  return (
    <>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-32 rounded-lg" />
        ))}
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        <Skeleton className="h-128 rounded-lg" />
        <Skeleton className="h-128 rounded-lg" />
        <Skeleton className="h-128 rounded-lg" />
      </div>
      <Skeleton className="h-64 rounded-lg" />
    </>
  )
}
