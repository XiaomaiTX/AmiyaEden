import { useCallback, useEffect, useState } from 'react'
import { fetchFuxiHall, updateOwnFuxiHallCard } from '@/api/fuxi-hall'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { RichTextEditor } from '@/components/rich-text-editor'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { ShopDialog } from './shop-page-utils'
import type { FuxiHallCard, FuxiHallPublicPageResponse } from '@/types/api/fuxi-hall'

export function FuxiHallPublicPage() {
  const { t } = useI18n()
  const [data, setData] = useState<FuxiHallPublicPageResponse | null>(null)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState<FuxiHallCard | null>(null)
  const [nickname, setNickname] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)
  const [editError, setEditError] = useState('')
  const message = useCallback(
    (caught: unknown) => {
      const value = caught instanceof Error ? caught.message : 'fuxiHall.errors.load'
      return value.startsWith('fuxiHall.') ? t(value) : value
    },
    [t]
  )
  const load = useCallback(async () => {
    setError('')
    try {
      setData(await fetchFuxiHall())
    } catch (caught) {
      setError(message(caught))
    }
  }, [message])
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [load])
  const edit = (card: FuxiHallCard) => {
    setEditing(card)
    setNickname(card.nickname)
    setDescription(card.description_html)
    setEditError('')
  }
  const save = async () => {
    if (!editing || saving) return
    setSaving(true)
    setEditError('')
    try {
      const card = await updateOwnFuxiHallCard(editing.id, {
        nickname,
        description_html: description,
      })
      setData((previous) =>
        previous
          ? {
              ...previous,
              cards: previous.cards.map((row) =>
                row.id === card.id
                  ? { ...row, nickname: card.nickname, description_html: card.description_html }
                  : row
              ),
            }
          : previous
      )
      setEditing(null)
    } catch (caught) {
      setEditError(message(caught))
    } finally {
      setSaving(false)
    }
  }
  return (
    <section className="min-w-0 space-y-8 p-4 md:p-6">
      <header className="mx-auto max-w-5xl text-center">
        <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">
          {data?.page.title || t('fuxiHall.fame.title')}
        </h1>
        {data?.page.subtitle && <p className="mt-3 text-muted-foreground">{data.page.subtitle}</p>}
        <p className="mt-3 text-xs text-muted-foreground">{t('fuxiHall.contributions.daily')}</p>
      </header>
      {error && (
        <div role="alert" className="text-center text-destructive">
          {error}{' '}
          <Button variant="outline" onClick={() => void load()}>
            {t('fuxiHall.fame.retry')}
          </Button>
        </div>
      )}
      {!data && !error && <p>{t('common.loading')}</p>}
      {data?.page.description_html && (
        <article
          className="mx-auto max-w-5xl leading-relaxed [&_img]:max-w-full"
          dangerouslySetInnerHTML={{ __html: data.page.description_html }}
        />
      )}
      {data && data.cards.length === 0 && (
        <p className="py-12 text-center text-muted-foreground">{t('fuxiHall.public.emptyTitle')}</p>
      )}
      {data?.page.floors.map((floor) => {
        const cards = data.cards.filter((card) => card.floor_key === floor.key)
        return (
          <section
            key={floor.key}
            aria-label={floor.name || t(`fuxiHall.fame.floors.${floor.key}`)}
            className="mx-auto w-full max-w-[1440px] space-y-4"
          >
            <div className="flex items-baseline justify-between border-b border-border pb-3">
              <h2 className="text-xl font-semibold">
                {floor.name || t(`fuxiHall.fame.floors.${floor.key}`)}
              </h2>
              <span className="text-sm text-muted-foreground">
                {t('fuxiHall.fame.memberCount', { count: cards.length })}
              </span>
            </div>
            <div
              className="mx-auto grid w-full grid-cols-[repeat(auto-fit,minmax(min(100%,300px),1fr))] items-start gap-4"
              style={{ maxWidth: Math.max(1, cards.length) * 396 - 16 }}
            >
              {cards.map((card) => (
                <div key={card.id} className="flex min-w-0 flex-col gap-2">
                  <div className="min-w-0">
                    <FuxiHallMemberCard card={card} />
                  </div>
                  {card.can_edit && (
                    <Button variant="outline" size="sm" onClick={() => edit(card)}>
                      {t('fuxiHall.fame.editMine')}
                    </Button>
                  )}
                </div>
              ))}
            </div>
          </section>
        )
      })}
      {editing && (
        <ShopDialog
          open
          title={t('fuxiHall.fame.editMine')}
          onClose={() => {
            if (!saving) setEditing(null)
          }}
          closeLabel={t('common.cancel')}
        >
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              void save()
            }}
          >
            <label className="block space-y-2">
              <span>{t('fuxiHall.manage.nickname')}</span>
              <Input
                maxLength={256}
                required
                value={nickname}
                disabled={saving}
                onChange={(event) => setNickname(event.target.value)}
              />
            </label>
            <div className="space-y-2">
              <p>{t('fuxiHall.manage.cardDescription')}</p>
              <RichTextEditor
                ariaLabel={t('fuxiHall.manage.cardDescription')}
                value={description}
                onChange={setDescription}
              />
            </div>
            {editError && (
              <p role="alert" className="text-destructive">
                {editError}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <Button variant="outline" isDisabled={saving} onClick={() => setEditing(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" isDisabled={saving || !nickname.trim()}>
                {t('common.save')}
              </Button>
            </div>
          </form>
        </ShopDialog>
      )}
    </section>
  )
}
