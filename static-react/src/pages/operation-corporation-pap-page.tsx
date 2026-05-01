import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchCorporationPapSummary } from '@/api/fleet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type {
  CorporationPapOverview,
  CorporationPapSummaryItem,
  PapSummaryPeriod,
} from '@/types/api/fleet'
import { getErrorMessage, ShopBadge } from './shop-page-utils'

const defaultTickers = ['FUXI', 'FMA.1']

function periodClass(value: string) {
  switch (value) {
    case 'current_month':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'last_month':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'at_year':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

export function OperationCorporationPapPage() {
  const { t } = useI18n()
  const currentYear = useMemo(() => new Date().getFullYear(), [])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [records, setRecords] = useState<CorporationPapSummaryItem[]>([])
  const [overview, setOverview] = useState<CorporationPapOverview>({
    filtered_pap_total: 0,
    filtered_strat_op_total: 0,
    all_pap_total: 0,
    filtered_user_count: 0,
    period: 'last_month',
  })
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)
  const [total, setTotal] = useState(0)
  const [period, setPeriod] = useState<PapSummaryPeriod>('last_month')
  const [year, setYear] = useState(currentYear)
  const [tickerText, setTickerText] = useState(defaultTickers.join(', '))

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const tickerList = useMemo(
    () =>
      Array.from(
        new Set(
          tickerText
            .split(',')
            .map((ticker) => ticker.trim())
            .filter(Boolean)
        )
      ),
    [tickerText]
  )

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchCorporationPapSummary({
        current: page,
        size: pageSize,
        period,
        year: period === 'at_year' ? year : undefined,
        corp_tickers: tickerList.join(','),
      })
      setRecords(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
      setOverview(
        response.overview ?? {
          filtered_pap_total: 0,
          filtered_strat_op_total: 0,
          all_pap_total: 0,
          filtered_user_count: 0,
          period,
        }
      )
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.corporationPap.loadFailed')))
      setRecords([])
      setTotal(0)
      setOverview({
        filtered_pap_total: 0,
        filtered_strat_op_total: 0,
        all_pap_total: 0,
        filtered_user_count: 0,
        period,
      })
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, period, t, tickerList, year])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData])

  const stats = useMemo(
    () => [
      {
        label: t('fleet.corporationPap.stats.filteredPap'),
        value: overview.filtered_pap_total,
      },
      {
        label: t('fleet.corporationPap.stats.filteredStratOp'),
        value: overview.filtered_strat_op_total,
      },
      {
        label: t('fleet.corporationPap.stats.allPap'),
        value: overview.all_pap_total,
      },
      {
        label: t('fleet.corporationPap.stats.users'),
        value: overview.filtered_user_count,
      },
    ],
    [overview, t]
  )

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('fleet.corporationPap.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('fleet.corporationPap.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('fleet.corporationPap.filters.period')}</span>
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={period}
                onChange={(event) => {
                  setPeriod(event.target.value as PapSummaryPeriod)
                  setPage(1)
                }}
              >
                <option value="current_month">{t('fleet.corporationPap.periods.currentMonth')}</option>
                <option value="last_month">{t('fleet.corporationPap.periods.lastMonth')}</option>
                <option value="at_year">{t('fleet.corporationPap.periods.atYear')}</option>
                <option value="all">{t('fleet.corporationPap.periods.all')}</option>
              </select>
            </label>
            {period === 'at_year' ? (
              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('fleet.corporationPap.filters.year')}</span>
                <Input
                  type="number"
                  min={2003}
                  max={2100}
                  value={String(year)}
                  onChange={(event) => setYear(Number(event.target.value))}
                />
              </label>
            ) : null}
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('fleet.corporationPap.filters.corpTickers')}</span>
              <Input value={tickerText} onChange={(event) => setTickerText(event.target.value)} />
            </label>
            <Button type="button" variant="outline" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('fleet.corporationPap.loading')}</p> : null}

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => (
          <div key={stat.label} className="rounded-lg border bg-card p-4">
            <div className="text-xs text-muted-foreground">{stat.label}</div>
            <div className="mt-2 text-2xl font-semibold">{stat.value}</div>
          </div>
        ))}
      </div>

      <div className="rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('fleet.corporationPap.summaryTitle')}</div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.corpTicker')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.nickname')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.mainCharacter')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.characterCount')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.stratOpPaps')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.skirmishPaps')}</th>
                <th className="px-3 py-2">{t('fleet.corporationPap.columns.allianceStratPaps')}</th>
              </tr>
            </thead>
            <tbody>
              {records.map((row) => (
                <tr key={row.user_id} className="border-b">
                  <td className="px-3 py-2">
                    <ShopBadge className={periodClass(period)}>{row.corp_ticker}</ShopBadge>
                  </td>
                  <td className="px-3 py-2">{row.nickname || '-'}</td>
                  <td className="px-3 py-2">{row.main_character_name || '-'}</td>
                  <td className="px-3 py-2">{row.character_count}</td>
                  <td className="px-3 py-2">{row.strat_op_paps}</td>
                  <td className="px-3 py-2">{row.skirmish_paps}</td>
                  <td className="px-3 py-2">{row.alliance_strat_paps}</td>
                </tr>
              ))}
              {!loading && records.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={7}>
                    {t('fleet.corporationPap.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button type="button" variant="outline" size="sm" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          disabled={records.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={pageSize}
            onChange={(event) => {
              setPageSize(Number(event.target.value))
              setPage(1)
            }}
          >
            {[20, 50, 100, 200].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
    </section>
  )
}
