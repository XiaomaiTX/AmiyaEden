import { Textarea } from '@/components/ui/textarea'
import { useCallback, useEffect, useState } from 'react'
import {
  createToolBookmark,
  deleteToolBookmark,
  fetchAdminToolBookmarks,
  fetchVisibleToolBookmarks,
  updateToolBookmark,
} from '@/api/tool-bookmark'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { confirmAction, notifyError, notifySuccess, notifyWarning } from '@/feedback'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { ToolBookmark, ToolBookmarkUpsertRequest } from '@/types/api/tool-bookmark'
import { ShopDialog } from './shop-page-utils'

const emptyForm: ToolBookmarkUpsertRequest = {
  name: '',
  url: '',
  description: '',
  is_enabled: true,
  sort_order: 0,
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function InfoToolBookmarksPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const isAdmin = roles.includes('admin') || roles.includes('super_admin')
  const [bookmarks, setBookmarks] = useState<ToolBookmark[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editingBookmark, setEditingBookmark] = useState<ToolBookmark | null>(null)
  const [form, setForm] = useState<ToolBookmarkUpsertRequest>(emptyForm)

  const loadBookmarks = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setBookmarks(isAdmin ? await fetchAdminToolBookmarks() : await fetchVisibleToolBookmarks())
    } catch (caughtError) {
      setBookmarks([])
      setError(getErrorMessage(caughtError, t('infoToolBookmarks.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [isAdmin, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadBookmarks()
    }, 0)

    return () => window.clearTimeout(timer)
  }, [loadBookmarks])

  const openCreateDialog = () => {
    setEditingBookmark(null)
    setForm({ ...emptyForm, sort_order: bookmarks.length + 1 })
    setDialogOpen(true)
  }

  const openEditDialog = (bookmark: ToolBookmark) => {
    setEditingBookmark(bookmark)
    setForm({
      name: bookmark.name,
      url: bookmark.url,
      description: bookmark.description,
      is_enabled: bookmark.is_enabled,
      sort_order: bookmark.sort_order,
    })
    setDialogOpen(true)
  }

  const submitForm = async () => {
    const data: ToolBookmarkUpsertRequest = {
      ...form,
      name: form.name.trim(),
      url: form.url.trim(),
      description: form.description?.trim() ?? '',
    }
    if (!data.name || !data.url) {
      notifyWarning(t('infoToolBookmarks.requiredFields'))
      return
    }

    setSaving(true)
    try {
      if (editingBookmark) {
        await updateToolBookmark(editingBookmark.id, data)
      } else {
        await createToolBookmark(data)
      }
      setDialogOpen(false)
      notifySuccess(t('infoToolBookmarks.saveSuccess'))
      await loadBookmarks()
    } catch (caughtError) {
      notifyError(getErrorMessage(caughtError, t('infoToolBookmarks.saveFailed')))
    } finally {
      setSaving(false)
    }
  }

  const removeBookmark = async (bookmark: ToolBookmark) => {
    const confirmed = await confirmAction({
      title: t('common.delete'),
      message: t('infoToolBookmarks.deleteConfirm', { name: bookmark.name }),
      confirmText: t('common.delete'),
      cancelText: t('common.cancel'),
    })
    if (!confirmed) {
      return
    }

    try {
      await deleteToolBookmark(bookmark.id)
      notifySuccess(t('infoToolBookmarks.deleteSuccess'))
      await loadBookmarks()
    } catch (caughtError) {
      notifyError(getErrorMessage(caughtError, t('infoToolBookmarks.deleteFailed')))
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('infoToolBookmarks.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('infoToolBookmarks.subtitle')}</p>
          </div>
          {isAdmin ? (
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => void loadBookmarks()}
                isDisabled={loading}
              >
                {t('common.refresh')}
              </Button>
              <Button type="button" onClick={openCreateDialog}>
                {t('infoToolBookmarks.add')}
              </Button>
            </div>
          ) : null}
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('infoToolBookmarks.loading')}</p>
      ) : null}

      {!loading && !error && bookmarks.length === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          {t('infoToolBookmarks.empty')}
        </div>
      ) : null}

      {!loading && bookmarks.length > 0 ? (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {bookmarks.map((bookmark) => (
            <article
              key={bookmark.id}
              className="flex flex-col gap-4 rounded-lg border bg-card p-4"
            >
              <div className="flex min-w-0 gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted text-sm font-semibold text-muted-foreground">
                  {bookmark.logo_url ? (
                    <img
                      src={bookmark.logo_url}
                      alt={bookmark.name}
                      className="size-full object-cover"
                      loading="lazy"
                    />
                  ) : (
                    bookmark.name.slice(0, 1)
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <a
                      className="font-semibold text-primary underline-offset-4 hover:underline"
                      href={bookmark.url}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {bookmark.name}
                    </a>
                    {isAdmin && !bookmark.is_enabled ? (
                      <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                        {t('infoToolBookmarks.disabled')}
                      </span>
                    ) : null}
                  </div>
                  {bookmark.description ? (
                    <p className="mt-2 text-sm">{bookmark.description}</p>
                  ) : null}
                  <p className="mt-2 break-all text-xs text-muted-foreground">{bookmark.url}</p>
                </div>
              </div>
              {isAdmin ? (
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => openEditDialog(bookmark)}
                  >
                    {t('common.edit')}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="destructive"
                    onClick={() => void removeBookmark(bookmark)}
                  >
                    {t('common.delete')}
                  </Button>
                </div>
              ) : null}
            </article>
          ))}
        </div>
      ) : null}

      {dialogOpen ? (
        <ShopDialog
          open={dialogOpen}
          title={editingBookmark ? t('infoToolBookmarks.edit') : t('infoToolBookmarks.add')}
          onClose={() => setDialogOpen(false)}
          closeLabel={t('common.close')}
          widthClass="w-full max-w-sm"
          scrollable={false}
        >
          <div className="space-y-4">
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('infoToolBookmarks.fields.name')}
              </span>
              <Input
                value={form.name}
                maxLength={128}
                onChange={(event) =>
                  setForm((current) => ({ ...current, name: event.target.value }))
                }
              />
            </label>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('infoToolBookmarks.fields.url')}
              </span>
              <Input
                value={form.url}
                onChange={(event) =>
                  setForm((current) => ({ ...current, url: event.target.value }))
                }
              />
            </label>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('infoToolBookmarks.fields.description')}
              </span>
              <Textarea
                className="min-h-24"
                value={form.description ?? ''}
                maxLength={1024}
                onChange={(event) =>
                  setForm((current) => ({ ...current, description: event.target.value }))
                }
              />
            </label>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">
                {t('infoToolBookmarks.fields.sortOrder')}
              </span>
              <Input
                type="number"
                min={0}
                value={String(form.sort_order ?? 0)}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    sort_order: Number(event.target.value) || 0,
                  }))
                }
              />
            </label>
            <label className="flex items-center gap-3 text-sm">
              <Checkbox
                isSelected={form.is_enabled ?? true}
                onChange={(selected) =>
                  setForm((current) => ({ ...current, is_enabled: selected === true }))
                }
              />
              {t('infoToolBookmarks.fields.enabled')}
            </label>
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
                isDisabled={saving}
              >
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void submitForm()} isDisabled={saving}>
                {saving ? t('infoToolBookmarks.saving') : t('common.save')}
              </Button>
            </div>
          </div>
        </ShopDialog>
      ) : null}
    </section>
  )
}
