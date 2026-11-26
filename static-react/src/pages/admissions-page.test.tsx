import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { AdmissionsPage } from './admissions-page'
import type { AdmissionContext, AdmissionDetail } from '@/api/generated/client'
import { operationRates } from '@/api/generated/client'
vi.mock('@/api/generated/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/generated/client')>()),
  operationRates: vi.fn(),
  getAdmission: mocks.detail,
  listAdmissionReviews: mocks.reviews,
  listMyAdmissions: mocks.history,
  inviteAdmission: mocks.invite,
  rejectAdmission: mocks.reject,
}))

const mocks = vi.hoisted(() => ({
  context: null as AdmissionContext | null,
  reload: vi.fn(),
  save: vi.fn(),
  detail: vi.fn(),
  reviews: vi.fn(),
  history: vi.fn(),
  invite: vi.fn(),
  reject: vi.fn(),
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
  getAdmissionEvidence: mocks.evidence,
}))
vi.mock('@/lib/admissions', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/admissions')>()),
  compressAdmissionImage: async (file: File) => file,
}))

function success<T>(data: T) {
  return { code: 200, msg: 'ok', data }
}

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
      created_at: '2026-11-10T00:00:00Z',
      updated_at: '2026-11-10T00:00:00Z',
      note: '',
      handled_by: null,
      handler_name: '',
      closed_at: null,
      evidence_expires_at: null,
      rejection_reason: status === 'rejected' ? 'Bind missing pilot' : '',
      characters: [{ application_id: 17, character_id: 101, character_name: 'Pilot A' }],
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
    blacklist_hits: [],
  }
}
const blacklistHits: AdmissionDetail['blacklist_hits'] = [
  { kind: 'character', value: '9555', display: 'Bad Pilot', reason: 'Scam\nRepeat offender' },
  { kind: 'qq', value: '11111111', display: '11111111', reason: '' },
]
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
  vi.mocked(operationRates).mockResolvedValue({
    code: 200,
    msg: '',
    data: {
      recruitment: 4.25,
      welfare_delivery: 18,
      shop_delivery: 6.5,
      srp_payout: 10,
    },
  })
  const emptyList = success({ list: [], total: 0, page: 1, pageSize: 20 })
  mocks.reviews.mockResolvedValue(emptyList)
  mocks.history.mockResolvedValue(emptyList)
  mocks.invite.mockResolvedValue(success(detail().application))
  mocks.reject.mockResolvedValue(success(detail('rejected').application))
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: detail() })
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
test('character tiles show portraits without IDs and support whole-tile multi-selection', async () => {
  const user = userEvent.setup()
  mocks.context!.characters.push({
    character_id: 103,
    character_name: 'Already joined',
    corporation_id: 98824507,
  })
  open('/admissions/new')
  const group = screen.getByRole('group', { name: '本单加入人物' })
  const pilotA = within(group).getByRole('checkbox', { name: 'Pilot A' })
  const pilotB = within(group).getByRole('checkbox', { name: 'Pilot B' })
  expect(within(group).queryByText(/101|102|103/)).not.toBeInTheDocument()
  expect(group.querySelectorAll('img')).toHaveLength(3)
  expect(group.querySelector('img')).toHaveAttribute(
    'src',
    expect.stringContaining('/characters/101/portrait?size=64')
  )
  await user.click(within(group).getByText('Pilot A'))
  await user.click(within(group).getByText('Pilot B'))
  expect(pilotA).toBeChecked()
  expect(pilotB).toBeChecked()
  await user.click(within(group).getByText('Pilot A'))
  expect(pilotA).not.toBeChecked()
  expect(pilotB).toBeChecked()
  const joined = within(group).getByRole('checkbox', { name: 'Already joined' })
  await user.click(within(group).getByText('Already joined'))
  expect(joined).toBeDisabled()
  expect(joined).not.toBeChecked()
  await user.selectOptions(screen.getByRole('combobox'), '98185110')
  expect(pilotB).not.toBeChecked()
  expect(joined).toBeEnabled()
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
    .forEach((link) =>
      expect(link).toHaveAttribute('href', 'https://seat.winterco.org/character/list')
    )
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
test('member can submit one alt with a note and open alliance SeAT', async () => {
  mocks.context = context(true)
  mocks.save.mockRejectedValue(new Error('admissions.errors.conflict'))
  usePreferenceStore.setState({ locale: 'en-US' })
  open('/admissions/alt')
  expect(document.querySelectorAll('input[type=file]')).toHaveLength(0)
  expect(screen.getByRole('link', { name: 'alliance SeAT' })).toHaveAttribute(
    'href',
    'https://seat.winterco.org/character/list'
  )
  const note = screen.getByRole('textbox', { name: 'Application note (optional)' })
  expect(note).toHaveAttribute('maxlength', '2000')
  fireEvent.change(note, {
    target: { value: 'Other alts: Pilot B, Pilot C\nAll bound to alliance SeAT.' },
  })
  fireEvent.click(screen.getAllByRole('checkbox')[0])
  fireEvent.click(screen.getByRole('button', { name: 'Submit application' }))
  await screen.findByRole('alert')
  expect(mocks.save).toHaveBeenCalledWith(
    expect.objectContaining({
      kind: 'alt',
      character_ids: [101],
      note: 'Other alts: Pilot B, Pilot C\nAll bound to alliance SeAT.',
    }),
    {},
    undefined,
    false
  )
  expect(note).toHaveValue('Other alts: Pilot B, Pilot C\nAll bound to alliance SeAT.')
})
test('alt note is shown in detail and restored when editing', async () => {
  mocks.context = context(true)
  const value = detail('rejected')
  value.application.kind = 'alt'
  value.application.note = 'Pilot B\nPilot C'
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: value })
  usePreferenceStore.setState({ locale: 'en-US' })
  const view = open('/admissions/applications/17')
  expect(await screen.findByText('Pilot B Pilot C')).toBeInTheDocument()
  view.unmount()
  open('/admissions/applications/17/edit')
  expect(await screen.findByRole('textbox', { name: 'Application note (optional)' })).toHaveValue(
    'Pilot B\nPilot C'
  )
})
test('rejected application resubmits the original ID and current version, preserving rejection history', async () => {
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: detail('rejected') })
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
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: pending })
  mocks.invite
    .mockRejectedValueOnce(new Error('admissions.errors.walletDisabled'))
    .mockResolvedValueOnce(success({ ...pending.application, status: 'invited' }))
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
  await waitFor(() => expect(mocks.invite).toHaveBeenCalledTimes(2))
  expect(mocks.invite).toHaveBeenLastCalledWith(17, { expected_version: 3, reason: '' })
})
test('English management filters include kind and only reviewable corporations', async () => {
  usePreferenceStore.setState({ locale: 'en-US' })
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  open('/admissions/review')
  fireEvent.change(await screen.findByRole('combobox', { name: 'Application type' }), {
    target: { value: 'alt' },
  })
  await waitFor(() =>
    expect(mocks.reviews).toHaveBeenLastCalledWith({
      current: 1,
      size: 20,
      status: 'pending',
      kind: 'alt',
    })
  )
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
  mocks.reviews.mockResolvedValue(success({ list: [app], total: 1, page: 1, pageSize: 20 }))
  open('/admissions/review')
  fireEvent.click(await screen.findByRole('tab', { name: 'Processed' }))
  expect(await screen.findByText('Handled by: Reviewer Seven')).toBeInTheDocument()
  expect(screen.getByText('Bind missing pilot')).toBeInTheDocument()
  expect(screen.getByRole('cell', { name: 'Apply for alts' })).toBeInTheDocument()
  expect(document.querySelector('time')).toHaveAttribute('dateTime', app.closed_at)
  await waitFor(() =>
    expect(mocks.reviews).toHaveBeenLastCalledWith({
      current: 1,
      size: 20,
      status: 'processed',
      kind: '',
    })
  )
})

test('expired rejected evidence uses the resubmit endpoint from the primary submit action', async () => {
  const rejected = detail('rejected')
  rejected.application.evidence_expires_at = '2025-01-01T00:00:00Z'
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: rejected })
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
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: value })
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
    mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: value })
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
  mocks.detail.mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: { ...detail(status), is_owner: owner },
  })
  open('/admissions/applications/17/edit')
  await screen.findByRole('alert')
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
})

test('history offers separate editing only for unfinished applications', async () => {
  mocks.history.mockResolvedValue(
    success({
      list: [detail().application, { ...detail('invited').application, id: 18 }],
      total: 2,
      page: 1,
      pageSize: 20,
    })
  )
  open('/admissions/history')
  const edit = await screen.findByRole('button', { name: '修改申请' })
  expect(mocks.reviews).not.toHaveBeenCalled()
  expect(screen.getAllByRole('button', { name: '修改申请' })).toHaveLength(1)
  expect(screen.getAllByRole('button', { name: '查看详情' })).toHaveLength(2)
  expect(screen.queryByRole('button', { name: '已邀请' })).toBeNull()
  fireEvent.click(edit)
  expect(await screen.findByRole('heading', { name: '修改申请 #17' })).toBeInTheDocument()
})

test('review list invites pending alt applications directly, except the reviewer own', async () => {
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  const alt = {
    ...detail().application,
    id: 18,
    kind: 'alt' as const,
    version: 4,
    characters: [{ character_id: 201, character_name: 'Alt' }],
  }
  const own = { ...alt, id: 19 }
  own.characters = [{ character_id: 101, character_name: 'Pilot A' }]
  const processed = { ...alt, id: 20, status: 'invited' as const }
  mocks.reviews.mockResolvedValue(
    success({
      list: [{ ...detail().application, user_id: 5 }, alt, own, processed],
      total: 4,
      page: 1,
      pageSize: 20,
    })
  )
  const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
  open('/admissions/review')
  const invite = await screen.findAllByRole('button', { name: '已邀请' })
  expect(invite).toHaveLength(1)
  expect(screen.getAllByRole('button', { name: '查看详情' })).toHaveLength(4)
  fireEvent.click(invite[0])
  await waitFor(() =>
    expect(mocks.invite).toHaveBeenCalledWith(18, { expected_version: 4, reason: '' })
  )
  expect(confirm).toHaveBeenCalledWith(expect.not.stringContaining('工资'))
  expect(operationRates).not.toHaveBeenCalled()
  await waitFor(() => expect(mocks.reviews).toHaveBeenCalledTimes(2))
})

test.each([false, true])(
  'recruitment salary confirmation requires consent: %s',
  async (consent) => {
    mocks.detail.mockResolvedValue({
      code: 200,
      msg: 'ok',
      data: { ...detail(), can_review: true, is_owner: false },
    })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open('/admissions/applications/17')
    fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(expect.stringContaining('4.25 伏羲币工资'))
    )
    if (consent)
      await waitFor(() =>
        expect(mocks.invite).toHaveBeenCalledWith(17, { expected_version: 3, reason: '' })
      )
    else expect(mocks.invite).not.toHaveBeenCalled()
  }
)

test('alt invitation confirmation shows no recruitment salary', async () => {
  const value = detail()
  value.application.kind = 'alt'
  mocks.detail.mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: { ...value, can_review: true, is_owner: false },
  })
  const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
  open('/admissions/applications/17')
  fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
  await waitFor(() =>
    expect(mocks.invite).toHaveBeenCalledWith(17, { expected_version: 3, reason: '' })
  )
  expect(confirm).toHaveBeenCalledWith(expect.not.stringContaining('工资'))
  expect(operationRates).not.toHaveBeenCalled()
})

test('blacklist hits warn the reviewer and disable only Invited', async () => {
  mocks.detail.mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: {
      ...detail(),
      can_review: true,
      is_owner: false,
      blacklist_hits: blacklistHits,
    },
  })
  open('/admissions/applications/17')
  const invited = await screen.findByRole('button', { name: '已邀请' })
  const warning = screen.getByRole('alert')
  expect(warning).toHaveTextContent('申请人命中联盟黑名单')
  expect(warning).toHaveTextContent('人物: Bad Pilot（9555）')
  expect(warning).toHaveTextContent('QQ: 11111111')
  expect(screen.getByText(/Scam/)).toHaveClass('whitespace-pre-line')
  expect(invited).toBeDisabled()
  fireEvent.change(screen.getByPlaceholderText('拒绝申请时必须填写原因'), {
    target: { value: 'No' },
  })
  const reject = screen.getByRole('button', { name: '拒绝' })
  expect(reject).toBeEnabled()
  vi.spyOn(window, 'confirm').mockReturnValue(true)
  fireEvent.click(reject)
  await waitFor(() =>
    expect(mocks.reject).toHaveBeenCalledWith(17, { expected_version: 3, reason: 'No' })
  )
  expect(mocks.invite).not.toHaveBeenCalled()
})

test('applicant view never shows the blacklist warning', async () => {
  mocks.detail.mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: { ...detail(), blacklist_hits: blacklistHits },
  })
  open('/admissions/applications/17')
  await screen.findByText('Applicant', { exact: false })
  expect(screen.queryByText('申请人命中联盟黑名单', { exact: false })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '已邀请' })).not.toBeInTheDocument()
})

test('review list marks blacklisted rows and disables their Invited button', async () => {
  mocks.context = { ...context(true), reviewable_corporation_ids: [98185110] }
  const alt = {
    ...detail().application,
    id: 18,
    kind: 'alt' as const,
    characters: [{ character_id: 201, character_name: 'Alt' }],
    blacklisted: false,
  }
  mocks.reviews.mockResolvedValue(
    success({
      list: [alt, { ...alt, id: 19, blacklisted: true }],
      total: 2,
      page: 1,
      pageSize: 20,
    })
  )
  open('/admissions/review')
  const invite = await screen.findAllByRole('button', { name: '已邀请' })
  expect(invite.map((button) => (button as HTMLButtonElement).disabled)).toEqual([false, true])
  expect(screen.getAllByText('黑名单')).toHaveLength(1)
  expect(screen.getAllByRole('row')[2]).toHaveTextContent('黑名单')
})

test('Invited refused for a newly blacklisted applicant reloads the detail', async () => {
  const clean = { ...detail(), can_review: true, is_owner: false }
  mocks.detail.mockResolvedValueOnce({ code: 200, msg: 'ok', data: clean }).mockResolvedValueOnce({
    code: 200,
    msg: 'ok',
    data: { ...clean, blacklist_hits: blacklistHits },
  })
  mocks.invite.mockRejectedValue(new Error('admissions.errors.blacklisted'))
  vi.spyOn(window, 'confirm').mockReturnValue(true)
  open('/admissions/applications/17')
  fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
  expect(
    await screen.findByText('申请人命中联盟黑名单，不能登记为已邀请；仍可驳回。', { exact: false })
  ).toBeInTheDocument()
  expect(mocks.detail).toHaveBeenCalledTimes(2)
  expect(await screen.findByRole('button', { name: '已邀请' })).toBeDisabled()
  expect(screen.getByText(/Bad Pilot/)).toBeInTheDocument()
})

test('recruitment cannot submit when current salary cannot be read', async () => {
  mocks.detail.mockResolvedValue({ code: 200, msg: 'ok', data: { ...detail(), can_review: true } })
  vi.mocked(operationRates).mockRejectedValueOnce(new Error('offline'))
  const confirm = vi.spyOn(window, 'confirm')
  open('/admissions/applications/17')
  fireEvent.click(await screen.findByRole('button', { name: '已邀请' }))
  await screen.findByRole('alert')
  expect(confirm).not.toHaveBeenCalled()
  expect(mocks.invite).not.toHaveBeenCalled()
})
