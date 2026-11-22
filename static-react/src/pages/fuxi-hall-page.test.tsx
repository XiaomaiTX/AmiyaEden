import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FuxiHallPage } from '@/pages/fuxi-hall-page'
import { I18nProvider } from '@/i18n'
import { useSessionStore } from '@/stores'

const { manageMounted } = vi.hoisted(() => ({ manageMounted: vi.fn() }))
vi.mock('@/pages/fuxi-hall-public-page', () => ({
  FuxiHallPublicPage: () => <p>Hall display</p>,
}))
vi.mock('@/pages/fuxi-hall-manage-page', () => ({
  FuxiHallManagePage: () => {
    manageMounted()
    return <p>Hall management</p>
  },
}))

beforeEach(() => {
  manageMounted.mockClear()
  useSessionStore.getState().setSessionSnapshot({ roles: ['user'] })
})

function renderHall() {
  return render(
    <I18nProvider>
      <FuxiHallPage />
    </I18nProvider>
  )
}

test('members enter the display directly without management tabs or mounting management', () => {
  renderHall()
  expect(screen.getByText('Hall display')).toBeInTheDocument()
  expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  expect(manageMounted).not.toHaveBeenCalled()
})

test.each(['admin', 'super_admin'])('%s can switch to management within the page', async (role) => {
  const user = userEvent.setup()
  useSessionStore.getState().setSessionSnapshot({ roles: [role] })
  renderHall()
  expect(screen.getByRole('tab', { name: '伏羲名人堂' })).toHaveAttribute('aria-selected', 'true')
  expect(manageMounted).not.toHaveBeenCalled()
  await user.click(screen.getByRole('tab', { name: '伏羲大厅管理' }))
  expect(await screen.findByText('Hall management')).toBeInTheDocument()
  expect(screen.queryByText('Hall display')).not.toBeInTheDocument()
  await user.click(screen.getByRole('tab', { name: '伏羲名人堂' }))
  expect(screen.getByText('Hall display')).toBeInTheDocument()
  expect(screen.queryByText('Hall management')).not.toBeInTheDocument()
})

test('losing the admin role unmounts management and returns to display', async () => {
  const user = userEvent.setup()
  useSessionStore.getState().setSessionSnapshot({ roles: ['admin'] })
  renderHall()
  await user.click(screen.getByRole('tab', { name: '伏羲大厅管理' }))
  expect(await screen.findByText('Hall management')).toBeInTheDocument()
  act(() => useSessionStore.getState().setSessionSnapshot({ roles: ['user'] }))
  expect(screen.getByText('Hall display')).toBeInTheDocument()
  expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  expect(screen.queryByText('Hall management')).not.toBeInTheDocument()
})
