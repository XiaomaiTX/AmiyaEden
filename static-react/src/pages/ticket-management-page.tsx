import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { adminListTickets, adminUpdateTicketPriority, adminUpdateTicketStatus } from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { TicketItem, TicketPriority, TicketStatus } from '@/types/api/ticket'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string | undefined) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function statusTone(status: TicketStatus) {
  switch (status) {
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'in_progress':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'completed':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function priorityTone(priority: TicketPriority) {
  switch (priority) {
    case 'high':
      return 'bg-red-100 text-red-700 dark:bg-red-500/10 dark:text-red-300'
    case 'medium':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'low':
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

export function TicketManagementPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tickets, setTickets] = useState<TicketItem[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TicketStatus | ''>('')
  const [updatingId, setUpdatingId] = useState<number | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const response = await adminListTickets({
          current: page,
          size: pageSize,
          keyword: keyword.trim() || undefined,
          status,
        })

        if (cancelled) return

        setTickets(response.list ?? [])
        setTotal(response.total ?? 0)
        setPage(response.page ?? page)
        setPageSize(response.pageSize ?? pageSize)
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('ticketManagement.loadFailed')))
          setTickets([])
          setTotal(0)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadData()

    return () => {
      cancelled = true
    }
  }, [keyword, page, pageSize, refreshSeed, status, t])

  const pageCount = Math.max(1, Math.ceil(total / pageSize) || 1)

  const refresh = () => setRefreshSeed((current) => current + 1)

  const updateTicketStatus = async (ticketId: number, nextStatus: TicketStatus) => {
    setUpdatingId(ticketId)
    try {
      await adminUpdateTicketStatus(ticketId, { status: nextStatus })
      setTickets((current) =>
        current.map((ticket) => (ticket.id === ticketId ? { ...ticket, status: nextStatus } : ticket))
      )
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketManagement.updateFailed')))
    } finally {
      setUpdatingId(null)
    }
  }

  const updateTicketPriority = async (ticketId: number, nextPriority: TicketPriority) => {
    setUpdatingId(ticketId)
    try {
      await adminUpdateTicketPriority(ticketId, { priority: nextPriority })
      setTickets((current) =>
        current.map((ticket) =>
          ticket.id === ticketId ? { ...ticket, priority: nextPriority } : ticket
        )
      )
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketManagement.updateFailed')))
    } finally {
      setUpdatingId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('ticketManagement.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('ticketManagement.subtitle')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('ticketManagement.filters.keyword')}</span>
              <input
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder={t('ticketManagement.filters.keywordPlaceholder')}
              />
            </label>

            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('ticketManagement.filters.status')}</span>
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={status}
                onChange={(event) => {
                  setStatus(event.target.value as TicketStatus | '')
                  setPage(1)
                }}
              >
                <option value="">{t('ticketManagement.allStatuses')}</option>
                <option value="pending">{t('ticketManagement.status.pending')}</option>
                <option value="in_progress">{t('ticketManagement.status.in_progress')}</option>
                <option value="completed">{t('ticketManagement.status.completed')}</option>
              </select>
            </label>

            <Button type="button" onClick={refresh}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('ticketManagement.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('ticketManagement.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('ticketManagement.columns.id')}</th>
                <th className="px-3 py-2">{t('ticketManagement.columns.title')}</th>
                <th className="px-3 py-2">{t('ticketManagement.columns.status')}</th>
                <th className="px-3 py-2">{t('ticketManagement.columns.priority')}</th>
                <th className="px-3 py-2">{t('ticketManagement.columns.updatedAt')}</th>
                <th className="px-3 py-2">{t('ticketManagement.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map((ticket) => (
                <tr key={ticket.id} className="border-b">
                  <td className="px-3 py-2">{ticket.id}</td>
                  <td className="px-3 py-2">
                    <div className="font-medium">{ticket.title}</div>
                    <div className="line-clamp-2 text-xs text-muted-foreground">
                      {ticket.description}
                    </div>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      className={`h-9 rounded-md border px-2 text-xs ${statusTone(ticket.status)}`}
                      value={ticket.status}
                      disabled={updatingId === ticket.id}
                      onChange={(event) => {
                        void updateTicketStatus(ticket.id, event.target.value as TicketStatus)
                      }}
                    >
                      <option value="pending">{t('ticketManagement.status.pending')}</option>
                      <option value="in_progress">{t('ticketManagement.status.in_progress')}</option>
                      <option value="completed">{t('ticketManagement.status.completed')}</option>
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      className={`h-9 rounded-md border px-2 text-xs ${priorityTone(ticket.priority)}`}
                      value={ticket.priority}
                      disabled={updatingId === ticket.id}
                      onChange={(event) => {
                        void updateTicketPriority(ticket.id, event.target.value as TicketPriority)
                      }}
                    >
                      <option value="low">{t('ticketManagement.priority.low')}</option>
                      <option value="medium">{t('ticketManagement.priority.medium')}</option>
                      <option value="high">{t('ticketManagement.priority.high')}</option>
                    </select>
                  </td>
                  <td className="px-3 py-2">{formatTime(ticket.updated_at)}</td>
                  <td className="px-3 py-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/ticket/admin-detail/${ticket.id}`)}
                    >
                      {t('ticketManagement.viewDetail')}
                    </Button>
                  </td>
                </tr>
              ))}
              {!loading && tickets.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('ticketManagement.empty')}
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
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          disabled={page <= 1}
        >
          {t('ticketManagement.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          disabled={tickets.length < pageSize || page * pageSize >= total}
        >
          {t('ticketManagement.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('ticketManagement.pageSize')}</span>
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
    </section>
  )
}
