import type {
  QQOfficialGroup,
  QQOfficialJoinRequest,
  QQOfficialSettings,
  QQOfficialStatus,
} from '@/api/generated/client'

export const taskStatuses = [
  'queued',
  'executing',
  'retry_wait',
  'awaiting_review',
  'succeeded',
  'failed',
  'unknown',
  'expired',
  'cancelled',
] as const
export const taskKinds = ['send_text', 'group_state', 'join_list', 'join_decision'] as const
export const taskSources = [
  'test',
  'verification',
  'join_refresh',
  'join_decision',
  'fleet',
  'structure',
] as const
export const requestStatuses = [
  'pending',
  'queued',
  'approved',
  'declined',
  'closed_elsewhere',
  'unknown',
  'expired',
  'failed',
] as const

export function hasUsableQQCredentials(status: QQOfficialStatus | null) {
  return !!status?.configured && status.token_state !== 'credentials_unavailable'
}

export function isQQDispatchReady(status: QQOfficialStatus | null) {
  return !!status && hasUsableQQCredentials(status) && status.enabled && !status.paused
}

export function qqCredentialDraft(
  currentAppID: string | undefined,
  idInput: string,
  secret: string
) {
  const appID = idInput.trim()
  const idChanged = appID !== (currentAppID ?? '')
  if (!idChanged && secret === '') return {}
  if (!/^\d{1,32}$/.test(appID)) return { error: 'qqOfficial.appIDInvalid' }
  if (secret === '') return { error: 'qqOfficial.appSecretRequired' }
  if (secret.length > 4096 || /[^!-~]/.test(secret)) return { error: 'qqOfficial.appSecretInvalid' }

  // Always bind the entered secret to the explicit AppID, even for a rotation.
  const patch: QQOfficialSettings = { app_id: appID, app_secret: secret }
  return { patch }
}

export function canCancelQQTask(status: string) {
  return status === 'queued' || status === 'retry_wait'
}

export function shouldPollQQTask(status?: string) {
  return (
    status === undefined ||
    canCancelQQTask(status) ||
    status === 'executing' ||
    status === 'awaiting_review'
  )
}

export function canDecideQQRequest(
  status: QQOfficialStatus | null,
  group: QQOfficialGroup,
  request: QQOfficialJoinRequest,
  now = Date.now()
) {
  return (
    isQQDispatchReady(status) &&
    !!status?.approval_writes_enabled &&
    group.present &&
    group.authorized &&
    group.approval_enabled &&
    group.approval_capability === 'available' &&
    group.id === request.group_id &&
    request.status === 'pending' &&
    Date.parse(request.expires_at) > now
  )
}

export function isQQGroupOpenID(value: string) {
  return /^[A-Za-z0-9_-]{16,128}$/.test(value.trim())
}

export function parseQQCorporationTarget(text: string): number | undefined {
  const value = text.trim()
  if (value === '') return 0
  if (!/^\d+$/.test(value)) return undefined
  const number = Number(value)
  return Number.isSafeInteger(number) ? number : undefined
}

// Reuse a key when a submission's acknowledgement is lost. On acceptance the next
// intentional click gets a different key; callers clear any key whose input changes.
export function createQQSubmissionKeys(create: () => string = () => crypto.randomUUID()) {
  const pending = new Map<string, string>()
  return {
    get(action: string) {
      const existing = pending.get(action)
      if (existing) return existing
      const next = create()
      if (pending.size >= 64) {
        const oldest = pending.keys().next()
        if (!oldest.done) pending.delete(oldest.value)
      }
      pending.set(action, next)
      return next
    },
    clear(action: string) {
      pending.delete(action)
    },
  }
}

export function qqStateLabel(t: (key: string) => string, state?: string) {
  if (!state) return '—'
  const key = `qqOfficial.state.${state}`
  const label = t(key)
  return label === key ? t('qqOfficial.unknownState') : label
}

export function qqErrorKey(code?: number) {
  if (code === 400) return 'qqOfficial.invalid'
  if (code === 409) return 'qqOfficial.conflict'
  if (code === 429) return 'qqOfficial.rateLimited'
  if (code === 503) return 'qqOfficial.unavailable'
  return 'qqOfficial.actionFailed'
}
