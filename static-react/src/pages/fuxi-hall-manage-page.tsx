import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  refreshFuxiHallContributions,
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
import type {
  FuxiHallContributions,
  FuxiHallCard,
  FuxiHallCardCreate,
  FuxiHallPageConfig,
  FuxiHallPageKey,
} from '@/types/api/fuxi-hall'
import {
  CONTRIBUTION_KEYS,
  CONTRIBUTION_OFFSET_MAX,
  CONTRIBUTION_OFFSET_MIN,
  FLOOR_CAPACITY_MAX,
  FLOOR_CAPACITY_MIN,
  FONT_SCALE_MAX,
  FONT_SCALE_MIN,
  normalizeOffsetDraftsForPreview,
  parseIntegerInput,
  validateBoundedInteger,
  validateContributionOffsets,
  validateFloorDrafts,
  withPreviewCard,
  type ContributionOffsetDrafts,
  type NumericInputDraft,
  type PageFloorDraft,
} from './fuxi-hall-manage-helpers'
import { ShopDialog } from './shop-page-utils'

type PageDraft = Omit<FuxiHallPageConfig, 'floors'> & {
  floors: PageFloorDraft[]
}

type CardForm = {
  id: number
  nickname: string
  main_character_name: string
  title_tags: string[]
  description_html: string
  accent_color: string
  avatar_shape: 'circle' | 'rounded' | 'square'
  font_scale: NumericInputDraft
  visible: boolean
  floor_key: string
  is_current_management: boolean
  contribution_offsets: ContributionOffsetDrafts
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
  floor_key: 'first',
  is_current_management: false,
  contribution_offsets: { fleets: 0, welfare: 0, srp: 0, recruitment: 0, mentoring: 0 },
})
const contributionKeys = CONTRIBUTION_KEYS
const asPreview = (
  form: CardForm,
  pageKey: FuxiHallPageKey,
  existing?: FuxiHallCard
): FuxiHallCard => {
  const offsets = normalizeOffsetDraftsForPreview(form.contribution_offsets)
  return {
    ...existing,
    contributions: contributionKeys.reduce<FuxiHallContributions>(
      (totals, key) => {
        totals[key] =
          Math.max(
            0,
            (existing?.contributions?.[key] ?? 0) - (existing?.contribution_offsets?.[key] ?? 0)
          ) + offsets[key]
        return totals
      },
      {
        corp_pap: existing?.contributions?.corp_pap ?? 0,
        fleets: 0,
        welfare: 0,
        srp: 0,
        recruitment: 0,
        mentoring: 0,
      }
    ),
    ...form,
    font_scale: validateBoundedInteger(form.font_scale, FONT_SCALE_MIN, FONT_SCALE_MAX) ?? 14,
    contribution_offsets: offsets,
    page_key: pageKey,
    id: form.id || -1,
    main_character_id: 0,
    sort_order: 0,
    created_at: '',
    updated_at: '',
  }
}

export function FuxiHallManagePage() {
  const { t } = useI18n()
  const pageKey: FuxiHallPageKey = 'hall-of-fame'
  const [page, setPage] = useState<PageDraft | null>(null)
  const [cards, setCards] = useState<FuxiHallCard[]>([])
  const [refreshing, setRefreshing] = useState(false)
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
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.manage.loadFailed')
      )
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
    if (!dialogOpen || !form.nickname.trim()) return cards.filter((card) => card.visible)
    return withPreviewCard(
      cards,
      asPreview(
        form,
        pageKey,
        cards.find((card) => card.id === form.id)
      )
    )
  }, [cards, dialogOpen, form, pageKey])
  const savePage = async () => {
    if (!page?.title.trim()) return notifyError(t('fuxiHall.manage.titleRequired'))
    const validatedFloors = validateFloorDrafts(page.floors)
    if (!validatedFloors) return notifyError(t('fuxiHall.errors.floors'))
    setSaving(true)
    try {
      setPage(
        await updateFuxiHallPage(pageKey, {
          title: page.title.trim(),
          subtitle: page.subtitle.trim(),
          description_html: page.description_html,
          floors: validatedFloors,
        })
      )
      notifySuccess(t('fuxiHall.manage.saveSuccess'))
    } catch (caught) {
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.manage.saveFailed')
      )
    } finally {
      setSaving(false)
    }
  }
  const openCreate = () => {
    setForm({ ...emptyForm(), floor_key: page?.floors.at(-1)?.key ?? 'first' })
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
      floor_key: card.floor_key,
      is_current_management: card.is_current_management,
      contribution_offsets: { ...emptyForm().contribution_offsets, ...card.contribution_offsets },
    })
    setTag('')
    setDialogOpen(true)
  }
  const submitCard = async () => {
    if (
      !form.nickname.trim() ||
      !form.main_character_name.trim() ||
      (form.is_current_management && !form.title_tags.length)
    )
      return notifyError(t('fuxiHall.manage.requiredFields'))
    const fontScale = validateBoundedInteger(form.font_scale, FONT_SCALE_MIN, FONT_SCALE_MAX)
    if (fontScale === null) return notifyError(t('fuxiHall.errors.invalid'))
    const contributionOffsets = validateContributionOffsets(form.contribution_offsets)
    if (!contributionOffsets) return notifyError(t('fuxiHall.errors.offsets'))
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
        font_scale: fontScale,
        contribution_offsets: contributionOffsets,
        visible: form.visible,
        floor_key: form.floor_key,
        is_current_management: form.is_current_management,
      }
      if (form.id) {
        await updateFuxiHallCard(form.id, payload)
      } else await createFuxiHallCard(payload)
      setDialogOpen(false)
      await load()
      notifySuccess(t('fuxiHall.manage.saveSuccess'))
    } catch (caught) {
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.manage.saveFailed')
      )
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
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.manage.saveFailed')
      )
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
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.manage.sortFailed')
      )
      await load()
    }
  }
  const refresh = async () => {
    setRefreshing(true)
    try {
      await refreshFuxiHallContributions()
      setCards(await fetchFuxiHallCards(pageKey))
      notifySuccess(t('fuxiHall.contributions.refreshed'))
    } catch (caught) {
      notifyError(
        caught instanceof Error
          ? caught.message.startsWith('fuxiHall.')
            ? t(caught.message)
            : caught.message
          : t('fuxiHall.errors.refresh')
      )
    } finally {
      setRefreshing(false)
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
                <fieldset className="space-y-3">
                  <legend>{t('fuxiHall.fame.floorSettings')}</legend>
                  <p className="text-sm text-muted-foreground">{t('fuxiHall.fame.floorHint')}</p>
                  {page.floors.map((floor, index) => (
                    <div key={floor.key} className="flex flex-wrap gap-2">
                      <Input
                        aria-label={t('fuxiHall.fame.floorName')}
                        maxLength={64}
                        placeholder={
                          ['top', 'second', 'first'].includes(floor.key)
                            ? t(`fuxiHall.fame.floors.${floor.key}`)
                            : t('fuxiHall.fame.floorName')
                        }
                        value={floor.name}
                        onChange={(event) =>
                          setPage({
                            ...page,
                            floors: page.floors.map((item) =>
                              item.key === floor.key ? { ...item, name: event.target.value } : item
                            ),
                          })
                        }
                      />
                      <Input
                        aria-label={t('fuxiHall.fame.capacity')}
                        type="number"
                        min={FLOOR_CAPACITY_MIN}
                        max={FLOOR_CAPACITY_MAX}
                        step={1}
                        value={floor.capacity}
                        onChange={(event) =>
                          setPage({
                            ...page,
                            floors: page.floors.map((item) =>
                              item.key === floor.key
                                ? {
                                    ...item,
                                    capacity: parseIntegerInput(
                                      event.target.value,
                                      FLOOR_CAPACITY_MAX
                                    ),
                                  }
                                : item
                            ),
                          })
                        }
                      />
                      <Button
                        type="button"
                        aria-label={t('fuxiHall.fame.moveUp')}
                        isDisabled={index === 0}
                        onClick={() => {
                          const floors = [...page.floors]
                          ;[floors[index - 1], floors[index]] = [floors[index], floors[index - 1]]
                          setPage({ ...page, floors })
                        }}
                      >
                        ↑
                      </Button>
                      <Button
                        type="button"
                        aria-label={t('fuxiHall.fame.moveDown')}
                        isDisabled={index === page.floors.length - 1}
                        onClick={() => {
                          const floors = [...page.floors]
                          ;[floors[index + 1], floors[index]] = [floors[index], floors[index + 1]]
                          setPage({ ...page, floors })
                        }}
                      >
                        ↓
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() =>
                          setPage({
                            ...page,
                            floors: page.floors.filter((item) => item.key !== floor.key),
                          })
                        }
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    isDisabled={page.floors.length >= 12}
                    onClick={() =>
                      setPage({
                        ...page,
                        floors: [
                          ...page.floors,
                          {
                            key: `floor-${Date.now()}`,
                            name: t('fuxiHall.fame.newFloor'),
                            capacity: 12,
                          },
                        ],
                      })
                    }
                  >
                    {t('fuxiHall.fame.addFloor')}
                  </Button>
                </fieldset>
                <Button type="button" onClick={() => void savePage()} isDisabled={saving}>
                  {t('common.save')}
                </Button>
              </div>
            </div>
            <div className="rounded-lg border bg-card p-5">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold">{t('fuxiHall.manage.cardList')}</h2>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    isDisabled={refreshing || saving}
                    onClick={() => void refresh()}
                  >
                    {t(
                      refreshing
                        ? 'fuxiHall.contributions.refreshing'
                        : 'fuxiHall.contributions.refresh'
                    )}
                  </Button>
                  <Button type="button" onClick={openCreate}>
                    {t('fuxiHall.manage.addCard')}
                  </Button>
                </div>
              </div>
              <div className="mt-4 space-y-2">
                {cards.map((card, index) => (
                  <div
                    key={card.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded border p-3"
                  >
                    <div>
                      <p className="font-medium">{card.nickname}</p>
                      <p className="text-xs text-muted-foreground">
                        {page.floors.find((floor) => floor.key === card.floor_key)?.name ||
                          t(`fuxiHall.fame.floors.${card.floor_key}`)}{' '}
                        · {card.title_tags.join(' · ')}
                      </p>
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
                {page.floors.map((floor) => (
                  <section key={floor.key} className="col-span-full space-y-3">
                    <h4>
                      {floor.name || t(`fuxiHall.fame.floors.${floor.key}`)} ·{' '}
                      {previewCards.filter((card) => card.floor_key === floor.key).length}/
                      {floor.capacity === '' ? '—' : floor.capacity}
                    </h4>
                    <div
                      className="mx-auto grid w-full grid-cols-[repeat(auto-fit,minmax(min(100%,300px),1fr))] items-start gap-4"
                      style={{
                        maxWidth:
                          Math.max(
                            1,
                            previewCards.filter((card) => card.floor_key === floor.key).length
                          ) *
                            396 -
                          16,
                      }}
                    >
                      {previewCards
                        .filter((card) => card.floor_key === floor.key)
                        .map((card) => (
                          <FuxiHallMemberCard key={card.id} card={card} />
                        ))}
                    </div>
                  </section>
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
          widthClass="w-full sm:max-w-3xl"
        >
          <div className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span>{t('fuxiHall.fame.floor')}</span>
                <Select
                  selectedKey={form.floor_key}
                  onSelectionChange={(key) => setForm({ ...form, floor_key: String(key) })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {page?.floors.map((floor) => (
                      <SelectItem key={floor.key} id={floor.key}>
                        {floor.name || t(`fuxiHall.fame.floors.${floor.key}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>
              <label className="flex items-center gap-2">
                <Checkbox
                  isSelected={form.is_current_management}
                  onChange={(value) => setForm({ ...form, is_current_management: value })}
                />
                {t('fuxiHall.fame.current')}
              </label>
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
                      variant="outline"
                      size="sm"
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
                  onSelectionChange={(key) =>
                    ((value) =>
                      setForm({
                        ...form,
                        avatar_shape: value as CardForm['avatar_shape'],
                      }))(String(key))
                  }
                >
                  <SelectTrigger className="h-10 w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem id="circle">{t('fuxiHall.manage.avatarShapeCircle')}</SelectItem>
                    <SelectItem id="rounded">{t('fuxiHall.manage.avatarShapeRounded')}</SelectItem>
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
                  min={FONT_SCALE_MIN}
                  max={FONT_SCALE_MAX}
                  step={1}
                  value={form.font_scale}
                  onChange={(event) =>
                    setForm({
                      ...form,
                      font_scale: parseIntegerInput(event.target.value, FONT_SCALE_MAX),
                    })
                  }
                />
              </label>
              <label className="flex items-center gap-2 pt-7">
                <Checkbox
                  isSelected={form.visible}
                  onChange={(visible) => setForm({ ...form, visible: visible === true })}
                />
                {t('fuxiHall.manage.visible')}
              </label>
              <fieldset className="space-y-3 md:col-span-2">
                <legend className="font-medium">{t('fuxiHall.contributions.offsets')}</legend>
                <p className="text-xs text-muted-foreground">
                  {t('fuxiHall.contributions.offsetHint')}
                </p>
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                  {contributionKeys.map((key) => (
                    <label key={key} className="space-y-1">
                      <span className="text-sm">{t(`fuxiHall.contributions.${key}`)}</span>
                      <Input
                        type="number"
                        min={CONTRIBUTION_OFFSET_MIN}
                        max={CONTRIBUTION_OFFSET_MAX}
                        step={1}
                        value={form.contribution_offsets[key]}
                        onChange={(event) =>
                          setForm({
                            ...form,
                            contribution_offsets: {
                              ...form.contribution_offsets,
                              [key]: parseIntegerInput(event.target.value, CONTRIBUTION_OFFSET_MAX),
                            },
                          })
                        }
                      />
                    </label>
                  ))}
                </div>
              </fieldset>
              <div className="space-y-2 md:col-span-2">
                <span className="text-sm">{t('fuxiHall.manage.cardDescription')}</span>
                <RichTextEditor
                  ariaLabel={t('fuxiHall.manage.cardDescription')}
                  value={form.description_html ?? ''}
                  onChange={(value) => setForm({ ...form, description_html: value })}
                />
              </div>
            </div>
            <div className="flex justify-end gap-2">
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
