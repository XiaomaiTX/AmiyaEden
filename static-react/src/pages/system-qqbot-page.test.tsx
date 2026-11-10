import { render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { SystemQQBotPage } from '@/pages/system-qqbot-page'
import {
  fetchQQBotSnapshot,
  saveQQBotConfig,
  cancelQQBotOperation,
  retryQQBotOperation,
} from '@/api/qqbot'
vi.mock('@/api/qqbot', () => ({
  fetchQQGroupBusiness: vi
    .fn()
    .mockResolvedValue({
      rules: [],
      applications: [],
      members: [],
      inspections: [],
      judgements: [],
    }),
  saveQQGroupRule: vi.fn(),
  fetchQQNotificationChannels: vi.fn().mockResolvedValue([]),
  fetchQQNotificationTargets: vi.fn().mockResolvedValue([]),
  saveQQNotificationChannel: vi.fn(),
  testQQNotification: vi.fn(),
  fetchQQBotConfig: vi
    .fn()
    .mockResolvedValue({ enabled: false, bot_id: 42, cidrs: '127.0.0.1/32', interval_ms: 100 }),
  fetchQQBotSnapshot: vi.fn().mockResolvedValue({
    connected: false,
    redis_available: true,
    rate_wait_ms: 0,
    last_frame: '',
    ingress_error: '',
    operations: [
      {
        id: 'unknown',
        business_key: 'business-send',
        kind: 'group_message',
        status: 'unknown',
        failures: 0,
        result: 'unconfirmed',
        wait_reason: '',
      },
    ],
    attempts: [],
    events: [],
    deliveries: [],
    circuits: [],
  }),
  fetchQQBotOperation: vi.fn(),
  saveQQBotConfig: vi.fn(),
  cancelQQBotOperation: vi.fn(),
  retryQQBotOperation: vi.fn(),
  retryQQBotDelivery: vi.fn(),
  pauseQQBotCircuit: vi.fn(),
}))
test('loads operational state without submitting writes and does not offer retry for unknown message results', async () => {
  render(
    <I18nProvider>
      <SystemQQBotPage />
    </I18nProvider>
  )
  await screen.findByText('business-send')
  expect(screen.getByText('unconfirmed')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '受限重试' })).not.toBeInTheDocument()
  await waitFor(() => expect(fetchQQBotSnapshot).toHaveBeenCalled())
  expect(saveQQBotConfig).not.toHaveBeenCalled()
  expect(cancelQQBotOperation).not.toHaveBeenCalled()
  expect(retryQQBotOperation).not.toHaveBeenCalled()
})
