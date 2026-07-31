import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  createFuxiHallCard,
  deleteFuxiHallCard,
  fetchFuxiHallCards,
  fetchFuxiHallPage,
  reorderFuxiHallCards,
  updateFuxiHallCard,
  updateFuxiHallPage,
} from '@/api/fuxi-hall'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { RichTextEditor } from '@/components/rich-text-editor'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { confirmAction, notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type {
  FuxiHallCard,
  FuxiHallCardCreate,
  FuxiHallPageConfig,
  FuxiHallPageKey,
} from '@/types/api/fuxi-hall'
import { ShopDialog } from './shop-page-utils'

type CardForm = {
  id: number
  nickname: string
  main_character_name: string
  title_tags: string[]
  description_html: string
  accent_color: string
  avatar_shape: 'circle' | 'rounded' | 'square'
  font_scale: number
  visible: boolean
  welfare_delivery_offset: number
}
const emptyForm = (): CardForm => ({
  id: 0,
  nickname: '',
  main_character_name: '',
  title_tags: [],
  description_html: '',
  accent_color: '#3b82f6',
  avatar_shape: 'circle',
  font_scale: 14,
  visible: true,
  welfare_delivery_offset: 0,
})
const asPreview = (form: CardForm, pageKey: FuxiHallPageKey): FuxiHallCard => ({
  ...form,
  page_key: pageKey,
  id: form.id || -1,
  main_character_id: 0,
  sort_order: 0,
  created_at: '',
  updated_at: '',
})

export function FuxiHallManagePage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const isSuperAdmin = roles.includes('super_admin')
  const [pageKey, setPageKey] = useState<FuxiHallPageKey>('leadership')
  const [page, setPage] = useState<FuxiHallPageConfig | null>(null)
  const [cards, setCards] = useState<FuxiHallCard[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState<CardForm>(emptyForm)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [tag, setTag] = useState('')
  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [nextPage, nextCards] = await Promise.all([
        fetchFuxiHallPage(pageKey),
        fetchFuxiHallCards(pageKey),
      ])
      setPage(nextPage)
      setCards(nextCards)
    } catch (caught) {
      notifyError(caught instanceof Error ? caught.message : t('fuxiHall.manage.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [pageKey, t])
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [load])
  const previewCards = useMemo(() => {
    const visible = cards.filter((card) => card.visible)
    if (!dialogOpen || !form.nickname.trim() || !form.visible) return visible
    const preview = asPreview(form, pageKey)
    return form.id
      ? visible.map((card) => (card.id === form.id ? preview : card))
      : [...visible, preview]
  }, [cards, dialogOpen, form, pageKey])
  const savePage = async () => {
    if (!page?.title.trim()) return notifyError(t('fuxiHall.manage.titleRequired'))
    setSaving(true)
    try {
      setPage(
        await updateFuxiHallPage(pageKey, {
          title: page.title.trim(),
          subtitle: page.subtitle.trim(),
          description_html: page.description_html,
        })
      )
      notifySuccess(t('fuxiHall.manage.saveSuccess'))
    } catch (caught) {
      notifyError(caught instanceof Error ? caught.message : t('fuxiHall.manage.saveFailed'))
    } finally {
      setSaving(false)
    }
  }
  const openCreate = () => {
    setForm(emptyForm())
    setTag('')
    setDialogOpen(true)
  }
  const openEdit = (card: FuxiHallCard) => {
    setForm({
      id: card.id,
      nickname: card.nickname,
      main_character_name: card.main_character_name,
      title_tags: [...card.title_tags],
      description_html: card.description_html,
      accent_color: card.accent_color,
      avatar_shape: card.avatar_shape,
      font_scale: card.font_scale,
      visible: card.visible,
      welfare_delivery_offset: card.welfare_delivery_offset ?? 0,
    })
    setTag('')
    setDialogOpen(true)
  }
  const submitCard = async () => {
    if (!form.nickname.trim() || !form.main_character_name.trim() || !form.title_tags.length)
      return notifyError(t('fuxiHall.manage.requiredFields'))
    setSaving(true)
    try {
      const payload: FuxiHallCardCreate = {
        page_key: pageKey,
        nickname: form.nickname.trim(),
        main_character_name: form.main_character_name.trim(),
        title_tags: form.title_tags,
        description_html: form.description_html,
        accent_color: form.accent_color,
        avatar_shape: form.avatar_shape,
        font_scale: form.font_scale,
        visible: form.visible,
      }
      if (form.id) {
        await updateFuxiHallCard(
          form.id,
          isSuperAdmin
            ? { ...payload, welfare_delivery_offset: form.welfare_delivery_offset }
            : payload
        )
      } else await createFuxiHallCard(payload)
      setDialogOpen(false)
      await load()
      notifySuccess(t('fuxiHall.manage.saveSuccess'))
    } catch (caught) {
      notifyError(caught instanceof Error ? caught.message : t('fuxiHall.manage.saveFailed'))
    } finally {
      setSaving(false)
    }
  }
  const removeCard = async (card: FuxiHallCard) => {
    if (
      !(await confirmAction({
        title: t('common.delete'),
        message: t('fuxiHall.manage.deleteConfirm'),
        confirmText: t('common.delete'),
        cancelText: t('common.cancel'),
      }))
    )
      return
    try {
      await deleteFuxiHallCard(card.id)
      await load()
      notifySuccess(t('fuxiHall.manage.deleteSuccess'))
    } catch (caught) {
      notifyError(caught instanceof Error ? caught.message : t('fuxiHall.manage.saveFailed'))
    }
  }
  const move = async (index: number, delta: number) => {
    const target = index + delta
    if (target < 0 || target >= cards.length) return
    const next = [...cards]
    ;[next[index], next[target]] = [next[target], next[index]]
    setCards(next)
    try {
      await reorderFuxiHallCards({ page_key: pageKey, ordered_ids: next.map((card) => card.id) })
    } catch (caught) {
      notifyError(caught instanceof Error ? caught.message : t('fuxiHall.manage.sortFailed'))
      await load()
    }
  }
  const addTag = () => {
    const next = tag.trim()
    if (next && !form.title_tags.includes(next))
      setForm((current) => ({ ...current, title_tags: [...current.title_tags, next] }))
    setTag('')
  }
  return (
    <section className="space-y-4">
      <div className="flex gap-2 border-b">
        {(['leadership', 'contributors'] as const).map((key) => (
          <Button
            key={key}
            type="button"
            variant={pageKey === key ? 'default' : 'ghost'}
            onClick={() => setPageKey(key)}
          >
            {t(`fuxiHall.manage.${key}Tab`)}
          </Button>
        ))}
      </div>
      {loading ? <p className="text-sm text-muted-foreground">{t('common.loading')}</p> : null}
      {page ? (
        <div className="grid gap-4 xl:grid-cols-2">
          <div className="space-y-4">
            <div className="rounded-lg border bg-card p-5">
              <h1 className="text-xl font-semibold">{t('fuxiHall.manage.pageConfig')}</h1>
              <div className="mt-4 space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm">{t('fuxiHall.manage.pageTitle')}</span>
                  <Input
                    value={page.title}
                    onChange={(event) => setPage({ ...page, title: event.target.value })}
                  />
                </label>
                <label className="block space-y-2">
                  <span className="text-sm">{t('fuxiHall.manage.pageSubtitle')}</span>
                  <Input
                    value={page.subtitle}
                    onChange={(event) => setPage({ ...page, subtitle: event.target.value })}
                  />
                </label>
                <div>
                  <span className="text-sm">{t('fuxiHall.manage.pageDescription')}</span>
                  <div className="mt-2">
                    <RichTextEditor
                      ariaLabel={t('fuxiHall.manage.pageDescription')}
                      value={page.description_html}
                      onChange={(value) => setPage({ ...page, description_html: value })}
                    />
                  </div>
                </div>
                <Button type="button" onClick={() => void savePage()} isDisabled={saving}>
                  {t('common.save')}
                </Button>
              </div>
            </div>
            <div className="rounded-lg border bg-card p-5">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold">{t('fuxiHall.manage.cardList')}</h2>
                <Button type="button" onClick={openCreate}>
                  {t('fuxiHall.manage.addCard')}
                </Button>
              </div>
              <div className="mt-4 space-y-2">
                {cards.map((card, index) => (
                  <div
                    key={card.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded border p-3"
                  >
                    <div>
                      <p className="font-medium">{card.nickname}</p>
                      <p className="text-xs text-muted-foreground">{card.title_tags.join(' · ')}</p>
                    </div>
                    <div className="flex gap-1">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void move(index, -1)}
                      >
                        ↑
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void move(index, 1)}
                      >
                        ↓
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => openEdit(card)}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="destructive"
                        onClick={() => void removeCard(card)}
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
          <div className="rounded-lg border bg-card p-5">
            <h2 className="font-semibold">{t('fuxiHall.manage.previewPanel')}</h2>
            <h3 className="mt-4 text-xl font-semibold">
              {page.title || t('fuxiHall.manage.previewFallbackTitle')}
            </h3>
            {page.subtitle ? <p className="text-muted-foreground">{page.subtitle}</p> : null}
            {page.description_html ? (
              <div
                className="mt-4 rounded bg-muted/50 p-3 [&_img]:max-w-full"
                dangerouslySetInnerHTML={{ __html: page.description_html }}
              />
            ) : null}
            {previewCards.length ? (
              <div className="mt-4 grid gap-3 md:grid-cols-2">
                {previewCards.map((card) => (
                  <FuxiHallMemberCard key={card.id} card={card} showStats />
                ))}
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground">
                {t('fuxiHall.manage.previewEmpty')}
              </p>
            )}
          </div>
        </div>
      ) : null}
      {dialogOpen ? (
        <ShopDialog
          open={dialogOpen}
          title={form.id ? t('fuxiHall.manage.editCard') : t('fuxiHall.manage.addCard')}
          onClose={() => setDialogOpen(false)}
          closeLabel={t('common.close')}
          widthClass="max-w-3xl"
        >
          <div className="mx-auto my-8 max-w-3xl rounded-lg border bg-card p-5">
            <h2 className="text-lg font-semibold">
              {form.id ? t('fuxiHall.manage.editCard') : t('fuxiHall.manage.addCard')}
            </h2>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm">{t('fuxiHall.manage.nickname')}</span>
                <Input
                  value={form.nickname}
                  onChange={(event) => setForm({ ...form, nickname: event.target.value })}
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm">{t('fuxiHall.manage.mainCharacterName')}</span>
                <Input
                  value={form.main_character_name}
                  onChange={(event) =>
                    setForm({ ...form, main_character_name: event.target.value })
                  }
                />
              </label>
              <div className="space-y-2 md:col-span-2">
                <span className="text-sm">{t('fuxiHall.manage.title')}</span>
                <div className="flex gap-2">
                  <Input
                    value={tag}
                    placeholder={t('fuxiHall.manage.titleTagPlaceholder')}
                    onChange={(event) => setTag(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') {
                        event.preventDefault()
                        addTag()
                      }
                    }}
                  />
                  <Button type="button" onClick={addTag}>
                    {t('fuxiHall.manage.addTitleTag')}
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {form.title_tags.map((item) => (
                    <Button
                      key={item}
                      type="button"
                      className="rounded-full border px-2 py-1 text-xs"
                      onClick={() =>
                        setForm({
                          ...form,
                          title_tags: form.title_tags.filter((value) => value !== item),
                        })
                      }
                    >
                      {item} ×
                    </Button>
                  ))}
                </div>
              </div>
              <label className="space-y-2">
                <span className="text-sm">{t('fuxiHall.manage.avatarShape')}</span>
                <Select
                  selectedKey={String(form.avatar_shape ?? '')}
                  onSelectionChange={(key) => ((value) =>
                    setForm({
                      ...form,
                      avatar_shape: value as CardForm['avatar_shape'],
                    }))(String(key))}
                >
                  <SelectTrigger className="h-10 w-full rounded-md border bg-background px-3">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem id="circle">{t('fuxiHall.manage.avatarShapeCircle')}</SelectItem>
                    <SelectItem id="rounded">
                      {t('fuxiHall.manage.avatarShapeRounded')}
                    </SelectItem>
                    <SelectItem id="square">{t('fuxiHall.manage.avatarShapeSquare')}</SelectItem>
                  </SelectContent>
                </Select>
              </label>
              <label className="space-y-2">
                <span className="text-sm">{t('fuxiHall.manage.accentColor')}</span>
                <Input
                  type="color"
                  value={form.accent_color}
                  onChange={(event) => setForm({ ...form, accent_color: event.target.value })}
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm">{t('fuxiHall.manage.fontScale')}</span>
                <Input
                  type="number"
                  min={12}
                  max={20}
                  value={form.font_scale}
                  onChange={(event) => setForm({ ...form, font_scale: Number(event.target.value) })}
                />
              </label>
              <label className="flex items-center gap-2 pt-7">
                <Checkbox
                  isSelected={form.visible}
                  onChange={(visible) => setForm({ ...form, visible: visible === true })}
                />
                {t('fuxiHall.manage.visible')}
              </label>
              {form.id && isSuperAdmin ? (
                <label className="space-y-2">
                  <span className="text-sm">{t('fuxiHall.manage.welfareDeliveryOffset')}</span>
                  <Input
                    type="number"
                    min={0}
                    value={form.welfare_delivery_offset}
                    onChange={(event) =>
                      setForm({
                        ...form,
                        welfare_delivery_offset: Math.max(0, Number(event.target.value)),
                      })
                    }
                  />
                </label>
              ) : null}
              <div className="space-y-2 md:col-span-2">
                <span className="text-sm">{t('fuxiHall.manage.cardDescription')}</span>
                <RichTextEditor
                  ariaLabel={t('fuxiHall.manage.cardDescription')}
                  value={form.description_html ?? ''}
                  onChange={(value) => setForm({ ...form, description_html: value })}
                />
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" isDisabled={saving} onClick={() => void submitCard()}>
                {t('common.save')}
              </Button>
            </div>
          </div>
        </ShopDialog>
      ) : null}
    </section>
  )
}
