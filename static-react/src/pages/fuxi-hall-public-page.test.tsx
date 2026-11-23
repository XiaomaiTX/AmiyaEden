import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FuxiHallPublicPage } from '@/pages/fuxi-hall-public-page'
import { I18nProvider } from '@/i18n'
import { fetchFuxiHall, updateOwnFuxiHallCard } from '@/api/fuxi-hall'
import type { FuxiHallCard } from '@/types/api/fuxi-hall'

vi.mock('@/api/fuxi-hall', () => ({ fetchFuxiHall: vi.fn(), updateOwnFuxiHallCard: vi.fn() }))
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

const card: FuxiHallCard = {
  id: 1,
  page_key: 'hall-of-fame',
  nickname: 'Alpha',
  main_character_id: 10,
  main_character_name: 'Main',
  title_tags: ['Director'],
  description_html: '<p>Bio</p>',
  accent_color: '#3b82f6',
  avatar_shape: 'circle',
  font_scale: 14,
  visible: true,
  floor_key: 'top',
  is_current_management: true,
  tenure_months: 26,
  can_edit: true,
  sort_order: 1,
  created_at: '',
  updated_at: '',
}
beforeEach(() => {
  vi.mocked(fetchFuxiHall).mockResolvedValue({
    page: {
      id: 1,
      page_key: 'hall-of-fame',
      title: '伏羲名人堂',
      subtitle: '',
      description_html: '',
      floors: [
        { key: 'top', name: '', capacity: 8 },
        { key: 'first', name: '一楼', capacity: 33 },
      ],
    },
    cards: [
      card,
      {
        ...card,
        id: 2,
        nickname: 'Beta',
        floor_key: 'first',
        can_edit: false,
        is_current_management: false,
        tenure_months: null,
      },
    ],
  })
})

test('groups cards by configured floors and displays management and cumulative tenure', async () => {
  render(
    <I18nProvider>
      <FuxiHallPublicPage />
    </I18nProvider>
  )
  const top = await screen.findByRole('region', { name: '顶楼' })
  expect(within(top).getByText('Alpha')).toBeInTheDocument()
  expect(within(top).queryByText('Beta')).not.toBeInTheDocument()
  expect(within(top).getByText('在职管理')).toBeInTheDocument()
  expect(screen.getByText(/2年2个月/)).toBeInTheDocument()
  expect(screen.getByText(/暂无记录/)).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: '编辑我的昵称和简介' })).toHaveLength(1)
})

test('opens a configured floor from the banner without changing routes', async () => {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <FuxiHallPublicPage />
    </I18nProvider>
  )
  const floor = await screen.findByRole('region', { name: '一楼' })
  const scroll = vi.fn()
  floor.scrollIntoView = scroll
  await user.click(
    within(screen.getByRole('navigation', { name: '楼层导览' })).getByRole('button', {
      name: '一楼',
    })
  )
  expect(floor).toHaveFocus()
  expect(scroll).toHaveBeenCalledWith({ block: 'start' })
})

test('self edit submits only nickname and bio and retains draft on failure', async () => {
  const user = userEvent.setup()
  vi.mocked(updateOwnFuxiHallCard)
    .mockRejectedValueOnce(new Error('fuxiHall.errors.denied'))
    .mockResolvedValueOnce({ ...card, nickname: 'New name', description_html: '<p>New bio</p>' })
  render(
    <I18nProvider>
      <FuxiHallPublicPage />
    </I18nProvider>
  )
  await user.click(await screen.findByRole('button', { name: '编辑我的昵称和简介' }))
  const dialog = screen.getByRole('dialog')
  await user.clear(within(dialog).getByLabelText('昵称'))
  await user.type(within(dialog).getByLabelText('昵称'), 'New name')
  await user.clear(within(dialog).getByRole('textbox', { name: '成员简介（富文本）' }))
  await user.type(
    within(dialog).getByRole('textbox', { name: '成员简介（富文本）' }),
    '<p>New bio</p>'
  )
  await user.click(within(dialog).getByRole('button', { name: '保存' }))
  expect(await within(dialog).findByRole('alert')).toHaveTextContent('只能编辑')
  expect(within(dialog).getByLabelText('昵称')).toHaveValue('New name')
  expect(updateOwnFuxiHallCard).toHaveBeenCalledWith(1, {
    nickname: 'New name',
    description_html: '<p>New bio</p>',
  })
  await user.click(within(dialog).getByRole('button', { name: '保存' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  expect(screen.getByText('New name')).toBeInTheDocument()
})

test('owner cancels the inline popup without saving or navigating', async () => {
  vi.mocked(updateOwnFuxiHallCard).mockClear()
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <FuxiHallPublicPage />
    </I18nProvider>
  )
  await user.click(await screen.findByRole('button', { name: '编辑我的昵称和简介' }))
  const dialog = screen.getByRole('dialog')
  expect(within(dialog).getAllByRole('textbox')).toHaveLength(2)
  await user.clear(within(dialog).getByLabelText('昵称'))
  await user.type(within(dialog).getByLabelText('昵称'), 'Unsaved')
  await user.click(within(dialog).getByRole('button', { name: '取消' }))
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  expect(screen.getByText('Alpha')).toBeInTheDocument()
  expect(screen.queryByText('Unsaved')).not.toBeInTheDocument()
  expect(updateOwnFuxiHallCard).not.toHaveBeenCalled()
})
