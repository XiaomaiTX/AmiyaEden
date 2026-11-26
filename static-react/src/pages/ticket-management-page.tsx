import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { adminListTickets } from '@/api/generated/client'
import { adminUpdateStatus } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { TicketAdminItem, TicketStatus } from '@/api/generated/client'

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

export function TicketManagementPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tickets, setTickets] = useState<TicketAdminItem[]>([])
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
        const response = await unwrapContract(
          adminListTickets({
            current: page,
            size: pageSize,
            keyword: keyword.trim() || undefined,
            status: status || undefined,
          }),
          'admin list tickets failed'
        )

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
      await unwrapContract(
        adminUpdateStatus(ticketId, { status: nextStatus }),
        'admin update ticket status failed'
      )
      setTickets((current) =>
        current.map((ticket) =>
          ticket.id === ticketId ? { ...ticket, status: nextStatus } : ticket
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
              <span className="text-sm text-muted-foreground">
                {t('ticketManagement.filters.keyword')}
              </span>
              <Input
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder={t('ticketManagement.filters.keywordPlaceholder')}
              />
            </label>

            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('ticketManagement.filters.status')}
              </span>
              <Select
                selectedKey={String(status ?? '')}
                onSelectionChange={(key) =>
                  ((value) => {
                    setStatus(value as TicketStatus | '')
                    setPage(1)
                  })(String(key))
                }
              >
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('ticketManagement.allStatuses')}</SelectItem>
                  <SelectItem id="pending">{t('ticketManagement.status.pending')}</SelectItem>
                  <SelectItem id="in_progress">
                    {t('ticketManagement.status.in_progress')}
                  </SelectItem>
                  <SelectItem id="completed">{t('ticketManagement.status.completed')}</SelectItem>
                </SelectContent>
              </Select>
            </label>

            <Button type="button" onClick={refresh}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('ticketManagement.loading')}</p>
      ) : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('ticketManagement.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('ticketManagement.columns.id')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketManagement.columns.title')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketManagement.columns.status')}</TableHead>
                <TableHead className="px-3 py-2">
                  {t('ticketManagement.columns.updatedAt')}
                </TableHead>
                <TableHead className="px-3 py-2">{t('ticketManagement.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tickets.map((ticket) => (
                <TableRow key={ticket.id} className="border-b">
                  <TableCell className="px-3 py-2">{ticket.id}</TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="font-medium">{ticket.title}</div>
                    <div className="line-clamp-2 text-xs text-muted-foreground">
                      {ticket.description}
                    </div>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <Select
                      selectedKey={String(ticket.status ?? '')}
                      onSelectionChange={(key) =>
                        ((value) => {
                          void updateTicketStatus(ticket.id, value as TicketStatus)
                        })(String(key))
                      }
                    >
                      <SelectTrigger
                        className={`h-9 text-xs ${statusTone(ticket.status)}`}
                        isDisabled={updatingId === ticket.id}
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem id="pending">{t('ticketManagement.status.pending')}</SelectItem>
                        <SelectItem id="in_progress">
                          {t('ticketManagement.status.in_progress')}
                        </SelectItem>
                        <SelectItem id="completed">
                          {t('ticketManagement.status.completed')}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell className="px-3 py-2">{formatTime(ticket.updated_at)}</TableCell>
                  <TableCell className="px-3 py-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/ticket/admin-detail/${ticket.id}`)}
                    >
                      {t('ticketManagement.viewDetail')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {!loading && tickets.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                    {t('ticketManagement.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
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
          isDisabled={page <= 1}
        >
          {t('ticketManagement.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          isDisabled={tickets.length < pageSize || page * pageSize >= total}
        >
          {t('ticketManagement.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('ticketManagement.pageSize')}</span>
          <Select
            selectedKey={String(pageSize ?? '')}
            onSelectionChange={(key) =>
              ((value) => {
                setPageSize(Number(value))
                setPage(1)
              })(String(key))
            }
          >
            <SelectTrigger className="h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[10, 20, 50].map((size) => (
                <SelectItem key={size} id={String(size ?? '')}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
      </div>
    </section>
  )
}
