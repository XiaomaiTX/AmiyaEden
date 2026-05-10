import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchMyAlliancePAP } from '@/api/alliance-pap'
import { fetchMyPapLogs } from '@/api/fleet'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { AlliancePAPFleet, AlliancePAPSummary } from '@/types/api/alliance-pap'
import type { FleetPapLog } from '@/types/api/fleet'
import { formatDateTime, getErrorMessage, ShopBadge } from './shop-page-utils'

type ActiveTab = 'corporation' | 'alliance'

function importanceClass(value: string) {
  switch (value) {
    case 'cta':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'strat_op':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

function levelClass(value: string) {
  switch (value) {
    case 'cta':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'strat_op':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

function currentMonthValue() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

export function OperationPapPage() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<ActiveTab>('corporation')
  const [loading, setLoading] = useState(true)
  const [allianceLoading, setAllianceLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [papLogs, setPapLogs] = useState<FleetPapLog[]>([])
  const [papPage, setPapPage] = useState(1)
  const [papPageSize, setPapPageSize] = useState(50)
  const [allianceMonth, setAllianceMonth] = useState(currentMonthValue())
  const [allianceSummary, setAllianceSummary] = useState<AlliancePAPSummary | null>(null)
  const [allianceFleets, setAllianceFleets] = useState<AlliancePAPFleet[]>([])
  const [alliancePage, setAlliancePage] = useState(1)
  const [alliancePageSize, setAlliancePageSize] = useState(50)

  const papPageCount = useMemo(() => Math.max(1, Math.ceil(papLogs.length / papPageSize) || 1), [papLogs.length, papPageSize])
  const alliancePageCount = useMemo(
    () => Math.max(1, Math.ceil(allianceFleets.length / alliancePageSize) || 1),
    [allianceFleets.length, alliancePageSize]
  )
  const pagedPapLogs = useMemo(
    () => papLogs.slice((papPage - 1) * papPageSize, (papPage - 1) * papPageSize + papPageSize),
    [papLogs, papPage, papPageSize]
  )
  const pagedAllianceFleets = useMemo(
    () =>
      allianceFleets.slice(
        (alliancePage - 1) * alliancePageSize,
        (alliancePage - 1) * alliancePageSize + alliancePageSize
      ),
    [allianceFleets, alliancePage, alliancePageSize]
  )

  const loadPapLogs = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const logs = await fetchMyPapLogs()
      setPapLogs(logs ?? [])
      setPapPage(1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
      setPapLogs([])
    } finally {
      setLoading(false)
    }
  }, [t])

  const loadAlliancePAP = useCallback(async () => {
    setAllianceLoading(true)
    setError(null)
    try {
      const [year, month] = allianceMonth.split('-').map((value) => Number(value))
      const response = await fetchMyAlliancePAP({ year, month })
      setAllianceSummary(response.summary ?? null)
      setAllianceFleets(response.fleets ?? [])
      setAlliancePage(1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
      setAllianceSummary(null)
      setAllianceFleets([])
    } finally {
      setAllianceLoading(false)
    }
  }, [allianceMonth, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadPapLogs()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadPapLogs])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadAlliancePAP()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadAlliancePAP])

  const importanceLabel = (value: string) => {
    const key = `fleet.pap.importance.${value}`
    const translated = t(key)
    return translated === key ? value : translated
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant={activeTab === 'corporation' ? 'default' : 'outline'} onClick={() => setActiveTab('corporation')}>
            {t('fleet.pap.myTitle')}
          </Button>
          <Button type="button" variant={activeTab === 'alliance' ? 'default' : 'outline'} onClick={() => setActiveTab('alliance')}>
            {t('fleet.pap.allianceCard')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {activeTab === 'corporation' ? (
        <div className="rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h1 className="text-xl font-semibold">{t('fleet.pap.myTitle')}</h1>
              <p className="mt-1 text-sm text-muted-foreground">{t('fleet.pap.participations')}</p>
            </div>
            <Button type="button" variant="outline" onClick={() => void loadPapLogs()} disabled={loading}>
              {t('common.refresh')}
            </Button>
          </div>

          <div className="mt-4 flex gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {papLogs.reduce((sum, row) => sum + row.pap_count, 0)}
              </div>
              <div className="text-xs text-muted-foreground">{t('fleet.pap.totalPap')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-emerald-600">{papLogs.length}</div>
              <div className="text-xs text-muted-foreground">{t('fleet.pap.participations')}</div>
            </div>
          </div>

          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('fleet.pap.operation')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.level')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.character')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.ship')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.count')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.fc')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.issuedAt')}</th>
                </tr>
              </thead>
              <tbody>
                {pagedPapLogs.map((row) => (
                  <tr key={row.id} className="border-b">
                    <td className="px-3 py-2">{row.fleet_title || '-'}</td>
                    <td className="px-3 py-2">
                      <ShopBadge className={importanceClass(row.fleet_importance)}>
                        {importanceLabel(row.fleet_importance)}
                      </ShopBadge>
                    </td>
                    <td className="px-3 py-2">{row.character_name || row.character_id}</td>
                    <td className="px-3 py-2">{row.ship_type_id ?? '-'}</td>
                    <td className="px-3 py-2">
                      <ShopBadge className="bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
                        +{row.pap_count}
                      </ShopBadge>
                    </td>
                    <td className="px-3 py-2">{row.fc_character_name || row.issued_by}</td>
                    <td className="px-3 py-2">{formatDateTime(row.created_at)}</td>
                  </tr>
                ))}
                {!loading && papLogs.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={7}>
                      {t('fleet.pap.empty')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <span>
              {papPage}/{papPageCount}
            </span>
            <Button type="button" variant="outline" size="sm" onClick={() => setPapPage((current) => Math.max(1, current - 1))} disabled={papPage <= 1}>
              {t('welfareMy.pagination.prev')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setPapPage((current) => current + 1)}
              disabled={pagedPapLogs.length < papPageSize || papPage * papPageSize >= papLogs.length}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <select
              className="h-8 rounded-md border border-input bg-background px-2 text-sm"
              value={papPageSize}
              onChange={(event) => {
                setPapPageSize(Number(event.target.value))
                setPapPage(1)
              }}
            >
              {[50, 100, 200].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
        </div>
      ) : (
        <div className="rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="text-xl font-semibold">{t('fleet.pap.allianceCard')}</h1>
              <input
                type="month"
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={allianceMonth}
                onChange={(event) => setAllianceMonth(event.target.value)}
              />
            </div>
            <Button type="button" variant="outline" onClick={() => void loadAlliancePAP()} disabled={allianceLoading}>
              {t('common.refresh')}
            </Button>
          </div>

          {allianceSummary ? (
            <div className="mt-4 grid gap-3 md:grid-cols-3 xl:grid-cols-6">
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceMonthly')}</div>
                <div className="mt-1 text-xl font-semibold text-primary">{allianceSummary.total_pap}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceYearly')}</div>
                <div className="mt-1 text-xl font-semibold text-blue-500">{allianceSummary.yearly_total_pap}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceCorpMonthRank')}</div>
                <div className="mt-1 text-xl font-semibold text-emerald-600">
                  #{allianceSummary.monthly_rank}/{allianceSummary.total_in_corp}
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceGlobalMonthRank')}</div>
                <div className="mt-1 text-xl font-semibold text-yellow-500">
                  #{allianceSummary.global_monthly_rank}/{allianceSummary.total_global}
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceCorpYearRank')}</div>
                <div className="mt-1 text-xl font-semibold text-purple-500">#{allianceSummary.yearly_rank}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.pap.allianceLastCalc')}</div>
                <div className="mt-1 text-sm font-medium">{formatDateTime(allianceSummary.calculated_at)}</div>
              </div>
            </div>
          ) : null}

          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('fleet.pap.allianceOperationName')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.character')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.level')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.count')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.ship')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.allianceStartTime')}</th>
                  <th className="px-3 py-2">{t('fleet.pap.allianceEndTime')}</th>
                </tr>
              </thead>
              <tbody>
                {pagedAllianceFleets.map((row) => (
                  <tr key={row.id} className="border-b">
                    <td className="px-3 py-2">{row.title}</td>
                    <td className="px-3 py-2">{row.character_name}</td>
                    <td className="px-3 py-2">
                      <ShopBadge className={levelClass(row.level)}>{row.level}</ShopBadge>
                    </td>
                    <td className="px-3 py-2">
                      <ShopBadge className="bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
                        {row.pap}
                      </ShopBadge>
                    </td>
                    <td className="px-3 py-2">
                      {row.ship_type_name}
                      <span className="ml-1 text-xs text-muted-foreground">({row.ship_group_name})</span>
                    </td>
                    <td className="px-3 py-2">{formatDateTime(row.start_at)}</td>
                    <td className="px-3 py-2">{row.end_at ? formatDateTime(row.end_at) : '-'}</td>
                  </tr>
                ))}
                {!allianceLoading && allianceFleets.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={7}>
                      {t('fleet.pap.allianceEmpty')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <span>
              {alliancePage}/{alliancePageCount}
            </span>
            <Button type="button" variant="outline" size="sm" onClick={() => setAlliancePage((current) => Math.max(1, current - 1))} disabled={alliancePage <= 1}>
              {t('welfareMy.pagination.prev')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setAlliancePage((current) => current + 1)}
              disabled={pagedAllianceFleets.length < alliancePageSize || alliancePage * alliancePageSize >= allianceFleets.length}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <select
              className="h-8 rounded-md border border-input bg-background px-2 text-sm"
              value={alliancePageSize}
              onChange={(event) => {
                setAlliancePageSize(Number(event.target.value))
                setAlliancePage(1)
              }}
            >
              {[50, 100, 200].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
        </div>
      )}
    </section>
  )
}
