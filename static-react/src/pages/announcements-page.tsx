import { useEffect, useEffectEvent, useRef, useState } from 'react'
import {
  listAnnouncements,
  getAnnouncement,
  saveAnnouncement,
  readAnnouncement,
} from '@/api/announcements'
import type { Announcement } from '@/types/api/announcement'
import { useI18n } from '@/i18n'
import { useRole } from '@/hooks/use-permission'
import { useBadgeStore } from '@/stores/badge-store'
import { useSessionStore } from '@/stores/session-store'
import { RichTextEditor } from '@/components/rich-text-editor'
import { Button } from '@/components/ui/button'

export function AnnouncementsPage() {
  const { t } = useI18n()
  const manage = useRole(['super_admin', 'admin', 'senior_fc'])
  const [rows, setRows] = useState<Announcement[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [selected, setSelected] = useState<Announcement | null>(null)
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [hidden, setHidden] = useState(false)
  const [remind, setRemind] = useState(false)
  const [busy, setBusy] = useState(false)
  // null = no error; '' = failure without a server message (generic text is shown).
  const [error, setError] = useState<string | null>(null)
  const request = useRef(0)
  const autoOpened = useRef(false)
  const fail = (cause: unknown) => setError(cause instanceof Error ? cause.message : '')
  async function badges() {
    const id = useSessionStore.getState().characterId
    if (id) await useBadgeStore.getState().load(id, true)
  }
  async function load() {
    const data = await listAnnouncements(page)
    setRows(data.list)
    setTotal(data.total)
  }
  // On entry show the newest post, unless the user already opened one or started publishing.
  const openNewest = useEffectEvent((row: Announcement) => {
    if (autoOpened.current) return
    autoOpened.current = true
    void open(row)
  })
  useEffect(() => {
    let active = true
    const token = request.current
    void listAnnouncements(page)
      .then(async (data) => {
        if (!active) return
        setError(null)
        setRows(data.list)
        setTotal(data.total)
        if (token === request.current && data.list[0]) openNewest(data.list[0])
        await badges()
      })
      .catch((cause: unknown) => {
        if (active) fail(cause)
      })
    return () => {
      active = false
    }
  }, [page])
  useEffect(
    () => () => {
      request.current++
    },
    []
  )
  async function open(row: Announcement) {
    const token = ++request.current
    setBusy(true)
    setError(null)
    setEditing(false)
    let data: Announcement
    try {
      data = await getAnnouncement(row.id)
    } catch (cause) {
      if (token === request.current) {
        fail(cause)
        setBusy(false)
      }
      return
    }
    if (token !== request.current) return
    setSelected(data)
    setBusy(false)
    // Each request costs a full round trip, so acknowledgement runs in the background without
    // locking the list, and is skipped when the displayed version is already read.
    if (data.hidden || (!row.unread && data.version === row.version)) return
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
    )
    if (token !== request.current) return
    try {
      await readAnnouncement(data.id, data.version)
      setRows((current) =>
        current.map((item) =>
          item.id === data.id ? { ...item, unread: false, version: data.version } : item
        )
      )
      await badges()
    } catch (cause) {
      if (token === request.current) fail(cause)
    }
  }
  function edit() {
    setTitle(selected?.title ?? '')
    setBody(selected?.body_html ?? '')
    setHidden(selected?.hidden ?? false)
    setRemind(false)
    setEditing(true)
  }
  async function save() {
    setBusy(true)
    setError(null)
    try {
      const data = await saveAnnouncement(selected?.id ?? null, {
        title,
        body_html: body,
        hidden,
        remind,
        revision: selected?.revision ?? 0,
      })
      setSelected(data)
      setEditing(false)
      await load()
      await badges()
    } catch (cause) {
      fail(cause)
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="space-y-4 p-4 md:p-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t('announcement.title')}</h1>
        {manage && (
          <Button
            isDisabled={busy}
            onPress={() => {
              request.current++
              setSelected(null)
              setTitle('')
              setBody('')
              setHidden(false)
              setRemind(false)
              setEditing(true)
            }}
          >
            {t('announcement.publish')}
          </Button>
        )}
      </header>
      {error !== null && (
        <p role="alert" className="text-destructive">
          {error || t('announcement.error')}
        </p>
      )}
      <div className="grid items-start gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="space-y-2">
          {rows.length === 0 && <p>{t('announcement.empty')}</p>}
          {rows.map((row) => {
            const isUnread = Boolean(row.unread && !row.hidden)
            return (
              <button
                key={row.id}
                disabled={busy}
                onClick={() => void open(row)}
                className={`w-full rounded-lg border p-4 text-left transition-colors ${
                  isUnread
                    ? 'border-blue-500/50 border-l-4 border-l-blue-500 bg-blue-500/10 font-semibold text-foreground dark:border-blue-400/50 dark:border-l-blue-400 dark:bg-blue-400/15'
                    : 'border-border bg-card text-foreground/80 hover:bg-muted/50'
                } ${selected?.id === row.id ? 'ring-2 ring-primary' : ''}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <span className="flex min-w-0 items-start gap-2">
                    {isUnread && (
                      <span
                        aria-hidden="true"
                        className="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full bg-blue-500 dark:bg-blue-400"
                      />
                    )}
                    <span className="line-clamp-2 break-words">{row.title}</span>
                  </span>
                  {isUnread && (
                    <span className="shrink-0 rounded-full bg-blue-600 px-2 py-0.5 text-[11px] font-semibold text-white dark:bg-blue-400 dark:text-slate-950">
                      {t('announcement.unread')}
                    </span>
                  )}
                  {row.hidden && (
                    <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
                      {t('announcement.hidden')}
                    </span>
                  )}
                </div>
                <time className="mt-2 block text-xs font-normal text-muted-foreground">
                  {new Date(row.created_at).toLocaleString()}
                </time>
              </button>
            )
          })}
          <div className="flex justify-between">
            <Button
              variant="outline"
              isDisabled={page === 1 || busy}
              onPress={() => setPage(page - 1)}
            >
              {t('announcement.previous')}
            </Button>
            <Button
              variant="outline"
              isDisabled={page * 20 >= total || busy}
              onPress={() => setPage(page + 1)}
            >
              {t('announcement.next')}
            </Button>
          </div>
        </aside>
        <section aria-busy={busy} className="min-w-0 space-y-4 rounded-lg border p-5">
          {editing ? (
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                void save()
              }}
            >
              <label className="block">
                {t('announcement.subject')}
                <input
                  required
                  maxLength={200}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="mt-2 w-full rounded border bg-background p-2"
                />
              </label>
              <p className="text-sm text-muted-foreground">{t('announcement.imageHint')}</p>
              <RichTextEditor
                value={body}
                onChange={setBody}
                ariaLabel={t('announcement.body')}
                imageUploadLabel={t('announcement.image')}
                maxImages={6}
              />
              <label className="flex gap-2">
                <input
                  type="checkbox"
                  checked={hidden}
                  onChange={(e) => setHidden(e.target.checked)}
                />
                {t('announcement.hidden')}
              </label>
              {selected && (
                <label className="flex gap-2">
                  <input
                    type="checkbox"
                    checked={remind}
                    onChange={(e) => setRemind(e.target.checked)}
                  />
                  {t('announcement.remind')}
                </label>
              )}
              <div className="flex gap-2">
                <Button type="submit" isDisabled={busy}>
                  {t('announcement.save')}
                </Button>
                <Button variant="outline" isDisabled={busy} onPress={() => setEditing(false)}>
                  {t('announcement.cancel')}
                </Button>
              </div>
            </form>
          ) : selected ? (
            <>
              <div className="flex items-start justify-between gap-4">
                <h2 className="text-2xl font-semibold leading-snug">{selected.title}</h2>
                {manage && (
                  <Button variant="outline" isDisabled={busy} onPress={edit}>
                    {t('announcement.edit')}
                  </Button>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                {new Date(selected.created_at).toLocaleString()}
              </p>
              <div
                className="announcement-content border-t pt-4"
                dangerouslySetInnerHTML={{ __html: selected.body_html ?? '' }}
              />
            </>
          ) : (
            <p className="text-muted-foreground">{t('announcement.select')}</p>
          )}
        </section>
      </div>
    </div>
  )
}
