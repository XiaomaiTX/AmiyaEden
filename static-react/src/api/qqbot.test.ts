import { beforeEach, expect, test, vi } from 'vitest'
vi.mock('@/api/http-client', () => ({ requestJson: vi.fn() }))
import { requestJson } from '@/api/http-client'
import {
  fetchQQBotConfig,
  saveQQBotConfig,
  cancelQQBotOperation,
  retryQQBotOperation,
  retryQQBotDelivery,
  pauseQQBotCircuit,
} from '@/api/qqbot'
import { appRouteSpecs } from '@/app/migration-routes'
beforeEach(() => {
  vi.mocked(requestJson).mockReset().mockResolvedValue({ code: 0, msg: '', data: null })
})
test('configuration preserves absent secrets and operational requests use restricted endpoints', async () => {
  await fetchQQBotConfig()
  const config = { enabled: false, bot_id: 42, cidrs: '127.0.0.1/32', interval_ms: 100 }
  await saveQQBotConfig(config)
  expect(requestJson).toHaveBeenLastCalledWith('/api/v1/system/qqbot/config', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
  await cancelQQBotOperation('abc/def')
  expect(requestJson).toHaveBeenLastCalledWith('/api/v1/system/qqbot/operations/abc%2Fdef/cancel', {
    method: 'POST',
  })
  await retryQQBotOperation('op')
  await retryQQBotDelivery('delivery')
  await pauseQQBotCircuit('write', true)
  expect(requestJson).toHaveBeenLastCalledWith('/api/v1/system/qqbot/circuits/write', {
    method: 'PUT',
    body: '{"paused":true}',
  })
})
test('management route is super administrator only', () => {
  const route = appRouteSpecs.find((r) => r.path === 'system/qqbot')
  expect(route?.meta?.roles).toEqual(['super_admin'])
  expect(route?.meta?.corpCapabilitiesAny).toEqual(['system.manage'])
})
