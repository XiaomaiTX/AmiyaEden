export type WebhookType =
  | 'discord'
  | 'feishu'
  | 'dingtalk'
  | string

export interface WebhookConfig {
  url: string
  enabled: boolean
  type: WebhookType
  fleet_template: string
}

export interface WebhookTestParams {
  url?: string
  type: WebhookType
  content?: string
}
