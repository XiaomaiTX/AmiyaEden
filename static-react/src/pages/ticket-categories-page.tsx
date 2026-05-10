import { useEffect, useState } from 'react'
import {
  adminCreateTicketCategory,
  adminDeleteTicketCategory,
  adminListTicketCategories,
  adminUpdateTicketCategory,
} from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { TicketCategory, UpsertCategoryParams } from '@/types/api/ticket'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function TicketCategoriesPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [categories, setCategories] = useState<TicketCategory[]>([])
  const [visible, setVisible] = useState(false)
  const [editingId, setEditingId] = useState(0)
  const [saving, setSaving] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [form, setForm] = useState<UpsertCategoryParams>({
    name: '',
    name_en: '',
    description: '',
    sort_order: 0,
    enabled: true,
  })

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await adminListTicketCategories()
      setCategories(list)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketCategories.loadFailed')))
      setCategories([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  const openCreate = () => {
    setEditingId(0)
    setForm({
      name: '',
      name_en: '',
      description: '',
      sort_order: 0,
      enabled: true,
    })
    setVisible(true)
  }

  const openEdit = (category: TicketCategory) => {
    setEditingId(category.id)
    setForm({
      name: category.name,
      name_en: category.name_en,
      description: category.description,
      sort_order: category.sort_order,
      enabled: category.enabled,
    })
    setVisible(true)
  }

  const save = async () => {
    if (!form.name.trim() || !form.name_en.trim()) {
      setError(t('ticketCategories.required'))
      return
    }

    setSaving(true)
    try {
      if (editingId > 0) {
        await adminUpdateTicketCategory(editingId, form)
      } else {
        await adminCreateTicketCategory(form)
      }
      setVisible(false)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketCategories.saveFailed')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (id: number) => {
    if (!window.confirm(t('ticketCategories.deleteConfirm'))) {
      return
    }

    setDeletingId(id)
    try {
      await adminDeleteTicketCategory(id)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('ticketCategories.deleteFailed')))
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold">{t('ticketCategories.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('ticketCategories.subtitle')}</p>
          </div>
          <Button type="button" onClick={openCreate}>
            {t('ticketCategories.create')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('ticketCategories.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('ticketCategories.title')} ({categories.length})
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">ID</th>
                <th className="px-3 py-2">{t('ticketCategories.columns.name')}</th>
                <th className="px-3 py-2">{t('ticketCategories.columns.nameEn')}</th>
                <th className="px-3 py-2">{t('ticketCategories.columns.sortOrder')}</th>
                <th className="px-3 py-2">{t('ticketCategories.columns.enabled')}</th>
                <th className="px-3 py-2">{t('ticketCategories.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {categories.map((category) => (
                <tr key={category.id} className="border-b">
                  <td className="px-3 py-2">{category.id}</td>
                  <td className="px-3 py-2">{category.name}</td>
                  <td className="px-3 py-2">{category.name_en}</td>
                  <td className="px-3 py-2">{category.sort_order}</td>
                  <td className="px-3 py-2">{category.enabled ? t('ticketCategories.enabled') : t('ticketCategories.disabled')}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" variant="outline" onClick={() => openEdit(category)}>
                        {t('common.edit')}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void remove(category.id)}
                        disabled={deletingId === category.id}
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && categories.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('ticketCategories.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      {visible ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-2xl rounded-lg border bg-card p-5 shadow-xl">
            <h2 className="text-lg font-semibold">
              {editingId > 0 ? t('ticketCategories.edit') : t('ticketCategories.create')}
            </h2>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('ticketCategories.columns.name')}</span>
                <Input value={form.name ?? ''} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('ticketCategories.columns.nameEn')}</span>
                <Input value={form.name_en ?? ''} onChange={(event) => setForm((current) => ({ ...current, name_en: event.target.value }))} />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('ticketCategories.columns.description')}</span>
                <Input value={form.description ?? ''} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('ticketCategories.columns.sortOrder')}</span>
                <Input
                  type="number"
                  value={String(form.sort_order ?? 0)}
                  onChange={(event) => setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))}
                />
              </label>
              <label className="flex items-center gap-2 pt-8">
                <input
                  type="checkbox"
                  checked={Boolean(form.enabled)}
                  onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.checked }))}
                />
                <span className="text-sm text-muted-foreground">{t('ticketCategories.columns.enabled')}</span>
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => setVisible(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void save()} disabled={saving}>
                {saving ? t('ticketCategories.saving') : t('common.confirm')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
