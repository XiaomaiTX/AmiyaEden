import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { fetchAuditEvents } from '@/api/audit'
import { ShopDialog, formatDateTime, getErrorMessage } from './shop-page-utils'
import { useI18n } from '@/i18n'
import type { AuditEvent, AuditEventSearchParams } from '@/types/api/audit'

type AuditTab = 'events' | 'exports'

const defaultSearchDraft: AuditEventSearchParams = {
  current: 1,
  size: 20,
  start_date: '',
  end_date: '',
  category: '',
  action: '',
  actor_user_id: undefined,
  target_user_id: undefined,
  result: undefined,
  request_id: '',
  resource_id: '',
  keyword: '',
}

const categoryOptions = ['permission', 'fuxi_wallet', 'config', 'approval', 'task_ops', 'security'] as const

export function SystemAuditPage() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<AuditTab>('events')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rows, setRows] = useState<AuditEvent[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [searchDraft, setSearchDraft] = useState<AuditEventSearchParams>(defaultSearchDraft)
  const [searchState, setSearchState] = useState<AuditEventSearchParams>(defaultSearchDraft)
  const [detailOpen, setDetailOpen] = useState(false)
  const [currentEvent, setCurrentEvent] = useState<AuditEvent | null>(null)

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchAuditEvents({
        ...searchState,
        current: page,
        size: pageSize,
      })
      setRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('auditAdmin.messages.loadFailed')))
      setRows([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, searchState, t])

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
  }, [loadData])

  const applySearch = () => {
    setPage(1)
    setSearchState({
      ...searchDraft,
      actor_user_id: normalizeNumber(searchDraft.actor_user_id),
      target_user_id: normalizeNumber(searchDraft.target_user_id),
    })
  }

  const resetSearch = () => {
    setSearchDraft(defaultSearchDraft)
    setSearchState(defaultSearchDraft)
    setPage(1)
  }

  const openDetail = (row: AuditEvent) => {
    setCurrentEvent(row)
    setDetailOpen(true)
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('auditAdmin.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('auditAdmin.subtitle')}</p>
      </div>

      <div className="flex flex-wrap gap-2 rounded-lg border bg-card p-2">
        {([
          ['events', t('auditAdmin.tabs.events')],
          ['exports', t('auditAdmin.tabs.exports')],
        ] as const).map(([key, label]) => (
          <Button
            key={key}
            type="button"
            variant={activeTab === key ? 'default' : 'outline'}
            onClick={() => setActiveTab(key)}
          >
            {label}
          </Button>
        ))}
      </div>

      {activeTab === 'events' ? (
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-4">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.startDate')}</span>
                <Input
                  type="date"
                  value={searchDraft.start_date ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, start_date: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.endDate')}</span>
                <Input
                  type="date"
                  value={searchDraft.end_date ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, end_date: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.category')}</span>
                <select
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={searchDraft.category ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, category: event.target.value }))
                  }
                >
                  <option value="">{t('common.all')}</option>
                  {categoryOptions.map((value) => (
                    <option key={value} value={value}>
                      {t(`auditAdmin.categories.${value}`)}
                    </option>
                  ))}
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.result')}</span>
                <select
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={searchDraft.result ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({
                      ...current,
                      result: event.target.value ? (event.target.value as 'success' | 'failed') : undefined,
                    }))
                  }
                >
                  <option value="">{t('common.all')}</option>
                  <option value="success">{t('auditAdmin.results.success')}</option>
                  <option value="failed">{t('auditAdmin.results.failed')}</option>
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.action')}</span>
                <Input
                  value={searchDraft.action ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, action: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.actorUserId')}</span>
                <Input
                  type="number"
                  value={searchDraft.actor_user_id ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({
                      ...current,
                      actor_user_id: event.target.value ? Number(event.target.value) : undefined,
                    }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.targetUserId')}</span>
                <Input
                  type="number"
                  value={searchDraft.target_user_id ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({
                      ...current,
                      target_user_id: event.target.value ? Number(event.target.value) : undefined,
                    }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.keyword')}</span>
                <Input
                  value={searchDraft.keyword ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, keyword: event.target.value }))
                  }
                  placeholder={t('auditAdmin.placeholders.keyword')}
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.requestId')}</span>
                <Input
                  value={searchDraft.request_id ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, request_id: event.target.value }))
                  }
                  placeholder={t('auditAdmin.placeholders.requestId')}
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('auditAdmin.filters.resourceId')}</span>
                <Input
                  value={searchDraft.resource_id ?? ''}
                  onChange={(event) =>
                    setSearchDraft((current) => ({ ...current, resource_id: event.target.value }))
                  }
                />
              </label>
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              <Button type="button" onClick={applySearch}>
                {t('common.search')}
              </Button>
              <Button type="button" variant="outline" onClick={resetSearch}>
                {t('common.reset')}
              </Button>
            </div>
          </div>

          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('auditAdmin.tabs.events')}</div>
            <div className="overflow-x-auto">
              {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
              {loading ? (
                <p className="px-4 py-3 text-sm text-muted-foreground">{t('auditAdmin.messages.loading')}</p>
              ) : null}
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">#</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.time')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.category')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.action')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.actorUserId')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.targetUserId')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.result')}</th>
                    <th className="px-3 py-2">{t('auditAdmin.columns.requestId')}</th>
                    <th className="px-3 py-2">{t('common.operation')}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, index) => (
                    <tr key={row.event_id} className="border-b align-top">
                      <td className="px-3 py-2">{index + 1}</td>
                      <td className="px-3 py-2">{formatDateTime(row.occurred_at)}</td>
                      <td className="px-3 py-2">{getAuditCategoryLabel(t, row.category)}</td>
                      <td className="px-3 py-2">{row.action}</td>
                      <td className="px-3 py-2">{row.actor_user_id || '-'}</td>
                      <td className="px-3 py-2">{row.target_user_id || '-'}</td>
                      <td className="px-3 py-2">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                            row.result === 'success'
                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                              : 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
                          }`}
                        >
                          {t(`auditAdmin.results.${row.result}`)}
                        </span>
                      </td>
                      <td className="px-3 py-2">{row.request_id || '-'}</td>
                      <td className="px-3 py-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => openDetail(row)}>
                          {t('auditAdmin.actions.detail')}
                        </Button>
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
        </div>
      ) : (
        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-base font-semibold">{t('auditAdmin.tabs.exports')}</h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('auditAdmin.export.unavailableDescription')}</p>
        </div>
      )}

      <ShopDialog
        open={detailOpen}
        title={t('auditAdmin.detailTitle')}
        onClose={() => setDetailOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-3xl"
      >
        {currentEvent ? (
          <div className="space-y-4 text-sm">
            <div className="grid gap-3 md:grid-cols-2">
              <KeyValue label={t('auditAdmin.columns.eventId')} value={currentEvent.event_id} />
              <KeyValue label={t('auditAdmin.columns.time')} value={formatDateTime(currentEvent.occurred_at)} />
              <KeyValue label={t('auditAdmin.columns.category')} value={currentEvent.category} />
              <KeyValue label={t('auditAdmin.columns.action')} value={currentEvent.action} />
              <KeyValue label={t('auditAdmin.columns.actorUserId')} value={currentEvent.actor_user_id || '-'} />
              <KeyValue label={t('auditAdmin.columns.targetUserId')} value={currentEvent.target_user_id || '-'} />
              <KeyValue label={t('auditAdmin.columns.requestId')} value={currentEvent.request_id || '-'} />
              <KeyValue label={t('auditAdmin.columns.resourceId')} value={currentEvent.resource_id || '-'} />
              <KeyValue label={t('auditAdmin.columns.ip')} value={currentEvent.ip || '-'} />
              <KeyValue label={t('auditAdmin.columns.userAgent')} value={currentEvent.user_agent || '-'} />
            </div>

            <div className="rounded-lg border bg-muted/20 p-4">
              <div className="text-sm font-medium">{t('auditAdmin.columns.details')}</div>
              <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words text-xs leading-6">
                {prettyDetails(currentEvent.details_json)}
              </pre>
            </div>
          </div>
        ) : null}
      </ShopDialog>
    </section>
  )
}

function normalizeNumber(value: number | undefined) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined
}

function prettyDetails(raw: string) {
  if (!raw) {
    return '{}'
  }

  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

function getAuditCategoryLabel(
  t: (key: string, vars?: Record<string, string | number>) => string,
  category: string
) {
  const key = `auditAdmin.categories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}

function KeyValue({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-muted/20 px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 break-all text-sm font-medium">{value}</div>
    </div>
  )
}
