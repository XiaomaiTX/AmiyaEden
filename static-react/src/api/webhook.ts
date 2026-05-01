import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type { WebhookConfig, WebhookTestParams } from '@/types/api/webhook'

export async function fetchWebhookConfig() {
  const response = await requestJson<ApiResponse<WebhookConfig>>('/api/v1/system/webhook/config')
  return assertSuccess(response, 'fetch webhook config failed')
}

export async function setWebhookConfig(data: WebhookConfig) {
  const response = await requestJson<ApiResponse<null>>('/api/v1/system/webhook/config', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  assertSuccess(response, 'save webhook config failed')
}

export async function testWebhook(data: WebhookTestParams) {
  const response = await requestJson<ApiResponse<null>>('/api/v1/system/webhook/test', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  assertSuccess(response, 'test webhook failed')
}
