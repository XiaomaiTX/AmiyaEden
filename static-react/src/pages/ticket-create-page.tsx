import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createTicket, listTicketCategories } from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess, notifyWarning } from '@/feedback/service'
import { useI18n } from '@/i18n'
import type { CreateTicketParams, TicketCategory, TicketPriority } from '@/types/api/ticket'
import { getCategoryName } from '@/pages/ticket-category'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function TicketCreatePage() {
  const { locale, t } = useI18n()
  const navigate = useNavigate()
  const [loadingCategories, setLoadingCategories] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [categories, setCategories] = useState<TicketCategory[]>([])
  const [form, setForm] = useState<CreateTicketParams>({
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
              <Select
                aria-label={t('ticketCreate.fields.category')}
                selectedKey={String(form.category_id ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setForm((current) => ({ ...current, category_id: Number(value) }))
                })(String(key))}
              >
                <SelectTrigger
                  aria-label={t('ticketCreate.fields.category')}
                  className="h-10 w-full"
                  isDisabled={loadingCategories || categories.length === 0}
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((category) => (
                    <SelectItem key={category.id} id={String(category.id ?? '')}>
                      {getCategoryName(category, locale)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>

            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('ticketCreate.fields.priority')}
              </span>
              <Select
                selectedKey={String(form.priority ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setForm((current) => ({
                    ...current,
                    priority: value as TicketPriority,
                  }))
                })(String(key))}
              >
                <SelectTrigger className="h-10 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="low">{t('ticketCreate.priorities.low')}</SelectItem>
                  <SelectItem id="medium">{t('ticketCreate.priorities.medium')}</SelectItem>
                  <SelectItem id="high">{t('ticketCreate.priorities.high')}</SelectItem>
                </SelectContent>
              </Select>
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
            <Textarea
              className="min-h-40"
              value={form.description}
              placeholder={t('ticketCreate.placeholders.description')}
              onChange={(event) => {
                setForm((current) => ({ ...current, description: event.target.value }))
              }}
            />
          </label>

          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" onClick={() => void handleSubmit()} isDisabled={!canSubmit}>
              {submitting ? t('ticketCreate.actions.submitting') : t('ticketCreate.actions.submit')}
            </Button>
            <span className="text-sm text-muted-foreground">{t('ticketCreate.submitHint')}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
