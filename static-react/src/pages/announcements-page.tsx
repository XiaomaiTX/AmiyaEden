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
import { Clock3, FileText, Megaphone, UserRound } from 'lucide-react'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import './announcements.css'

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
    <div className="announcement-board">
      <header className="announcement-masthead">
        <div className="announcement-heading">
          <span className="announcement-mark">
            <Megaphone aria-hidden="true" />
          </span>
          <h1>{t('announcement.title')}</h1>
        </div>
        {manage && (
          <Button
            className="announcement-publish"
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
      <div className="announcement-layout">
        <aside className="announcement-index">
          <div className="announcement-index-heading">
            <h2>
              <FileText aria-hidden="true" />
              {t('announcement.archive')}
            </h2>
            <span>{total}</span>
          </div>
          {rows.length === 0 && <p className="announcement-empty">{t('announcement.empty')}</p>}
          <div className="announcement-entries">
            {rows.map((row) => {
              const isUnread = Boolean(row.unread && !row.hidden)
              return (
                <button
                  key={row.id}
                  disabled={busy}
                  aria-current={selected?.id === row.id ? 'true' : undefined}
                  onClick={() => void open(row)}
                  className={`announcement-entry ${isUnread ? 'is-unread' : ''} ${selected?.id === row.id ? 'is-selected' : ''}`}
                >
                  <div className="announcement-entry-heading">
                    <span className="announcement-entry-title">
                      {isUnread && <span aria-hidden="true" className="announcement-unread-dot" />}
                      <span className="announcement-title-text">{row.title}</span>
                    </span>
                    {isUnread && (
                      <span className="announcement-status">{t('announcement.unread')}</span>
                    )}
                    {row.hidden && (
                      <span className="announcement-status is-hidden">
                        {t('announcement.hidden')}
                      </span>
                    )}
                  </div>
                  <span className="announcement-entry-author">
                    <span
                      className="announcement-nickname"
                      title={row.author_nickname || t('announcement.unknownAuthor')}
                    >
                      {row.author_nickname || t('announcement.unknownAuthor')}
                    </span>
                    {row.author_character_name && (
                      <span className="announcement-character" title={row.author_character_name}>
                        {row.author_character_name}
                      </span>
                    )}
                  </span>
                  <time className="announcement-entry-time" dateTime={row.created_at}>
                    {new Date(row.created_at).toLocaleString()}
                  </time>
                </button>
              )
            })}
          </div>
          <div className="announcement-pagination">
            <Button
              variant="outline"
              isDisabled={page === 1 || busy}
              onPress={() => setPage(page - 1)}
            >
              {t('announcement.previous')}
            </Button>
            <Button
              variant="outline"
              isDisabled={page * 10 >= total || busy}
              onPress={() => setPage(page + 1)}
            >
              {t('announcement.next')}
            </Button>
          </div>
        </aside>
        <section aria-busy={busy} className="announcement-reader">
          {editing ? (
            <form
              className="announcement-editor space-y-4"
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
              <header className="announcement-article-heading">
                <h2>{selected.title}</h2>
                {manage && (
                  <Button variant="outline" isDisabled={busy} onPress={edit}>
                    {t('announcement.edit')}
                  </Button>
                )}
              </header>
              <div className="announcement-byline">
                <div className="announcement-author">
                  <span className="announcement-avatar">
                    {selected.author_character_id > 0 ? (
                      <img
                        src={buildEveCharacterPortraitUrl(selected.author_character_id, 64)}
                        alt=""
                      />
                    ) : (
                      <UserRound aria-hidden="true" />
                    )}
                  </span>
                  <div className="announcement-author-names">
                    <span className="announcement-author-label">{t('announcement.author')}</span>
                    <span className="announcement-author-identity">
                      <strong
                        className="announcement-nickname"
                        title={selected.author_nickname || t('announcement.unknownAuthor')}
                      >
                        {selected.author_nickname || t('announcement.unknownAuthor')}
                      </strong>
                      {selected.author_character_name && (
                        <span
                          className="announcement-character"
                          title={selected.author_character_name}
                        >
                          {selected.author_character_name}
                        </span>
                      )}
                    </span>
                  </div>
                </div>
                <time className="announcement-published" dateTime={selected.created_at}>
                  <Clock3 aria-hidden="true" />
                  <span>
                    {t('announcement.publishedAt')}
                    <strong>{new Date(selected.created_at).toLocaleString()}</strong>
                  </span>
                </time>
              </div>
              <div
                className="announcement-content announcement-body"
                dangerouslySetInnerHTML={{ __html: selected.body_html ?? '' }}
              />
            </>
          ) : (
            <div className="announcement-reader-empty">
              <Megaphone aria-hidden="true" />
              <p>{t(rows.length ? 'announcement.select' : 'announcement.empty')}</p>
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
