export type WebhookType =
  | 'discord'
  | 'feishu'
  | 'dingtalk'
  | 'onebot'
  | 'qq_governance_onebot'
  | string

export interface WebhookConfig {
  url: string
  enabled: boolean
  type: WebhookType
  fleet_template: string
  ob_target_type: 'group' | 'private'
  ob_target_id: number
  ob_token: string
  qq_governance_group_ids: number[]
}

export interface WebhookTestParams {
  url?: string
  type: WebhookType
  content?: string
  ob_target_type?: string
  ob_target_id?: number
  ob_token?: string
  qq_governance_group_ids?: number[]
}
