import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { AnnouncementsPage } from './announcements-page'
import { useSessionStore } from '@/stores/session-store'
import {
  createAnnouncement,
  listAnnouncements,
  getAnnouncement,
  readAnnouncement,
  updateAnnouncement,
} from '@/api/generated/client'

vi.mock('@/api/generated/client', () => ({
  listAnnouncements: vi.fn(),
  getAnnouncement: vi.fn(),
  readAnnouncement: vi.fn(),
  createAnnouncement: vi.fn(),
  updateAnnouncement: vi.fn(),
}))
vi.mock('@/i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
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
    <textarea aria-label={ariaLabel} value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}))

const success = <T,>(data: T) => ({ code: 200 as const, msg: 'ok', data })

const announcement = {
  id: 1,
  title: 'Fleet tonight',
  body_html: '<p>Form up at 19:00</p>',
  author_id: 1,
  author_nickname: '舰队指挥',
  author_character_id: 90000001,
  author_character_name: 'Fleet Commander',
  hidden: false,
  unread: true,
  version: 2,
  revision: 3,
  created_at: '2026-11-23T00:00:00Z',
  updated_at: '2026-11-23T00:00:00Z',
}
beforeEach(() => {
  vi.clearAllMocks()
  useSessionStore.getState().setSessionSnapshot({ roles: ['user'], characterId: null })
  vi.mocked(listAnnouncements).mockResolvedValue(success({ list: [announcement], total: 1 }))
  vi.mocked(getAnnouncement).mockResolvedValue(success(announcement))
  vi.mocked(readAnnouncement).mockResolvedValue(success(null))
})
test('opens the newest announcement on entry and acknowledges the displayed version', async () => {
  const older = { ...announcement, id: 2, title: 'Old notice', unread: false }
  vi.mocked(listAnnouncements).mockResolvedValue(success({ list: [announcement, older], total: 2 }))
  render(<AnnouncementsPage />)
  await screen.findByText('Form up at 19:00')
  expect(getAnnouncement).toHaveBeenCalledTimes(1)
  expect(getAnnouncement).toHaveBeenCalledWith(1)
  await waitFor(() => expect(readAnnouncement).toHaveBeenCalledWith(1, { version: 2 }))
  await waitFor(() => expect(screen.queryByText('announcement.unread')).not.toBeInTheDocument())
  expect(listAnnouncements).toHaveBeenCalledTimes(1)
})

test('shows the publisher nickname and primary character in the list and article', async () => {
  render(<AnnouncementsPage />)
  await screen.findByText('Form up at 19:00')
  expect(screen.getAllByText('舰队指挥')).toHaveLength(2)
  expect(screen.getAllByText('Fleet Commander')).toHaveLength(2)
  expect(screen.getByText('舰队指挥', { selector: 'strong' }).parentElement).toHaveTextContent(
    'Fleet Commander'
  )
  expect(screen.getByRole('button', { name: /Fleet tonight/ })).toHaveAttribute(
    'aria-current',
    'true'
  )
})

test('keeps announcements readable when the publisher profile is unavailable', async () => {
  const orphaned = {
    ...announcement,
    author_nickname: '',
    author_character_id: 0,
    author_character_name: '',
  }
  vi.mocked(listAnnouncements).mockResolvedValue(success({ list: [orphaned], total: 1 }))
  vi.mocked(getAnnouncement).mockResolvedValue(success(orphaned))
  render(<AnnouncementsPage />)
  await screen.findByText('Form up at 19:00')
  expect(screen.getAllByText('announcement.unknownAuthor')).toHaveLength(2)
  expect(screen.queryByRole('img')).not.toBeInTheDocument()
})
test('allows paging after ten announcements and disables next on the final page', async () => {
  vi.mocked(listAnnouncements)
    .mockResolvedValueOnce(success({ list: [announcement], total: 11 }))
    .mockResolvedValueOnce(
      success({ list: [{ ...announcement, id: 11, title: 'Last notice' }], total: 11 })
    )
  render(<AnnouncementsPage />)
  await screen.findByText('Form up at 19:00')
  const next = screen.getByRole('button', { name: 'announcement.next' })
  expect(next).toBeEnabled()
  fireEvent.click(next)
  await screen.findByRole('button', { name: /Last notice/ })
  expect(listAnnouncements).toHaveBeenLastCalledWith({ page: 2 })
  expect(next).toBeDisabled()
  expect(screen.getByRole('button', { name: 'announcement.previous' })).toBeEnabled()
})

test('reopening an already-read announcement skips the read request', async () => {
  vi.mocked(listAnnouncements).mockResolvedValue(
    success({
      list: [{ ...announcement, unread: false }],
      total: 1,
    })
  )
  render(<AnnouncementsPage />)
  await screen.findByText('Form up at 19:00')
  fireEvent.click(screen.getByRole('button', { name: /Fleet tonight/ }))
  await waitFor(() => expect(getAnnouncement).toHaveBeenCalledTimes(2))
  expect(screen.getByRole('button', { name: /Fleet tonight/ })).toBeEnabled()
  expect(readAnnouncement).not.toHaveBeenCalled()
})
test('a failed detail request keeps the announcement unread', async () => {
  vi.mocked(getAnnouncement).mockRejectedValueOnce(new Error('offline'))
  render(<AnnouncementsPage />)
  await screen.findByRole('alert')
  expect(screen.getByText('announcement.unread')).toBeInTheDocument()
  expect(readAnnouncement).not.toHaveBeenCalled()
})
test('a manager can publish a hidden announcement', async () => {
  useSessionStore.getState().setSessionSnapshot({ roles: ['senior_fc'], characterId: null })
  vi.mocked(createAnnouncement).mockResolvedValue(success({ ...announcement, hidden: true }))
  render(<AnnouncementsPage />)
  fireEvent.click(screen.getByRole('button', { name: 'announcement.publish' }))
  fireEvent.change(await screen.findByRole('textbox', { name: 'announcement.subject' }), {
    target: { value: 'Fleet tonight' },
  })
  fireEvent.change(screen.getByRole('textbox', { name: 'announcement.body' }), {
    target: { value: '<p>Form up at 19:00</p>' },
  })
  fireEvent.click(screen.getByRole('checkbox', { name: 'announcement.hidden' }))
  fireEvent.submit(screen.getByRole('button', { name: 'announcement.save' }).closest('form')!)
  await waitFor(() =>
    expect(createAnnouncement).toHaveBeenCalledWith({
      title: 'Fleet tonight',
      body_html: '<p>Form up at 19:00</p>',
      hidden: true,
      remind: false,
      revision: 0,
    })
  )
  await screen.findByRole('button', { name: 'announcement.edit' })
  // Starting to publish before the list arrives suppresses the automatic open.
  expect(getAnnouncement).not.toHaveBeenCalled()
})

test('editing an existing announcement sends its revision to the update operation', async () => {
  useSessionStore.getState().setSessionSnapshot({ roles: ['senior_fc'], characterId: null })
  vi.mocked(updateAnnouncement).mockResolvedValue(success({ ...announcement, title: 'Updated' }))
  render(<AnnouncementsPage />)
  fireEvent.click(await screen.findByRole('button', { name: 'announcement.edit' }))
  fireEvent.change(screen.getByRole('textbox', { name: 'announcement.subject' }), {
    target: { value: 'Updated' },
  })
  fireEvent.submit(screen.getByRole('button', { name: 'announcement.save' }).closest('form')!)
  await waitFor(() =>
    expect(updateAnnouncement).toHaveBeenCalledWith(
      announcement.id,
      expect.objectContaining({ title: 'Updated', revision: announcement.revision })
    )
  )
  expect(createAnnouncement).not.toHaveBeenCalled()
})
