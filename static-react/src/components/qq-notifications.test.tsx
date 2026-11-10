import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { QQNotifications } from './qq-notifications'
import { testQQNotification, saveQQNotificationChannel } from '@/api/qqbot'
vi.mock('@/api/qqbot', () => ({
  fetchQQNotificationChannels: vi
    .fn()
    .mockResolvedValue([{ name: 'general', enabled: true, group_ids: '[8,9]' }]),
  fetchQQNotificationTargets: vi.fn().mockResolvedValue([]),
  saveQQNotificationChannel: vi.fn().mockResolvedValue(null),
  testQQNotification: vi.fn().mockRejectedValue(new Error('network')),
}))
test('test retries reuse identity and expiry; configuration sends only typed group IDs', async () => {
  render(
    <I18nProvider>
      <QQNotifications />
    </I18nProvider>
  )
  await screen.findByDisplayValue('8,9')
  const testButton = screen.getByRole('button', { name: '发送固定测试（五分钟有效）' })
  fireEvent.click(testButton)
  await waitFor(() => expect(testButton).not.toBeDisabled())
  fireEvent.click(testButton)
  await waitFor(() => expect(testQQNotification).toHaveBeenCalledTimes(2))
  expect(vi.mocked(testQQNotification).mock.calls[0]).toEqual(
    vi.mocked(testQQNotification).mock.calls[1]
  )
  await waitFor(() => expect(testButton).not.toBeDisabled())
  fireEvent.click(screen.getByRole('button', { name: '保存配置' }))
  await waitFor(() =>
    expect(saveQQNotificationChannel).toHaveBeenCalledWith('general', true, [8, 9])
  )
})
