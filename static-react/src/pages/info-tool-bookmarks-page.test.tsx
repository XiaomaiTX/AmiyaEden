import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { FeedbackHost } from '@/feedback/feedback-host'
import { useFeedbackStore } from '@/feedback/store'
import { I18nProvider } from '@/i18n'
import { InfoToolBookmarksPage } from '@/pages/info-tool-bookmarks-page'
import { useSessionStore } from '@/stores'

vi.mock('@/api/tool-bookmark', () => ({
  createToolBookmark: vi.fn(),
  deleteToolBookmark: vi.fn(),
  fetchAdminToolBookmarks: vi.fn(),
  fetchVisibleToolBookmarks: vi.fn(),
  updateToolBookmark: vi.fn(),
}))

import {
  createToolBookmark,
  deleteToolBookmark,
  fetchAdminToolBookmarks,
  fetchVisibleToolBookmarks,
  updateToolBookmark,
} from '@/api/tool-bookmark'

const bookmark = {
  id: 1,
  name: 'EVE Tools',
  url: 'https://example.com',
  description: 'Useful tools',
  logo_url: '',
  logo_source: '',
  is_enabled: false,
  sort_order: 1,
  created_by: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function renderPage() {
  return render(
    <I18nProvider>
      <InfoToolBookmarksPage />
      <FeedbackHost />
    </I18nProvider>
  )
}

function setRoles(roles: string[]) {
  useSessionStore.getState().setSessionSnapshot({
    isLoggedIn: true,
    accessToken: 'token-123',
    characterId: 1001,
    characterName: 'Amiya',
    roles,
    corpCapabilities: ['menu.info'],
  })
}

describe('info tool bookmarks page', () => {
  beforeEach(() => {
    vi.mocked(fetchVisibleToolBookmarks).mockReset()
    vi.mocked(fetchAdminToolBookmarks).mockReset()
    vi.mocked(createToolBookmark).mockReset()
    vi.mocked(updateToolBookmark).mockReset()
    vi.mocked(deleteToolBookmark).mockReset()
    useFeedbackStore.setState({
      toasts: [],
      confirm: {
        open: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
      },
      confirmResolver: null,
    })
  })

  test('loads the visible bookmark list for regular users with safe external links', async () => {
    setRoles(['user'])
    vi.mocked(fetchVisibleToolBookmarks).mockResolvedValueOnce([{ ...bookmark, is_enabled: true }])

    renderPage()

    const link = await screen.findByRole('link', { name: 'EVE Tools' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    expect(fetchVisibleToolBookmarks).toHaveBeenCalledOnce()
    expect(fetchAdminToolBookmarks).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: '新增书签' })).not.toBeInTheDocument()
  })

  test('loads the management list and marks disabled bookmarks for administrators', async () => {
    setRoles(['admin'])
    vi.mocked(fetchAdminToolBookmarks).mockResolvedValueOnce([bookmark])

    renderPage()

    expect(await screen.findByText('已禁用')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '新增书签' })).toBeInTheDocument()
    expect(fetchAdminToolBookmarks).toHaveBeenCalledOnce()
    expect(fetchVisibleToolBookmarks).not.toHaveBeenCalled()
  })

  test('renders the bookmark editor as a single dialog surface', async () => {
    setRoles(['admin'])
    vi.mocked(fetchAdminToolBookmarks).mockResolvedValueOnce([bookmark])
    const user = userEvent.setup()

    renderPage()

    await screen.findByRole('link', { name: 'EVE Tools' })
    await user.click(screen.getByRole('button', { name: '编辑' }))

    expect(screen.getAllByRole('dialog')).toHaveLength(1)
    expect(document.querySelector('[data-slot="dialog-content"]')).toHaveClass(
      'w-full',
      'max-w-sm'
    )
    expect(screen.getByRole('dialog').querySelector('[data-slot="dialog-footer"]')).toBeNull()
    expect(document.querySelector('[data-slot="dialog-content"] .overflow-y-auto')).toBeNull()
  })

  test('shows the empty state', async () => {
    setRoles(['user'])
    vi.mocked(fetchVisibleToolBookmarks).mockResolvedValueOnce([])

    renderPage()

    expect(await screen.findByText('暂无工具书签')).toBeInTheDocument()
  })

  test('validates required name and URL before saving', async () => {
    setRoles(['admin'])
    vi.mocked(fetchAdminToolBookmarks).mockResolvedValueOnce([])
    const user = userEvent.setup()

    renderPage()
    await screen.findByText('暂无工具书签')
    await user.click(screen.getByRole('button', { name: '新增书签' }))
    await user.click(screen.getByRole('button', { name: '保存' }))

    expect(createToolBookmark).not.toHaveBeenCalled()
    expect(screen.getByText('名称和 URL 为必填项')).toBeInTheDocument()
  })

  test('reloads after creating, updating, and deleting bookmarks', async () => {
    setRoles(['super_admin'])
    vi.mocked(fetchAdminToolBookmarks).mockResolvedValue([bookmark])
    vi.mocked(createToolBookmark).mockResolvedValue(bookmark)
    vi.mocked(updateToolBookmark).mockResolvedValue(bookmark)
    vi.mocked(deleteToolBookmark).mockResolvedValue(null)
    const user = userEvent.setup()

    renderPage()
    await screen.findByRole('link', { name: 'EVE Tools' })

    await user.click(screen.getByRole('button', { name: '新增书签' }))
    const inputs = screen.getAllByRole('textbox')
    await user.type(inputs[0], 'New Tool')
    await user.type(inputs[1], 'https://new.example.com')
    await user.click(screen.getByRole('button', { name: '保存' }))
    await waitFor(() => expect(createToolBookmark).toHaveBeenCalledOnce())

    await user.click(screen.getByRole('button', { name: '编辑' }))
    await user.click(screen.getByRole('button', { name: '保存' }))
    await waitFor(() =>
      expect(updateToolBookmark).toHaveBeenCalledWith(
        1,
        expect.objectContaining({ name: 'EVE Tools' })
      )
    )

    await user.click(screen.getByRole('button', { name: '删除' }))
    await user.click(screen.getAllByRole('button', { name: '删除' }).at(-1)!)
    await waitFor(() => expect(deleteToolBookmark).toHaveBeenCalledWith(1))
    await waitFor(() => expect(fetchAdminToolBookmarks).toHaveBeenCalledTimes(4))
  })
})
