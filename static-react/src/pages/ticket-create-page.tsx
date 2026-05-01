import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createTicket, listTicketCategories } from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess, notifyWarning } from '@/feedback/service'
import { useI18n } from '@/i18n'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function getCategoryName(category: Api.Ticket.TicketCategory, locale: string) {
  if (locale === 'en-US' && category.name_en.trim()) {
    return category.name_en
  }

  return category.name
}

export function TicketCreatePage() {
  const { locale, t } = useI18n()
  const navigate = useNavigate()
  const [loadingCategories, setLoadingCategories] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [categories, setCategories] = useState<Api.Ticket.TicketCategory[]>([])
  const [form, setForm] = useState<Api.Ticket.CreateTicketParams>({
    category_id: 0,
    title: '',
    description: '',
    priority: 'medium',
  })

  useEffect(() => {
    let cancelled = false

    const loadCategories = async () => {
      setLoadingCategories(true)
      setLoadError(null)

      try {
        const response = await listTicketCategories()

        if (cancelled) {
          return
        }

        const nextCategories = response ?? []
        setCategories(nextCategories)
        setForm((current) => {
          if (
            current.category_id > 0 &&
            nextCategories.some((category) => category.id === current.category_id)
          ) {
            return current
          }

          return {
            ...current,
            category_id: nextCategories[0]?.id ?? 0,
          }
        })
      } catch (caughtError) {
        if (!cancelled) {
          setCategories([])
          setForm((current) => ({ ...current, category_id: 0 }))
          setLoadError(getErrorMessage(caughtError, t('ticketCreate.loadCategoriesFailed')))
        }
      } finally {
        if (!cancelled) {
          setLoadingCategories(false)
        }
      }
    }

    void loadCategories()

    return () => {
      cancelled = true
    }
  }, [t])

  const handleSubmit = async () => {
    if (!form.category_id || !form.title.trim() || !form.description.trim()) {
      notifyWarning(t('ticketCreate.messages.required'))
      return
    }

    setSubmitting(true)
    try {
      await createTicket({
        category_id: form.category_id,
        title: form.title.trim(),
        description: form.description.trim(),
        priority: form.priority,
      })
      notifySuccess(t('ticketCreate.messages.created'))
      navigate('/ticket/my-tickets')
    } catch (caughtError) {
      notifyError(getErrorMessage(caughtError, t('ticketCreate.messages.createFailed')))
    } finally {
      setSubmitting(false)
    }
  }

  const canSubmit =
    !loadingCategories &&
    !submitting &&
    categories.length > 0 &&
    form.category_id > 0 &&
    form.title.trim().length > 0 &&
    form.description.trim().length > 0

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('ticketCreate.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('ticketCreate.subtitle')}</p>
      </div>

      {loadError ? <p className="text-sm text-destructive">{loadError}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="space-y-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold">{t('ticketCreate.formTitle')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('ticketCreate.formSubtitle')}</p>
            </div>

            {loadingCategories ? (
              <span className="text-sm text-muted-foreground">
                {t('ticketCreate.loadingCategories')}
              </span>
            ) : categories.length === 0 ? (
              <span className="text-sm text-muted-foreground">
                {t('ticketCreate.noCategories')}
              </span>
            ) : null}
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('ticketCreate.fields.category')}
              </span>
              <select
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={form.category_id}
                onChange={(event) => {
                  setForm((current) => ({ ...current, category_id: Number(event.target.value) }))
                }}
                disabled={loadingCategories || categories.length === 0}
              >
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {getCategoryName(category, locale)}
                  </option>
                ))}
              </select>
            </label>

            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('ticketCreate.fields.priority')}
              </span>
              <select
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={form.priority}
                onChange={(event) => {
                  setForm((current) => ({
                    ...current,
                    priority: event.target.value as Api.Ticket.TicketPriority,
                  }))
                }}
              >
                <option value="low">{t('ticketCreate.priorities.low')}</option>
                <option value="medium">{t('ticketCreate.priorities.medium')}</option>
                <option value="high">{t('ticketCreate.priorities.high')}</option>
              </select>
            </label>
          </div>

          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('ticketCreate.fields.title')}</span>
            <Input
              value={form.title}
              maxLength={200}
              placeholder={t('ticketCreate.placeholders.title')}
              onChange={(event) => {
                setForm((current) => ({ ...current, title: event.target.value }))
              }}
            />
          </label>

          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('ticketCreate.fields.description')}
            </span>
            <textarea
              className="min-h-40 w-full rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
              value={form.description}
              placeholder={t('ticketCreate.placeholders.description')}
              onChange={(event) => {
                setForm((current) => ({ ...current, description: event.target.value }))
              }}
            />
          </label>

          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" onClick={() => void handleSubmit()} disabled={!canSubmit}>
              {submitting ? t('ticketCreate.actions.submitting') : t('ticketCreate.actions.submit')}
            </Button>
            <span className="text-sm text-muted-foreground">{t('ticketCreate.submitHint')}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
