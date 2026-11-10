import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthRegistrationForm } from './auth-registration-form'
import { ContactConflictDialog } from '@/components/contact-conflict-dialog'
import { usePreferenceStore, useSessionStore } from '@/stores'
import { I18nProvider } from '@/i18n'

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('contact registration', () => {
  beforeEach(() => {
    useSessionStore.getState().clearSession()
    usePreferenceStore.getState().setLocale('zh-CN')
    vi.restoreAllMocks()
  })

  test('shows primary owner guidance for duplicate QQ without logging in or losing form data', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        jsonResponse({ code: 200, data: { character_id: 9002, character_name: 'New Character' } })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          code: 409,
          msg: 'contact_already_bound',
          data: {
            contact: 'qq',
            primary_character_id: 9001,
            primary_character_name: 'Primary Owner',
          },
        })
      )
    const completed = vi.fn()
    render(
      <>
        <ContactConflictDialog />
        <AuthRegistrationForm ticket={'a'.repeat(64)} onComplete={completed} onLogin={vi.fn()} />
      </>
    )
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByRole('button', { name: '完成注册' })).toBeEnabled())
    expect(screen.getByRole('textbox', { name: '昵称' })).toHaveValue('')
    await user.type(screen.getByRole('textbox', { name: '昵称' }), 'New Character')
    expect(screen.getByRole('radio', { name: 'QQ' })).toBeChecked()
    expect(screen.queryByRole('textbox', { name: 'Discord ID' })).not.toBeInTheDocument()
    await user.type(screen.getByRole('textbox', { name: 'QQ' }), '123456')
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent(
      '这个 QQ 号已经被「Primary Owner」绑定'
    )
    expect(screen.getByRole('dialog')).toHaveTextContent('绑定新人物')
    await user.keyboard('{Escape}')
    expect(screen.getByRole('textbox', { name: 'QQ' })).toHaveValue('123456')
    expect(completed).not.toHaveBeenCalled()
    expect(useSessionStore.getState().isLoggedIn).toBe(false)
    const [, options] = fetchSpy.mock.calls[1]
    expect((options?.headers as Headers).get('X-Registration-Token')).toBe('a'.repeat(64))
  })

  test('rejects a blank contact and completes with Discord only', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        jsonResponse({ code: 200, data: { character_id: 9002, character_name: 'New Character' } })
      )
      .mockResolvedValueOnce(jsonResponse({ code: 200, data: { token: 'application-token' } }))
    const completed = vi.fn()
    render(
      <AuthRegistrationForm ticket={'b'.repeat(64)} onComplete={completed} onLogin={vi.fn()} />
    )
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByRole('button', { name: '完成注册' })).toBeEnabled())
    await user.type(screen.getByRole('textbox', { name: '昵称' }), 'New Character')
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('请填写昵称和所选的联系方式')
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    await user.type(screen.getByRole('textbox', { name: 'QQ' }), '123456')
    await user.click(screen.getByRole('radio', { name: 'Discord' }))
    expect(screen.queryByRole('textbox', { name: 'QQ' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('请填写昵称和所选的联系方式')
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole('radio', { name: 'QQ' }))
    expect(screen.getByRole('textbox', { name: 'QQ' })).toHaveValue('123456')
    await user.click(screen.getByRole('radio', { name: 'Discord' }))
    await user.type(screen.getByRole('textbox', { name: 'Discord ID' }), '123456789')
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    await waitFor(() => expect(completed).toHaveBeenCalledWith('application-token'))
    expect(JSON.parse(String(fetchSpy.mock.calls[1][1]?.body))).toEqual({
      nickname: 'New Character',
      qq: '',
      discord_id: '123456789',
    })
  })

  test('keeps nickname and selected contact while switching languages and rejects a one-character nickname', async () => {
    const fetchSpy = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(
        jsonResponse({ code: 200, data: { character_id: 9002, character_name: 'New Character' } })
      )
    render(
      <AuthRegistrationForm ticket={'c'.repeat(64)} onComplete={vi.fn()} onLogin={vi.fn()} />,
      { wrapper: I18nProvider }
    )
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByRole('button', { name: '完成注册' })).toBeEnabled())
    const nickname = screen.getByRole('textbox', { name: '昵称' })
    expect(nickname).toHaveValue('')
    expect(nickname).toBeRequired()
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    expect(nickname).toBeInvalid()
    await user.type(nickname, '中')
    await user.click(screen.getByRole('radio', { name: 'Discord' }))
    await user.type(screen.getByRole('textbox', { name: 'Discord ID' }), '123456789')
    await user.click(screen.getByRole('button', { name: '完成注册' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('昵称至少需要 2 个字符')
    await user.click(screen.getByRole('button', { name: 'EN' }))
    expect(screen.getByRole('heading', { name: 'Complete registration' })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('at least 2 characters')
    expect(screen.getByRole('textbox', { name: 'Nickname' })).toHaveValue('中')
    expect(screen.getByRole('radio', { name: 'Discord' })).toBeChecked()
    expect(screen.getByRole('textbox', { name: 'Discord ID' })).toHaveValue('123456789')
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole('button', { name: '中文' }))
    await user.type(screen.getByRole('textbox', { name: '昵称' }), '文')
    expect(screen.getByRole('textbox', { name: '昵称' })).toHaveValue('中文')
  })
})
