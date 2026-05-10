import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listMyTickets } from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function statusTone(status: Api.Ticket.TicketStatus) {
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

function priorityTone(priority: Api.Ticket.TicketPriority) {
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

function statusLabel(t: ReturnType<typeof useI18n>['t'], status: Api.Ticket.TicketStatus) {
  return t(`ticketMyTickets.statuses.${status}`)
}

function priorityLabel(t: ReturnType<typeof useI18n>['t'], priority: Api.Ticket.TicketPriority) {
  return t(`ticketMyTickets.priorities.${priority}`)
}

export function TicketMyTicketsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [appliedStatusFilter, setAppliedStatusFilter] = useState('')
  const [tickets, setTickets] = useState<Api.Ticket.TicketItem[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const response = await listMyTickets({
          current: page,
          size: pageSize,
          status: appliedStatusFilter as Api.Ticket.TicketStatus | '',
        })

        if (cancelled) {
          return
        }

        setTickets(response.list ?? [])
        setTotal(response.total ?? 0)
        setPage(response.page ?? page)
        setPageSize(response.pageSize ?? pageSize)
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('ticketMyTickets.loadFailed')))
          setTickets([])
          setTotal(0)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadData()

    return () => {
      cancelled = true
    }
  }, [appliedStatusFilter, page, pageSize, t])

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const handleSearch = () => {
    setAppliedStatusFilter(statusFilter)
    setPage(1)
  }

  const goCreate = () => navigate('/ticket/create')
  const goDetail = (id: number) => navigate(`/ticket/detail/${id}`)

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('ticketMyTickets.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('ticketMyTickets.subtitle')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('ticketMyTickets.filters.status')}
              </span>
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value)}
              >
                <option value="">{t('ticketMyTickets.allStatuses')}</option>
                <option value="pending">{statusLabel(t, 'pending')}</option>
                <option value="in_progress">{statusLabel(t, 'in_progress')}</option>
                <option value="completed">{statusLabel(t, 'completed')}</option>
              </select>
            </label>

            <Button type="button" onClick={handleSearch}>
              {t('common.search')}
            </Button>
            <Button type="button" variant="outline" onClick={goCreate}>
              {t('ticketMyTickets.createTicket')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('ticketMyTickets.loading')}</p>
      ) : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('ticketMyTickets.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('ticketMyTickets.columns.id')}</th>
                <th className="px-3 py-2">{t('ticketMyTickets.columns.title')}</th>
                <th className="px-3 py-2">{t('ticketMyTickets.columns.status')}</th>
                <th className="px-3 py-2">{t('ticketMyTickets.columns.priority')}</th>
                <th className="px-3 py-2">{t('ticketMyTickets.columns.updatedAt')}</th>
                <th className="px-3 py-2">{t('ticketMyTickets.columns.actions')}</th>
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
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusTone(ticket.status)}`}
                    >
                      {statusLabel(t, ticket.status)}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${priorityTone(ticket.priority)}`}
                    >
                      {priorityLabel(t, ticket.priority)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatTime(ticket.updated_at)}</td>
                  <td className="px-3 py-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => goDetail(ticket.id)}
                    >
                      {t('ticketMyTickets.viewDetail')}
                    </Button>
                  </td>
                </tr>
              ))}
              {!loading && tickets.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('ticketMyTickets.empty')}
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
          {t('ticketMyTickets.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          disabled={tickets.length < pageSize || page * pageSize >= total}
        >
          {t('ticketMyTickets.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('ticketMyTickets.pageSize')}</span>
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
