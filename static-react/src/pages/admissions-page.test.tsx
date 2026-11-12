import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { AdmissionsPage } from './admissions-page'
import type { AdmissionContext, AdmissionDetail } from '@/types/api/admissions'
import { fetchOperationSalaryRates } from '@/api/salary'
vi.mock('@/api/salary', () => ({ fetchOperationSalaryRates: vi.fn() }))

const mocks = vi.hoisted(() => ({
  context: null as AdmissionContext | null,
  reload: vi.fn(),
  save: vi.fn(),
  detail: vi.fn(),
  list: vi.fn(),
  process: vi.fn(),
  evidence: vi.fn(),
}))
vi.mock('@/hooks/use-admission-context', () => ({
  useAdmissionContext: () => ({
    context: mocks.context,
    loading: false,
    error: false,
    reload: mocks.reload,
  }),
}))
vi.mock('@/api/admissions', () => ({
  saveAdmission: mocks.save,
  getAdmission: mocks.detail,
  listAdmissions: mocks.list,
  processAdmission: mocks.process,
  getAdmissionEvidence: mocks.evidence,
}))
vi.mock('@/lib/admissions', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/admissions')>()),
  compressAdmissionImage: async (file: File) => file,
}))

function context(member = false): AdmissionContext {
  return {
    is_member: member,
    can_create_new_member: !member,
    can_create_alt: member,
    qq: '',
    discord_id: 'discord-only',
    reviewable_corporation_ids: [],
    targets: [
      { corporation_id: 98185110, name: 'Fuxi Legion' },
      { corporation_id: 98824507, name: 'FMA' },
    ],
    characters: [
      { character_id: 101, character_name: 'Pilot A', corporation_id: 1001 },
      { character_id: 102, character_name: 'Pilot B', corporation_id: 1001 },
    ],
  }
}
function detail(status: 'pending' | 'rejected' | 'invited' = 'pending'): AdmissionDetail {
  return {
    application: {
      id: 17,
      user_id: 1,
      kind: 'new_member',
      target_corporation_id: 98185110,
      status,
      version: 3,
      applicant_name: 'Applicant',
      primary_character_id: 101,
      qq: '',
      discord_id: 'discord-only',
      submitted_at: '2026-11-10T00:00:00Z',
      handled_by: null,
      handler_name: '',
      closed_at: null,
      evidence_expires_at: null,
      rejection_reason: status === 'rejected' ? 'Bind missing pilot' : '',
      characters: [{ character_id: 101, character_name: 'Pilot A' }],
    },
    evidence: [],
    events: [
      {
        id: 1,
        application_id: 17,
        version: 2,
        actor_id: 2,
        actor_name: 'Recruiter',
        action: 'reject',
        from_status: 'pending',
        to_status: 'rejected',
        reason: 'Bind missing pilot',
        snapshot: '{}',
        created_at: '2026-11-10T00:00:00Z',
      },
    ],
    salary: null,
    can_review: false,
    is_owner: true,
  }
}
function open(path: string) {
  return render(
    <I18nProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/admissions/applications/:id/edit" element={<AdmissionsPage />} />
          <Route path="/admissions/applications/:id" element={<AdmissionsPage />} />
          <Route path="/admissions/*" element={<AdmissionsPage />} />
        </Routes>
      </MemoryRouter>
    </I18nProvider>
  )
}
beforeEach(() => {
  vi.clearAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
  mocks.context = context()
  vi.mocked(fetchOperationSalaryRates).mockResolvedValue({
    recruitment: 4.25,
    welfare_delivery: 18,
    shop_delivery: 6.5,
    srp_payout: 10,
  })
  mocks.list.mockResolvedValue({ list: [], total: 0, page: 1, pageSize: 20 })
  mocks.detail.mockResolvedValue(detail())
  mocks.evidence.mockResolvedValue(new Blob(['jpeg'], { type: 'image/jpeg' }))
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:admissions-test')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
})
afterEach(() => vi.restoreAllMocks())

test('unknown eligibility never opens a creation form or duplicates sidebar navigation', () => {
  mocks.context = null
  open('/admissions/new')
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
})
test('Discord-only multi-character application has exactly two image slots and preserves input on conflict', async () => {
  mocks.save.mockRejectedValue(new Error('admissions.errors.conflict'))
  open('/admissions/new')
  expect(screen.queryByText('联系资料')).not.toBeInTheDocument()
  expect(screen.queryByText('Discord: discord-only')).not.toBeInTheDocument()
  expect(screen.getByRole('combobox')).toHaveValue('98824507')
  expect(
    screen.getByRole('option', { name: '伏羲新手团（Fuxi Military Academy）' })
  ).toBeInTheDocument()
  expect(
    screen.getByRole('option', {
      name: '伏羲主团 （主号需7百万以上技能。已在伏羲的人，小号无要求）',
    })
  ).toBeInTheDocument()
  screen
    .getAllByRole('link', { name: '联盟SeAT' })
    .forEach((link) => expect(link).toHaveAttribute('href', 'https://seat.winterco.org'))
  expect(document.querySelectorAll('input[type=file]')).toHaveLength(2)
  document
    .querySelectorAll('input[type=file]')
    .forEach((input) => expect(input).toHaveAttribute('aria-required', 'true'))
  const images = screen.getAllByRole('img', { hidden: true })
  expect(images.map((image) => image.getAttribute('src'))).toEqual([
    '/admissions/qq-binding.png',
    '/admissions/seat-completed.png',
  ])
  screen.getAllByRole('checkbox').forEach((checkbox) => fireEvent.click(checkbox))
  fireEvent.click(screen.getByRole('button', { name: '提交申请' }))
  await screen.findByRole('alert')
  expect(mocks.save).toHaveBeenCalledWith(
    {
      kind: 'new_member',
      target_corporation_id: 98824507,
      character_ids: [101, 102],
      expected_version: 0,
    },
    {},
    undefined,
    false
  )
  expect(
    screen.getAllByRole('checkbox').every((input) => (input as HTMLInputElement).checked)
  ).toBe(true)
  expect(screen.getByRole('combobox')).toHaveValue('98824507')
})
test('member alt form needs no evidence and shows SeAT binding reminder', () => {
  mocks.context = context(true)
  open('/admissions/alt')
  expect(document.querySelectorAll('input[type=file]')).toHaveLength(0)
  expect(screen.getByText('请先把全部小号绑定到联盟 SeAT，再提交小号加团申请。')).toHaveClass(
    'text-red-600'
  )
})
test('rejected application resubmits the original ID and current version, preserving rejection history', async () => {
  mocks.detail.mockResolvedValue(detail('rejected'))
  mocks.save.mockRejectedValue(new Error('admissions.errors.evidence'))
  open('/admissions/applications/17/edit')
  fireEvent.click(await screen.findByRole('button', { name: '补正并重新提交' }))
  await waitFor(() =>
    expect(mocks.save).toHaveBeenCalledWith(
      expect.objectContaining({ expected_version: 3 }),
      {},
      17,
      true
    )
  )
  expect(screen.getAllByText(/Bind missing pilot/).length).toBeGreaterThan(0)
  expect(screen.getByRole('alert')).toHaveTextContent(
    '请提供有效的 JPEG/PNG 截图；压缩后每张不得超过 2 MiB。'
  )
})
test('review has only Invited and Reject; wallet failure preserves pending and retry uses the same version', async () => {
  const pending = detail()
  pending.can_review = true
  pending.is_owner = false
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  mocks.detail.mockResolvedValue(pending)
  mocks.process
    .mockRejectedValueOnce(new Error('admissions.errors.walletDisabled'))
    .mockResolvedValueOnce({ ...pending.application, status: 'invited' })
  vi.spyOn(window, 'confirm').mockReturnValue(true)
  open('/admissions/applications/17')
  const invited = await screen.findByRole('button', { name: '已邀请' })
  expect(screen.getByRole('button', { name: '拒绝' })).toBeDisabled()
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  fireEvent.click(invited)
  await waitFor(() =>
    expect(window.confirm).toHaveBeenCalledWith(
      expect.stringContaining('确认本单全部人物都已在游戏内邀请？')
    )
  )
  expect(await screen.findByRole('alert')).toHaveTextContent('请联系超管调整配置后重试')
  fireEvent.click(invited)
  await waitFor(() => expect(mocks.process).toHaveBeenCalledTimes(2))
  expect(mocks.process).toHaveBeenLastCalledWith(17, 3, 'invited', '')
})
test('English management filters include kind and only reviewable corporations', async () => {
  usePreferenceStore.setState({ locale: 'en-US' })
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  open('/admissions/review')
  fireEvent.change(await screen.findByRole('combobox', { name: 'Application type' }), {
    target: { value: 'alt' },
  })
  await waitFor(() => expect(mocks.list).toHaveBeenLastCalledWith(true, 1, 'pending', '', 'alt'))
  expect(
    screen.queryByRole('option', { name: 'Fuxi Newcomer Corporation' })
  ).not.toBeInTheDocument()
})

test('processed rows identify application kind and processing result without opening details', async () => {
  usePreferenceStore.setState({ locale: 'en-US' })
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  const app = {
    ...detail('rejected').application,
    kind: 'alt' as const,
    handler_name: 'Reviewer Seven',
    closed_at: '2026-11-11T00:00:00Z',
  }
  mocks.list.mockResolvedValue({ list: [app], total: 1, page: 1, pageSize: 20 })
  open('/admissions/review')
  fireEvent.click(await screen.findByRole('tab', { name: 'Processed' }))
  expect(await screen.findByText('Handled by: Reviewer Seven')).toBeInTheDocument()
  expect(screen.getByText('Bind missing pilot')).toBeInTheDocument()
  expect(screen.getByRole('cell', { name: 'Apply for alts' })).toBeInTheDocument()
  expect(document.querySelector('time')).toHaveAttribute('dateTime', app.closed_at)
  await waitFor(() => expect(mocks.list).toHaveBeenLastCalledWith(true, 1, 'processed', '', ''))
})

test('expired rejected evidence uses the resubmit endpoint from the primary submit action', async () => {
  const rejected = detail('rejected')
  rejected.application.evidence_expires_at = '2025-01-01T00:00:00Z'
  mocks.detail.mockResolvedValue(rejected)
  mocks.save.mockResolvedValue(rejected.application)
  open('/admissions/applications/17/edit')
  const button = await screen.findByRole('button', { name: '补正并重新提交' })
  expect(screen.queryByRole('button', { name: '保存修改' })).not.toBeInTheDocument()
  fireEvent.submit(button.closest('form')!)
  await waitFor(() =>
    expect(mocks.save).toHaveBeenCalledWith(
      expect.objectContaining({ expected_version: 3 }),
      {},
      17,
      true
    )
  )
})

test('owner details are read-only and do not show salary', async () => {
  const value = detail()
  value.salary = {
    application_id: 17,
    user_id: 2,
    amount: 10,
    wallet_transaction_id: 99,
    created_at: '2026-11-11T00:00:00Z',
  }
  mocks.detail.mockResolvedValue(value)
  open('/admissions/applications/17')
  await screen.findByText('Applicant', { exact: false })
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '保存修改' })).not.toBeInTheDocument()
  expect(screen.queryByText(/本单工资/)).not.toBeInTheDocument()
})

test.each(['new_member', 'alt'] as const)(
  'review guidance is scoped to %s applications',
  async (kind) => {
    const value = detail()
    value.application.kind = kind
    value.can_review = true
    value.is_owner = false
    mocks.detail.mockResolvedValue(value)
    open('/admissions/applications/17')
    await screen.findByRole('button', { name: '已邀请' })
    expect(
      screen.queryByText('请先把全部小号绑定到联盟 SeAT，再提交小号加团申请。')
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '拒绝' })).toBeInTheDocument()
    const guidance = screen.queryByText(/请核对登录器截图与联盟 SeAT 截图中的人物数量/)
    if (kind === 'new_member') expect(guidance).toBeInTheDocument()
    else expect(guidance).not.toBeInTheDocument()
  }
)

test.each([
  { status: 'invited' as const, owner: true },
  { status: 'pending' as const, owner: false },
])('edit route refuses $status / owner=$owner', async ({ status, owner }) => {
  mocks.detail.mockResolvedValue({ ...detail(status), is_owner: owner })
  open('/admissions/applications/17/edit')
  await screen.findByRole('alert')
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
})

test('history offers separate editing only for unfinished applications', async () => {
  mocks.list.mockResolvedValue({
    list: [detail().application, { ...detail('invited').application, id: 18 }],
    total: 2,
    page: 1,
    pageSize: 20,
  })
  open('/admissions/history')
  const edit = await screen.findByRole('link', { name: '修改申请' })
  expect(edit).toHaveAttribute('href', '/admissions/applications/17/edit')
  expect(screen.getAllByRole('link', { name: '修改申请' })).toHaveLength(1)
})

test.each([false, true])(
  'recruitment salary confirmation requires consent: %s',
  async (consent) => {
    mocks.detail.mockResolvedValue({ ...detail(), can_review: true, is_owner: false })
    mocks.process.mockResolvedValue({})
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open('/admissions/applications/17')
    fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(expect.stringContaining('4.25 伏羲币工资'))
    )
    if (consent)
      await waitFor(() => expect(mocks.process).toHaveBeenCalledWith(17, 3, 'invited', ''))
    else expect(mocks.process).not.toHaveBeenCalled()
  }
)

test('recruitment cannot submit when current salary cannot be read', async () => {
  mocks.detail.mockResolvedValue({ ...detail(), can_review: true })
  vi.mocked(fetchOperationSalaryRates).mockRejectedValueOnce(new Error('offline'))
  const confirm = vi.spyOn(window, 'confirm')
  open('/admissions/applications/17')
  fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
  await screen.findByRole('alert')
  expect(confirm).not.toHaveBeenCalled()
  expect(mocks.process).not.toHaveBeenCalled()
})
