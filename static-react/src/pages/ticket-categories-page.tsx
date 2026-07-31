import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { useCallback, useEffect, useState } from 'react'
import {
  adminCreateTicketCategory,
  adminDeleteTicketCategory,
  adminListTicketCategories,
  adminUpdateTicketCategory,
} from '@/api/ticket'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { useCorpCapability } from '@/hooks/use-corp-capability'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { TicketCategory, UpsertCategoryParams } from '@/types/api/ticket'
import { ShopDialog } from './shop-page-utils'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function TicketCategoriesPage() {
  const { t } = useI18n()
  const { hasCapability } = useCorpCapability()
  const canManageCategories = hasCapability('ticket.admin.manage')
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

  const loadData = useCallback(async () => {
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
  }, [t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData])

  const openCreate = () => {
    if (!canManageCategories) return
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
    if (!canManageCategories) return
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
    if (!canManageCategories) return
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
    if (!canManageCategories) return
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
          {canManageCategories ? (
            <Button type="button" onClick={openCreate}>
              {t('ticketCategories.create')}
            </Button>
          ) : null}
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('ticketCategories.loading')}</p>
      ) : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('ticketCategories.title')} ({categories.length})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('common.id')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketCategories.columns.name')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketCategories.columns.nameEn')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketCategories.columns.sortOrder')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketCategories.columns.enabled')}</TableHead>
                <TableHead className="px-3 py-2">{t('ticketCategories.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {categories.map((category) => (
                <TableRow key={category.id} className="border-b">
                  <TableCell className="px-3 py-2">{category.id}</TableCell>
                  <TableCell className="px-3 py-2">{category.name}</TableCell>
                  <TableCell className="px-3 py-2">{category.name_en}</TableCell>
                  <TableCell className="px-3 py-2">{category.sort_order}</TableCell>
                  <TableCell className="px-3 py-2">
                    {category.enabled
                      ? t('ticketCategories.enabled')
                      : t('ticketCategories.disabled')}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {canManageCategories ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => openEdit(category)}
                        >
                          {t('common.edit')}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => void remove(category.id)}
                          isDisabled={deletingId === category.id}
                        >
                          {t('common.delete')}
                        </Button>
                      </div>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
              {!loading && categories.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('ticketCategories.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      {visible ? (
        <ShopDialog
          open={visible}
          title={editingId > 0 ? t('ticketCategories.edit') : t('ticketCategories.create')}
          onClose={() => setVisible(false)}
          closeLabel={t('common.close')}
          widthClass="max-w-2xl"
        >
          <div className="w-full max-w-2xl rounded-lg border bg-card p-5 shadow-xl">
            <div className="mt-4 grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">
                  {t('ticketCategories.columns.name')}
                </span>
                <Input
                  value={form.name ?? ''}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, name: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">
                  {t('ticketCategories.columns.nameEn')}
                </span>
                <Input
                  value={form.name_en ?? ''}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, name_en: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">
                  {t('ticketCategories.columns.description')}
                </span>
                <Input
                  value={form.description ?? ''}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, description: event.target.value }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">
                  {t('ticketCategories.columns.sortOrder')}
                </span>
                <Input
                  type="number"
                  value={String(form.sort_order ?? 0)}
                  onChange={(event) =>
                    setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))
                  }
                />
              </label>
              <label className="flex items-center gap-2 pt-8">
                <Checkbox
                  isSelected={Boolean(form.enabled)}
                  onChange={(selected) =>
                    setForm((current) => ({ ...current, enabled: selected === true }))
                  }
                />
                <span className="text-sm text-muted-foreground">
                  {t('ticketCategories.columns.enabled')}
                </span>
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => setVisible(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void save()} isDisabled={saving}>
                {saving ? t('ticketCategories.saving') : t('common.confirm')}
              </Button>
            </div>
          </div>
        </ShopDialog>
      ) : null}
    </section>
  )
}
