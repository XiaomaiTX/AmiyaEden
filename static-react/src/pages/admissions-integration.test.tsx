import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'
import { SystemUserPage } from './system-user-page'
import { SystemWalletPage } from './system-wallet-page'

const mocks = vi.hoisted(() => ({
  users: vi.fn(),
  definitions: vi.fn(),
  roles: vi.fn(),
  setRoles: vi.fn(),
  update: vi.fn(),
  wallets: vi.fn(),
  transactions: vi.fn(),
}))
vi.mock('@/api/system-manage', () => ({
  fetchGetUserList: mocks.users,
  fetchGetRoleDefinitions: mocks.definitions,
  fetchGetUserRoles: mocks.roles,
  fetchSetUserRoles: mocks.setRoles,
  fetchUpdateUser: mocks.update,
}))
vi.mock('@/api/sys-wallet', () => ({
  adminListWallets: mocks.wallets,
  adminListTransactions: mocks.transactions,
}))
vi.mock('@/hooks/use-corp-capability', () => ({
  useCorpCapability: () => ({ hasCapability: () => true }),
}))

beforeEach(() => {
  vi.clearAllMocks()
  usePreferenceStore.setState({ locale: 'en-US' })
  useSessionStore.getState().setSessionSnapshot({
    isLoggedIn: true,
    accessToken: 'token',
    roles: ['admin'],
    characterId: 1001,
    characterName: 'Admin',
    corpCapabilities: [],
  })
  mocks.wallets.mockResolvedValue({ list: [], total: 0, page: 1, pageSize: 20 })
})

test('derived recruiter stays read-only while ordinary roles remain editable', async () => {
  const definitions = ['user', 'recruiter', 'srp'].map((code, sort) => ({
    code,
    name: code,
    description: '',
    sort,
  }))
  mocks.definitions.mockResolvedValue(definitions)
  mocks.roles.mockResolvedValue(definitions.slice(0, 2))
  mocks.users.mockResolvedValue({
    list: [
      {
        id: 8,
        nickname: 'Pilot',
        roles: ['user', 'recruiter'],
        qq: '',
        discord_id: 'discord',
        status: 1,
        characters: [],
        created_at: '',
        updated_at: '',
      },
    ],
    total: 1,
    page: 1,
    pageSize: 20,
  })
  mocks.update.mockResolvedValue(undefined)
  mocks.setRoles.mockResolvedValue(undefined)
  render(
    <I18nProvider>
      <SystemUserPage />
    </I18nProvider>
  )
  fireEvent.click(await screen.findByRole('button', { name: 'Edit' }))
  const recruiter = await screen.findByRole('checkbox', { name: 'Recruiter' })
  await waitFor(() => expect(recruiter).toBeChecked())
  expect(recruiter).toBeDisabled()
  const srp = screen.getByRole('checkbox', { name: 'SRP' })
  expect(srp).not.toBeDisabled()
  fireEvent.click(srp)
  fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
  await waitFor(() => expect(mocks.setRoles).toHaveBeenCalledWith(8, ['user', 'recruiter', 'srp']))
})

test('recruiter salary has a readable label and can filter transaction queries', async () => {
  mocks.transactions.mockResolvedValue({
    list: [
      {
        id: 11,
        user_id: 8,
        amount: 10,
        balance_after: 10,
        reason: 'Corporation admission #17',
        ref_type: 'recruitment_invitation_salary',
        ref_id: 'admission:17',
        operator_id: 8,
        created_at: '2026-11-11T00:00:00Z',
      },
    ],
    total: 1,
    page: 1,
    pageSize: 20,
  })
  render(
    <I18nProvider>
      <SystemWalletPage />
    </I18nProvider>
  )
  fireEvent.click(screen.getByRole('button', { name: 'Transactions' }))
  expect(await screen.findByText('Corporation application #17')).toBeInTheDocument()
  expect(screen.getAllByText('Recruiter salary').some((node) => node.tagName === 'SPAN')).toBe(true)
  fireEvent.click(screen.getByRole('button', { name: /Transaction Type/ }))
  const list = await screen.findByRole('listbox')
  fireEvent.click(within(list).getByRole('option', { name: 'Recruiter salary' }))
  fireEvent.click(screen.getByRole('button', { name: 'Search' }))
  await waitFor(() =>
    expect(mocks.transactions).toHaveBeenLastCalledWith(
      expect.objectContaining({ ref_type: 'recruitment_invitation_salary' })
    )
  )
})
