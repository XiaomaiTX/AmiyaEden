import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { addMyTicketReply, getMyTicket, listMyTicketReplies } from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { notifyError, notifySuccess } from '@/feedback/service'
import { useI18n } from '@/i18n'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string | undefined) {
  if (!value) {
    return '-'
  }

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

export function TicketDetailPage() {
  const { t } = useI18n()
  const params = useParams()
  const ticketId = Number(params.id)
  const invalidId = !Number.isFinite(ticketId) || ticketId <= 0
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [ticket, setTicket] = useState<Api.Ticket.TicketItem | null>(null)
  const [replies, setReplies] = useState<Api.Ticket.TicketReply[]>([])
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (invalidId) {
      return
    }

    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const [ticketData, replyData] = await Promise.all([
          getMyTicket(ticketId),
          listMyTicketReplies(ticketId),
        ])

        if (!cancelled) {
          setTicket(ticketData)
          setReplies(replyData)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('ticket.messages.loadFailed')))
          setTicket(null)
          setReplies([])
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
  }, [invalidId, ticketId, t])

  const handleReply = async () => {
    if (!content.trim() || !ticket) {
      return
    }

    setSubmitting(true)
    try {
      await addMyTicketReply(ticket.id, { content: content.trim() })
      setContent('')
      const replyData = await listMyTicketReplies(ticket.id)
      setReplies(replyData)
      notifySuccess(t('ticket.messages.replyAdded'))
    } catch (caughtError) {
      notifyError(getErrorMessage(caughtError, t('ticket.messages.replyFailed')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('ticket.detailTitle')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('ticket.detailSubtitle')}</p>
      </div>

      {invalidId ? (
        <p className="text-sm text-destructive">{t('ticket.messages.invalidId')}</p>
      ) : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!invalidId && loading ? (
        <p className="text-sm text-muted-foreground">{t('ticket.detailLoading')}</p>
      ) : null}

      {ticket ? (
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold">
                #{ticket.id} {ticket.title}
              </h2>
              <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                {ticket.description}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span
                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusTone(ticket.status)}`}
              >
                {statusLabel(t, ticket.status)}
              </span>
              <span
                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${priorityTone(ticket.priority)}`}
              >
                {priorityLabel(t, ticket.priority)}
              </span>
            </div>
          </div>

          <div className="mt-4 grid gap-3 text-sm text-muted-foreground sm:grid-cols-2">
            <div>
              <span className="font-medium text-foreground">
                {t('ticket.detailFields.createdAt')}:
              </span>{' '}
              {formatTime(ticket.created_at)}
            </div>
            <div>
              <span className="font-medium text-foreground">
                {t('ticket.detailFields.updatedAt')}:
              </span>{' '}
              {formatTime(ticket.updated_at)}
            </div>
          </div>
        </div>
      ) : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-semibold">{t('ticket.replies')}</h2>
          <span className="text-sm text-muted-foreground">{replies.length}</span>
        </div>

        <div className="mt-4 space-y-3">
          {replies.length > 0 ? (
            replies.map((reply) => (
              <article key={reply.id} className="rounded-lg border bg-background p-4">
                <div className="flex items-center justify-between gap-3 text-sm">
                  <span className="font-medium">
                    {reply.is_internal ? t('ticket.replyInternal') : t('ticket.replyExternal')}
                  </span>
                  <span className="text-muted-foreground">{formatTime(reply.created_at)}</span>
                </div>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{reply.content}</p>
              </article>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">{t('ticket.noReplies')}</p>
          )}
        </div>

        <div className="mt-5 space-y-3">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('ticket.reply')}</span>
            <textarea
              className="min-h-28 w-full rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
              value={content}
              placeholder={t('ticket.replyPlaceholder')}
              onChange={(event) => setContent(event.target.value)}
            />
          </label>

          <Button
            type="button"
            onClick={() => void handleReply()}
            disabled={submitting || !content.trim()}
          >
            {submitting ? t('ticket.replySubmitting') : t('ticket.replySubmit')}
          </Button>
        </div>
      </div>
    </section>
  )
}
