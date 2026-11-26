import { useCallback, useEffect, useState } from 'react'
import { getHall, updateHallOwnCard } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { FuxiHallBanner } from '@/components/fuxi-hall-banner'
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
      setData(await unwrapContract(getHall(), 'fuxiHall.errors.load'))
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
      const card = await unwrapContract(
        updateHallOwnCard(editing.id, {
          nickname,
          description_html: description,
        }),
        'fuxiHall.errors.save'
      )
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
    <section className="fx-gallery">
      <FuxiHallBanner
        title={data?.page.title || t('fuxiHall.fame.title')}
        subtitle={data?.page.subtitle}
      >
        {data && data.page.floors.length > 1 && (
          <nav className="fx-index" aria-label={t('fuxiHall.fame.floorIndex')}>
            <span>{t('fuxiHall.fame.floorIndex')}</span>
            {data.page.floors.map((floor) => (
              <button
                key={floor.key}
                type="button"
                aria-controls={`fx-floor-${floor.key}`}
                onClick={() => {
                  const section = document.getElementById(`fx-floor-${floor.key}`)
                  section?.focus({ preventScroll: true })
                  section?.scrollIntoView({ block: 'start' })
                }}
              >
                {floor.name || t(`fuxiHall.fame.floors.${floor.key}`)}
              </button>
            ))}
          </nav>
        )}
      </FuxiHallBanner>
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
          className="fx-intro"
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
            id={`fx-floor-${floor.key}`}
            tabIndex={-1}
            className="fx-floor"
          >
            <div className="fx-floor__heading">
              <h2>{floor.name || t(`fuxiHall.fame.floors.${floor.key}`)}</h2>
              <span>{t('fuxiHall.fame.memberCount', { count: cards.length })}</span>
            </div>
            <div className="fx-grid" style={{ maxWidth: Math.max(1, cards.length) * 396 - 16 }}>
              {cards.map((card) => (
                <FuxiHallMemberCard
                  key={card.id}
                  card={card}
                  action={
                    card.can_edit && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="w-full"
                        onClick={() => edit(card)}
                      >
                        {t('fuxiHall.fame.editMine')}
                      </Button>
                    )
                  }
                />
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
