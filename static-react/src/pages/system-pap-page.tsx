import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { fetchAllAlliancePAP, importAlliancePAP, settleAlliancePAPMonth, triggerAlliancePAPFetch } from '@/api/alliance-pap'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import type { AlliancePAPImportInfo, AlliancePAPSummary } from '@/types/api/alliance-pap'
import { ShopDialog, formatDateTime } from './shop-page-utils'

const currentMonth = new Date().toISOString().slice(0, 7)

type ImportDraft = {
  content: string
}

export function SystemPAPPage() {
  const { t } = useI18n()
  const [month, setMonth] = useState(currentMonth)
  const [loading, setLoading] = useState(true)
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [rows, setRows] = useState<AlliancePAPSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [fetching, setFetching] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [importSaving, setImportSaving] = useState(false)
  const [importDraft, setImportDraft] = useState<ImportDraft>({ content: '' })
  const [settling, setSettling] = useState(false)

  const [year, monthNumber] = useMemo(() => {
    const [yearStr, monthStr] = month.split('-')
    return [Number(yearStr), Number(monthStr)] as const
  }, [month])

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchAllAlliancePAP({ current: page, size: pageSize, year, month: monthNumber })
      setRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch {
      setError(t('alliancePap.messages.loadFailed'))
      setRows([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [monthNumber, page, pageSize, t, year])

  useEffect(() => {
    let active = true
    void (async () => {
      if (!active) {
        return
      }
      await loadData()
    })()

    return () => {
      active = false
    }
  }, [loadData, refreshSeed])

  const handleFetch = async () => {
    setFetching(true)
    try {
      await triggerAlliancePAPFetch({ year, month: monthNumber })
      notifySuccess(t('alliancePap.messages.fetchTriggered'))
      setRefreshSeed((current) => current + 1)
    } catch {
      notifyError(t('alliancePap.messages.fetchFailed'))
    } finally {
      setFetching(false)
    }
  }

  const handleSettle = async () => {
    if (!window.confirm(t('alliancePap.messages.settleConfirm', { month }))) {
      return
    }

    setSettling(true)
    try {
      await settleAlliancePAPMonth({ year, month: monthNumber })
      notifySuccess(t('alliancePap.messages.settleSuccess'))
      setRefreshSeed((current) => current + 1)
    } catch {
      notifyError(t('alliancePap.messages.settleFailed'))
    } finally {
      setSettling(false)
    }
  }

  const handleImport = async () => {
    const rowsToImport = parseImportRows(importDraft.content)
    if (rowsToImport.length === 0) {
      notifyError(t('alliancePap.messages.importNoData'))
      return
    }

    setImportSaving(true)
    let success = 0
    try {
      for (const item of rowsToImport) {
        await importAlliancePAP({
          year,
          month: monthNumber,
          data: item,
        })
        success += 1
      }
      notifySuccess(t('alliancePap.messages.importSuccess', { count: success }))
      setImportOpen(false)
      setImportDraft({ content: '' })
      setRefreshSeed((current) => current + 1)
    } catch {
      notifyError(t('alliancePap.messages.importFailed'))
    } finally {
      setImportSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('alliancePap.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('alliancePap.subtitle')}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={() => setImportOpen(true)}>
              {t('alliancePap.actions.import')}
            </Button>
            <Button type="button" variant="outline" onClick={() => void handleFetch()} disabled={fetching}>
              {fetching ? t('alliancePap.messages.fetching') : t('alliancePap.actions.fetch')}
            </Button>
            <Button type="button" onClick={() => void handleSettle()} disabled={settling}>
              {settling ? t('alliancePap.messages.settling') : t('alliancePap.actions.settle')}
            </Button>
          </div>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('alliancePap.filters.month')}</span>
            <Input
              type="month"
              value={month}
              onChange={(event) => {
                setMonth(event.target.value)
                setPage(1)
              }}
            />
          </label>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('alliancePap.messages.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('alliancePap.table.title')}</div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('alliancePap.columns.mainCharacter')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.monthlyPap')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.yearlyPap')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.corpMonthlyRank')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.allianceMonthlyRank')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.corpYearlyRank')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.allianceYearlyRank')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.calculatedAt')}</th>
                <th className="px-3 py-2">{t('alliancePap.columns.status')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={row.id} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2 font-medium">{row.main_character}</td>
                  <td className="px-3 py-2">{row.total_pap}</td>
                  <td className="px-3 py-2">{row.yearly_total_pap}</td>
                  <td className="px-3 py-2">
                    <span className="font-medium text-emerald-600 dark:text-emerald-400">
                      #{row.monthly_rank}
                    </span>
                    <span className="text-xs text-muted-foreground"> / {row.total_in_corp}</span>
                  </td>
                  <td className="px-3 py-2">
                    <span className="font-medium text-amber-600 dark:text-amber-400">
                      #{row.global_monthly_rank}
                    </span>
                    <span className="text-xs text-muted-foreground"> / {row.total_global}</span>
                  </td>
                  <td className="px-3 py-2">
                    <span className="font-medium text-violet-600 dark:text-violet-400">
                      #{row.yearly_rank}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <span className="font-medium text-sky-600 dark:text-sky-400">
                      #{row.global_yearly_rank}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatDateTime(row.calculated_at)}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                        row.is_archived
                          ? 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
                          : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                      }`}
                    >
                      {row.is_archived ? t('alliancePap.status.archived') : t('alliancePap.status.current')}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          disabled={page <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => current + 1)}
          disabled={page >= pageCount}
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
            {[10, 20, 50].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>

      <ShopDialog
        open={importOpen}
        title={t('alliancePap.import.title')}
        onClose={() => setImportOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-2xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setImportOpen(false)} disabled={importSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void handleImport()} disabled={importSaving}>
              {importSaving ? t('alliancePap.messages.importing') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">{t('alliancePap.import.hint')}</p>
          <textarea
            className="min-h-56 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
            value={importDraft.content}
            onChange={(event) => setImportDraft({ content: event.target.value })}
            placeholder={t('alliancePap.import.placeholder')}
          />
        </div>
      </ShopDialog>
    </section>
  )
}

function parseImportRows(content: string): AlliancePAPImportInfo[] {
  return content
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .map((line) => {
      const parts = line.split(/[,\t]/).map((part) => part.trim())
      return {
        primary_character_name: parts[0] ?? '',
        monthly_pap: Number(parts[1] ?? 0),
        calculated_at: parts[2] ?? '',
      }
    })
    .filter((item) => item.primary_character_name !== '' && item.calculated_at !== '')
}
