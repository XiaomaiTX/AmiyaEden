export interface WebhookConfig {
  url: string
  enabled: boolean
  type: 'discord' | 'feishu' | 'dingtalk' | 'onebot' | string
  fleet_template: string
  ob_target_type: 'group' | 'private'
  ob_target_id: number
  ob_token: string
}

export interface WebhookTestParams {
  url: string
  type: string
  content?: string
  ob_target_type?: string
  ob_target_id?: number
  ob_token?: string
}
