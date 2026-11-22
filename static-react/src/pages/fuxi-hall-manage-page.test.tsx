import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nProvider } from '@/i18n'
import { FuxiHallManagePage } from '@/pages/fuxi-hall-manage-page'
import {
  parseIntegerInput,
  validateBoundedInteger,
  validateContributionOffsets,
  validateFloorDrafts,
  withPreviewCard,
} from '@/pages/fuxi-hall-manage-helpers'
import {
  createFuxiHallCard,
  fetchFuxiHallCards,
  fetchFuxiHallPage,
  updateFuxiHallPage,
} from '@/api/fuxi-hall'
import { notifyError, notifySuccess } from '@/feedback'
import type { FuxiHallCard, FuxiHallPageConfig } from '@/types/api/fuxi-hall'

vi.mock('@/api/fuxi-hall', () => ({
  refreshFuxiHallContributions: vi.fn(),
  createFuxiHallCard: vi.fn(),
  deleteFuxiHallCard: vi.fn(),
  fetchFuxiHallCards: vi.fn(),
  fetchFuxiHallPage: vi.fn(),
  reorderFuxiHallCards: vi.fn(),
  updateFuxiHallCard: vi.fn(),
  updateFuxiHallPage: vi.fn(),
}))

vi.mock('@/feedback', () => ({
  confirmAction: vi.fn(),
  notifyError: vi.fn(),
  notifySuccess: vi.fn(),
}))

vi.mock('@/components/rich-text-editor', () => ({
  RichTextEditor: ({
    value,
    onChange,
    ariaLabel,
  }: {
    value: string
    onChange: (value: string) => void
    ariaLabel: string
  }) => (
    <textarea
      aria-label={ariaLabel}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  ),
}))

const samplePage: FuxiHallPageConfig = {
  id: 1,
  page_key: 'hall-of-fame',
  title: '伏羲名人堂',
  subtitle: '',
  description_html: '',
  floors: [{ key: 'first', name: '一楼', capacity: 12 }],
}

const sampleCards: FuxiHallCard[] = []

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(fetchFuxiHallPage).mockResolvedValue(samplePage)
  vi.mocked(fetchFuxiHallCards).mockResolvedValue(sampleCards)
  vi.mocked(updateFuxiHallPage).mockResolvedValue(samplePage)
})

test('floor preview applies visibility changes from the draft card', () => {
  const card = (id: number, visible: boolean) => ({ id, visible, floor_key: 'first' }) as FuxiHallCard
  const shown = card(1, true)
  const hidden = card(2, false)
  const ids = (cards: FuxiHallCard[]) => cards.map((item) => item.id)

  expect(ids(withPreviewCard([shown, hidden], { ...shown, visible: false }))).toEqual([])
  expect(ids(withPreviewCard([shown, hidden], { ...hidden, visible: true }))).toEqual([1, 2])
  expect(ids(withPreviewCard([shown], card(-1, true)))).toEqual([1, -1])
  expect(ids(withPreviewCard([shown], card(-1, false)))).toEqual([1])
})

test('integer helpers truncate decimals, allow partial lower typing, and validate bounds on save', () => {
  expect(parseIntegerInput('1', 20)).toBe(1)
  expect(parseIntegerInput('16', 20)).toBe(16)
  expect(parseIntegerInput('2.5', 10000)).toBe(2)
  expect(parseIntegerInput('0', 10000)).toBe(0)
  expect(parseIntegerInput('99999', 10000)).toBe(10000)
  expect(parseIntegerInput('', 10000)).toBe('')

  expect(validateBoundedInteger(1, 12, 20)).toBeNull()
  expect(validateBoundedInteger(16, 12, 20)).toBe(16)
  expect(validateFloorDrafts([{ key: 'first', name: '', capacity: '' }])).toBeNull()
  expect(validateFloorDrafts([{ key: 'first', name: '', capacity: 0 }])).toBeNull()
  expect(validateFloorDrafts([{ key: 'first', name: '', capacity: 2.5 }])).toBeNull()
  expect(validateFloorDrafts([{ key: 'first', name: '', capacity: 10 }])).toEqual([
    { key: 'first', name: '', capacity: 10 },
  ])

  expect(
    validateContributionOffsets({
      fleets: '',
      welfare: 0,
      srp: 0,
      recruitment: 0,
      mentoring: 0,
    })
  ).toBeNull()
})

test('floor capacity input truncates decimals and blocks empty submission before API call', async () => {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <FuxiHallManagePage />
    </I18nProvider>
  )

  const capacityInput = await screen.findByLabelText('可见卡片人数上限')
  fireEvent.change(capacityInput, { target: { value: '2.5' } })
  expect(capacityInput).toHaveValue(2)

  await user.click(screen.getByRole('button', { name: '保存' }))
  await waitFor(() => expect(updateFuxiHallPage).toHaveBeenCalledTimes(1))
  expect(updateFuxiHallPage).toHaveBeenCalledWith('hall-of-fame', {
    title: '伏羲名人堂',
    subtitle: '',
    description_html: '',
    floors: [{ key: 'first', name: '一楼', capacity: 2 }],
  })
  expect(notifySuccess).toHaveBeenCalled()

  fireEvent.change(capacityInput, { target: { value: '' } })
  await user.click(screen.getByRole('button', { name: '保存' }))
  expect(updateFuxiHallPage).toHaveBeenCalledTimes(1)
  expect(notifyError).toHaveBeenCalledWith('请设置1至12个有效楼层，每层人数上限为1至10000')
})

test('card offset input truncates decimals and blocks empty submission before API call', async () => {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <FuxiHallManagePage />
    </I18nProvider>
  )

  await user.click(await screen.findByRole('button', { name: '新增卡片' }))
  const dialog = screen.getByRole('dialog')
  await user.type(within(dialog).getByLabelText('昵称'), 'Alpha')
  await user.type(within(dialog).getByLabelText('主角色名称'), 'MainChar')

  const fleetsInput = within(dialog).getByLabelText('带队')
  fireEvent.change(fleetsInput, { target: { value: '7.9' } })
  expect(fleetsInput).toHaveValue(7)

  fireEvent.change(fleetsInput, { target: { value: '' } })
  await user.click(within(dialog).getByRole('button', { name: '保存' }))
  expect(createFuxiHallCard).not.toHaveBeenCalled()
  expect(notifyError).toHaveBeenCalledWith('历史补录次数须为0至10亿的整数')
})

test('font scale input supports multi-digit typing like 16 and blocks out-of-range values on save', async () => {
  const user = userEvent.setup()
  vi.mocked(createFuxiHallCard).mockResolvedValue({
    id: 10,
    page_key: 'hall-of-fame',
    nickname: 'Alpha',
    main_character_id: 1,
    main_character_name: 'MainChar',
    title_tags: [],
    description_html: '',
    accent_color: '#3b82f6',
    avatar_shape: 'circle',
    font_scale: 16,
    visible: true,
    floor_key: 'first',
    is_current_management: false,
    sort_order: 1,
    created_at: '',
    updated_at: '',
  })

  render(
    <I18nProvider>
      <FuxiHallManagePage />
    </I18nProvider>
  )

  await user.click(await screen.findByRole('button', { name: '新增卡片' }))
  const dialog = screen.getByRole('dialog')
  await user.type(within(dialog).getByLabelText('昵称'), 'Alpha')
  await user.type(within(dialog).getByLabelText('主角色名称'), 'MainChar')

  const fontScaleInput = within(dialog).getByLabelText('字体大小')
  await user.clear(fontScaleInput)
  await user.type(fontScaleInput, '16')
  expect(fontScaleInput).toHaveValue(16)

  await user.click(within(dialog).getByRole('button', { name: '保存' }))
  await waitFor(() => expect(createFuxiHallCard).toHaveBeenCalledTimes(1))
  expect(createFuxiHallCard).toHaveBeenCalledWith(
    expect.objectContaining({
      font_scale: 16,
    })
  )
})
