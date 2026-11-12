import { useEffect, useMemo, useState } from 'react'
import { fetchBlacklistEntries, fetchBlacklistSummary, refreshBlacklist } from '@/api/blacklist'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import type {
  BlacklistEntry,
  BlacklistKind,
  BlacklistListParams,
  BlacklistRefreshSummary,
} from '@/types/api/blacklist'

const kinds: BlacklistKind[] = ['character', 'qq', 'discord']

// Backend errors are i18n keys; anything else (network, timeout) maps to the generic key.
function errorKey(error: unknown) {
  return error instanceof Error && error.message.startsWith('blacklist.errors.')
    ? error.message
    : 'blacklist.errors.internal'
}

interface ListResult {
  request: BlacklistListParams
  list: BlacklistEntry[]
  total: number
  error: string | null
}

export function SystemBlacklistPage() {
  const { t } = useI18n()
  const [kindDraft, setKindDraft] = useState('')
  const [keywordDraft, setKeywordDraft] = useState('')
  const [filters, setFilters] = useState<Pick<BlacklistListParams, 'kind' | 'keyword'>>({})
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [result, setResult] = useState<ListResult | null>(null)
  // undefined: still loading; null: never refreshed.
  const [summary, setSummary] = useState<BlacklistRefreshSummary | null | undefined>(undefined)
  const [refreshing, setRefreshing] = useState(false)
  const [now, setNow] = useState(Date.now)
  const nextRefreshAt = summary?.next_refresh_at
  const coolingDown = !!nextRefreshAt && new Date(nextRefreshAt).getTime() > now

  useEffect(() => {
    if (!nextRefreshAt) return
    const timer = window.setTimeout(
      () => setNow(Date.now()),
      Math.max(0, new Date(nextRefreshAt).getTime() - Date.now()) + 1
    )
    return () => window.clearTimeout(timer)
  }, [nextRefreshAt])

  // A new filters object (search or refresh) refetches even when the values are unchanged.
  const request = useMemo<BlacklistListParams>(
    () => ({ current: page, size: pageSize, ...filters }),
    [page, pageSize, filters]
  )
  const loading = result?.request !== request

  useEffect(() => {
    let active = true
    fetchBlacklistEntries(request)
      .then((value) => {
        if (active)
          setResult({ request, list: value.list ?? [], total: value.total ?? 0, error: null })
      })
      .catch((error: unknown) => {
        if (active) setResult({ request, list: [], total: 0, error: errorKey(error) })
      })
    return () => {
      active = false
    }
  }, [request])

  useEffect(() => {
    let active = true
    fetchBlacklistSummary()
      .then((value) => {
        if (active) setSummary(value ?? null)
      })
      .catch((error: unknown) => {
        if (active) notifyError(t(errorKey(error)))
      })
    return () => {
      active = false
    }
  }, [t])

  function search(event: React.FormEvent) {
    event.preventDefault()
    setFilters({
      kind: (kindDraft || undefined) as BlacklistKind | undefined,
      keyword: keywordDraft.trim() || undefined,
    })
    setPage(1)
  }

  async function refresh() {
    setRefreshing(true)
    try {
      const value = await refreshBlacklist()
      setSummary(value)
      const message = t('blacklist.refreshSuccess', {
        added: value.added,
        updated: value.updated,
        removed: value.removed,
      })
      notifySuccess(
        value.lookup_failed > 0
          ? `${message}\n${t('blacklist.lookupFailed', { count: value.lookup_failed })}`
          : message
      )
      setFilters((current) => ({ ...current }))
    } catch (error) {
      notifyError(t(errorKey(error)))
    } finally {
      setRefreshing(false)
    }
  }

  const columns = useMemo<ColumnDef<BlacklistEntry>[]>(
    () => [
      {
        accessorKey: 'kind',
        header: t('blacklist.columns.kind'),
        size: 120,
        cell: ({ row }) => (
          <span className="inline-flex rounded-md border border-destructive/40 px-2 py-0.5 text-xs text-destructive">
            {t(`blacklist.kinds.${row.original.kind}`)}
          </span>
        ),
      },
      {
        accessorKey: 'display',
        header: t('blacklist.columns.value'),
        cell: ({ row }) =>
          row.original.kind === 'character'
            ? t('blacklist.characterValue', { name: row.original.display, id: row.original.value })
            : row.original.display,
      },
      {
        accessorKey: 'reason',
        header: t('blacklist.columns.reason'),
        cell: ({ row }) => (
          <span className="whitespace-pre-line">{row.original.reason || '-'}</span>
        ),
      },
    ],
    [t]
  )

  return (
    <section className="min-w-0 space-y-4">
      <Card>
        <CardHeader className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle>{t('blacklist.summary.title')}</CardTitle>
          <Button isDisabled={refreshing || coolingDown} onClick={() => void refresh()}>
            {t(refreshing ? 'blacklist.refreshing' : 'blacklist.refresh')}
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground">{t('blacklist.description')}</p>
          {summary ? (
            <dl className="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-[auto_1fr]">
              <dt className="text-muted-foreground">{t('blacklist.summary.finishedAt')}</dt>
              <dd>{formatTime(summary.finished_at)}</dd>
              {coolingDown && nextRefreshAt ? (
                <>
                  <dt className="text-muted-foreground">{t('blacklist.summary.nextRefreshAt')}</dt>
                  <dd>{formatTime(nextRefreshAt)}</dd>
                </>
              ) : null}
              <dt className="text-muted-foreground">{t('blacklist.summary.operator')}</dt>
              <dd>{summary.operator_name || '-'}</dd>
              <dt className="text-muted-foreground">{t('blacklist.summary.totals')}</dt>
              <dd>
                {t('blacklist.summary.totalsValue', {
                  characters: summary.characters,
                  qqs: summary.qqs,
                  discords: summary.discords,
                })}
              </dd>
              <dt className="text-muted-foreground">{t('blacklist.summary.changes')}</dt>
              <dd>
                {t('blacklist.summary.changesValue', {
                  added: summary.added,
                  updated: summary.updated,
                  removed: summary.removed,
                })}
              </dd>
              <dt className="text-muted-foreground">{t('blacklist.summary.unresolved')}</dt>
              <dd>
                {t('blacklist.summary.unresolvedValue', {
                  added: summary.new_unresolved,
                  skipped: summary.skipped_unresolved,
                })}
              </dd>
              {summary.lookup_failed > 0 ? (
                <>
                  <dt className="text-muted-foreground">{t('blacklist.summary.lookupFailed')}</dt>
                  <dd>{t('blacklist.lookupFailed', { count: summary.lookup_failed })}</dd>
                </>
              ) : null}
            </dl>
          ) : summary === null ? (
            <p className="text-sm text-muted-foreground">{t('blacklist.summary.never')}</p>
          ) : null}
        </CardContent>
      </Card>

      <form className="flex flex-wrap items-end gap-3" onSubmit={search}>
        <Select
          aria-label={t('blacklist.filters.kind')}
          selectedKey={kindDraft}
          onSelectionChange={(key) => setKindDraft(String(key ?? ''))}
        >
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem id="">{t('blacklist.kinds.all')}</SelectItem>
            {kinds.map((kind) => (
              <SelectItem key={kind} id={kind}>
                {t(`blacklist.kinds.${kind}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          aria-label={t('blacklist.filters.keyword')}
          placeholder={t('blacklist.filters.keyword')}
          className="w-full sm:w-96"
          value={keywordDraft}
          onChange={(event) => setKeywordDraft(event.target.value)}
        />
        <Button type="submit">{t('common.search')}</Button>
      </form>

      <DataTable
        columns={columns}
        data={result?.list ?? []}
        getRowId={(entry) => `${entry.kind}:${entry.value}`}
        loading={loading}
        error={result?.error ? t(result.error) : null}
        loadingText={t('blacklist.loading')}
        emptyText={t('blacklist.empty')}
        pagination={{
          page,
          pageSize,
          total: result?.total ?? 0,
          onPageChange: setPage,
          onPageSizeChange: (next) => {
            setPageSize(next)
            setPage(1)
          },
          previousLabel: t('welfareMy.pagination.prev'),
          nextLabel: t('welfareMy.pagination.next'),
          pageSizeLabel: t('welfareMy.pageSize'),
        }}
      />
    </section>
  )
}
