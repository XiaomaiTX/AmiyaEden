import { useEffect, useMemo, useState } from 'react'
import {
  listFleetQQPingResults,
  sendFleetQQPing,
  type FleetQQPingPage,
  type FleetQQPingTask,
} from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { Button } from '@/components/ui/button'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { useI18n } from '@/i18n'
import { createFleetQQPingIntent, fleetQQPingErrorKey } from '@/lib/fleet-qq-ping'
import { qqStateLabel, shouldPollQQTask } from '@/lib/qq-official'
import { formatDateTime } from './shop-page-utils'

const POLL_MS = 5000

// The fleet history endpoint caps page sizes at 100 instead of the shared ledger default of 200.
const firstPage = { current: 1, size: 100 }

type HistoryResponse = {
  query: typeof firstPage
  data: FleetQQPingPage | null
}

export function FleetQQPingPanel({ fleetId }: { fleetId: string }) {
  const { t } = useI18n()
  const [intent] = useState(createFleetQQPingIntent)
  const [sending, setSending] = useState(false)
  const [queuedCount, setQueuedCount] = useState<number | null>(null)
  const [queueError, setQueueError] = useState('')
  const [query, setQuery] = useState(firstPage)
  const [response, setResponse] = useState<HistoryResponse>()
  const history = response?.query === query ? response : undefined

  useEffect(() => {
    const controller = new AbortController()
    let timer: number | undefined

    function schedulePoll() {
      timer = window.setTimeout(() => {
        if (document.hidden) schedulePoll()
        else void load()
      }, POLL_MS)
    }

    async function load() {
      try {
        const data = await unwrapContract(
          listFleetQQPingResults(fleetId, query, { signal: controller.signal })
        )
        if (controller.signal.aborted) return
        setResponse({ query, data })
        if (data.list.some((row) => shouldPollQQTask(row.status))) schedulePoll()
      } catch {
        if (!controller.signal.aborted) setResponse({ query, data: null })
      }
    }

    void load()
    return () => {
      controller.abort()
      window.clearTimeout(timer)
    }
  }, [fleetId, query])

  const columns = useMemo<ColumnDef<FleetQQPingTask>[]>(
    () => [
      {
        accessorKey: 'status',
        header: t('common.status'),
        cell: ({ row }) => qqStateLabel(t, row.original.status),
      },
      {
        accessorKey: 'reason',
        header: t('qqOfficial.reason'),
        cell: ({ row }) => qqStateLabel(t, row.original.reason),
      },
      {
        accessorKey: 'created_at',
        header: t('qqOfficial.created'),
        cell: ({ row }) => formatDateTime(row.original.created_at),
      },
      {
        accessorKey: 'finished_at',
        header: t('qqOfficial.finished'),
        cell: ({ row }) => formatDateTime(row.original.finished_at),
      },
    ],
    [t]
  )

  async function queue() {
    const request = intent.submit((idempotency_key) =>
      unwrapContract(sendFleetQQPing(fleetId, { idempotency_key }))
    )
    if (!request) return
    setSending(true)
    setQueueError('')
    try {
      const receipt = await request
      setQueuedCount(receipt.tasks.length)
      setQuery((previous) => ({ ...previous, current: 1 }))
    } catch (error) {
      setQueueError(fleetQQPingErrorKey(error))
    } finally {
      setSending(false)
    }
  }

  return (
    <section
      aria-label={t('fleet.qqPing.results')}
      className="space-y-3 rounded-lg border bg-card p-5"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">{t('fleet.qqPing.results')}</h2>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            isDisabled={!history}
            onClick={() => setQuery((previous) => ({ ...previous }))}
          >
            {t('fleet.qqPing.reload')}
          </Button>
          <Button
            type="button"
            size="sm"
            isDisabled={sending || queuedCount !== null}
            onClick={() => void queue()}
          >
            {t(queueError ? 'fleet.qqPing.retry' : 'fleet.qqPing.queue')}
          </Button>
          {queuedCount !== null ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => {
                if (intent.resetAfterAcceptance()) setQueuedCount(null)
              }}
            >
              {t('fleet.qqPing.prepareAnother')}
            </Button>
          ) : null}
        </div>
      </div>
      {queuedCount !== null ? (
        <p className="text-sm text-muted-foreground" role="status">
          {t('fleet.qqPing.queued', { count: queuedCount })}
        </p>
      ) : null}
      {queueError ? (
        <p className="text-sm text-destructive" role="alert">
          {t(queueError)}
        </p>
      ) : null}
      <DataTable
        columns={columns}
        data={history?.data?.list ?? []}
        getRowId={(row) => row.id}
        variant="ledger"
        loading={!history}
        error={history?.data === null ? t('fleet.qqPing.loadFailed') : null}
        loadingText={t('common.loading')}
        emptyText={t('fleet.qqPing.empty')}
        pagination={{
          page: history?.data?.page ?? query.current,
          pageSize: history?.data?.pageSize ?? query.size,
          total: history?.data?.total ?? 0,
          onPageChange: (current) => setQuery((previous) => ({ ...previous, current })),
          onPageSizeChange: (size) => setQuery({ current: 1, size }),
          pageSizeOptions: [20, 50, 100],
          previousLabel: t('welfareMy.pagination.prev'),
          nextLabel: t('welfareMy.pagination.next'),
          pageSizeLabel: t('welfareMy.pageSize'),
        }}
      />
    </section>
  )
}
