import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  adminAddTicketReply,
  adminGetTicket,
  adminListTicketReplies,
  adminListTicketStatusHistory,
} from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { TicketItem, TicketReply, TicketStatusHistory } from '@/types/api/ticket'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string | undefined) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function statusTone(status: string) {
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

function priorityTone(priority: string) {
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

export function TicketAdminDetailPage() {
  const { t } = useI18n()
  const params = useParams()
  const ticketId = Number(params.id)
  const invalidId = !Number.isFinite(ticketId) || ticketId <= 0
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [ticket, setTicket] = useState<TicketItem | null>(null)
  const [replies, setReplies] = useState<TicketReply[]>([])
  const [histories, setHistories] = useState<TicketStatusHistory[]>([])
  const [content, setContent] = useState('')
  const [isInternal, setIsInternal] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (invalidId) return

    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)
      try {
        const [ticketData, replyData, historyData] = await Promise.all([
          adminGetTicket(ticketId),
          adminListTicketReplies(ticketId),
          adminListTicketStatusHistory(ticketId),
        ])
        if (!cancelled) {
          setTicket(ticketData)
          setReplies(replyData)
          setHistories(historyData)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('ticketAdminDetail.loadFailed')))
          setTicket(null)
          setReplies([])
          setHistories([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadData()
    return () => {
      cancelled = true
    }
  }, [invalidId, ticketId, t])

  const submitReply = async () => {
    if (!content.trim() || invalidId) return

    setSubmitting(true)
    try {
      await adminAddTicketReply(ticketId, {
        content: content.trim(),
        is_internal: isInternal,
      })
      setContent('')
      const [replyData, historyData] = await Promise.all([
        adminListTicketReplies(ticketId),
        adminListTicketStatusHistory(ticketId),
      ])
      setReplies(replyData)
      setHistories(historyData)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketAdminDetail.replyFailed')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('ticketAdminDetail.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('ticketAdminDetail.subtitle')}</p>
      </div>

      {invalidId ? <p className="text-sm text-destructive">{t('ticketAdminDetail.invalidId')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!invalidId && loading ? <p className="text-sm text-muted-foreground">{t('ticketAdminDetail.loading')}</p> : null}

      {ticket ? (
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold">
                #{ticket.id} {ticket.title}
              </h2>
              <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
                {ticket.description}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusTone(ticket.status)}`}>
                {ticket.status}
              </span>
              <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${priorityTone(ticket.priority)}`}>
                {ticket.priority}
              </span>
            </div>
          </div>

          <div className="mt-4 grid gap-3 text-sm text-muted-foreground sm:grid-cols-2">
            <div>
              <span className="font-medium text-foreground">{t('ticketAdminDetail.fields.createdAt')}:</span>{' '}
              {formatTime(ticket.created_at)}
            </div>
            <div>
              <span className="font-medium text-foreground">{t('ticketAdminDetail.fields.updatedAt')}:</span>{' '}
              {formatTime(ticket.updated_at)}
            </div>
          </div>
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">{t('ticketAdminDetail.replies')}</h2>
            <span className="text-sm text-muted-foreground">{replies.length}</span>
          </div>

          <div className="mt-4 space-y-3">
            {replies.length > 0 ? (
              replies.map((reply) => (
                <article key={reply.id} className="rounded-lg border bg-background p-4">
                  <div className="flex items-center justify-between gap-3 text-sm">
                    <span className="font-medium">
                      {reply.is_internal ? t('ticketAdminDetail.replyInternal') : t('ticketAdminDetail.replyExternal')}
                    </span>
                    <span className="text-muted-foreground">{formatTime(reply.created_at)}</span>
                  </div>
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{reply.content}</p>
                </article>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">{t('ticketAdminDetail.noReplies')}</p>
            )}
          </div>

          <div className="mt-5 space-y-3">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={isInternal} onChange={(event) => setIsInternal(event.target.checked)} />
              <span>{t('ticketAdminDetail.internalNote')}</span>
            </label>
            <textarea
              className="min-h-28 w-full rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
              value={content}
              placeholder={t('ticketAdminDetail.replyPlaceholder')}
              onChange={(event) => setContent(event.target.value)}
            />
            <Button type="button" onClick={() => void submitReply()} disabled={submitting || !content.trim()}>
              {submitting ? t('ticketAdminDetail.replySubmitting') : t('ticketAdminDetail.replySubmit')}
            </Button>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-lg font-semibold">{t('ticketAdminDetail.statusHistory')}</h2>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('ticketAdminDetail.columns.fromStatus')}</th>
                  <th className="px-3 py-2">{t('ticketAdminDetail.columns.toStatus')}</th>
                  <th className="px-3 py-2">{t('ticketAdminDetail.columns.operator')}</th>
                  <th className="px-3 py-2">{t('ticketAdminDetail.columns.changedAt')}</th>
                </tr>
              </thead>
              <tbody>
                {histories.map((history) => (
                  <tr key={history.id} className="border-b">
                    <td className="px-3 py-2">{history.from_status}</td>
                    <td className="px-3 py-2">{history.to_status}</td>
                    <td className="px-3 py-2">{history.changed_by}</td>
                    <td className="px-3 py-2">{formatTime(history.changed_at)}</td>
                  </tr>
                ))}
                {!loading && histories.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={4}>
                      {t('ticketAdminDetail.noHistory')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  )
}
