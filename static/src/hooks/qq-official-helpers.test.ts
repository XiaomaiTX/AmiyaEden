import assert from 'node:assert/strict'
import test from 'node:test'
import type {
  QQOfficialGroup,
  QQOfficialJoinRequest,
  QQOfficialStatus
} from '@/api/generated/client'
import {
  canCancelQQTask,
  canDecideQQRequest,
  createQQSubmissionKeys,
  hasUsableQQCredentials,
  isQQGroupOpenID,
  parseQQCorporationTarget,
  qqCredentialDraft
} from './qq-official-helpers'

const status: QQOfficialStatus = {
  app_id: '10001',
  has_app_secret: true,
  configured: true,
  enabled: true,
  paused: false,
  certified: false,
  approval_writes_enabled: true,
  token_state: 'ready',
  gateway_state: 'connected',
  queue_depth: 0,
  attempts_last_minute: 0,
  messages_last_minute: 0,
  message_budget: 24,
  cooldowns: []
}
const group: QQOfficialGroup = {
  id: 'local-group',
  alias: '',
  present: true,
  authorized: true,
  notification_enabled: false,
  test_verified: false,
  approval_enabled: true,
  push_state: 'unknown',
  approval_capability: 'available'
}
const pending: QQOfficialJoinRequest = {
  id: 'local-request',
  username: 'Pilot',
  status: 'pending',
  group_id: group.id,
  applied_at: '2026-11-26T11:00:00Z',
  expires_at: '2026-11-26T12:00:00Z'
}

test('join decisions require live local eligibility and do not depend on notification opt-in', () => {
  const now = Date.parse('2026-11-26T11:10:00Z')
  assert.equal(canDecideQQRequest(status, group, pending, now), true)
  assert.equal(canDecideQQRequest({ ...status, paused: true }, group, pending, now), false)
  assert.equal(
    canDecideQQRequest({ ...status, approval_writes_enabled: false }, group, pending, now),
    false
  )
  assert.equal(
    canDecideQQRequest(status, { ...group, approval_capability: 'forbidden' }, pending, now),
    false
  )
  assert.equal(canDecideQQRequest(status, group, { ...pending, status: 'queued' }, now), false)
  assert.equal(canDecideQQRequest(status, group, { ...pending, group_id: 'other' }, now), false)
  assert.equal(canDecideQQRequest(status, group, pending, Date.parse(pending.expires_at)), false)
})

test('a stored but unreadable secret cannot enable the bot or submit decisions', () => {
  const unavailable = { ...status, configured: false, token_state: 'credentials_unavailable' }
  assert.equal(hasUsableQQCredentials(status), true)
  assert.equal(hasUsableQQCredentials(unavailable), false)
  assert.equal(canDecideQQRequest(unavailable, group, pending), false)
})

test('credential updates omit blank secrets and require a secret to change AppID', () => {
  const secret = 'fixture-secret'
  assert.deepEqual(qqCredentialDraft(undefined, '', ''), {})
  assert.deepEqual(qqCredentialDraft('001234', ' 001234 ', ''), {})
  assert.deepEqual(qqCredentialDraft('001234', '56789', ''), {
    error: 'qqOfficial.appSecretRequired'
  })
  assert.deepEqual(qqCredentialDraft('001234', '001234', secret), {
    patch: { app_id: '001234', app_secret: secret }
  })
  assert.deepEqual(qqCredentialDraft('001234', ' 056789 ', secret), {
    patch: { app_id: '056789', app_secret: secret }
  })
  assert.deepEqual(qqCredentialDraft(undefined, 'invalid', secret), {
    error: 'qqOfficial.appIDInvalid'
  })
  for (const invalid of [' secret', 'secret\n', '密', 'x'.repeat(4097)]) {
    assert.deepEqual(qqCredentialDraft(undefined, '12345', invalid), {
      error: 'qqOfficial.appSecretInvalid'
    })
  }
})

test('cancel affordance only appears for queued and delayed retries', () => {
  assert.equal(canCancelQQTask('queued'), true)
  assert.equal(canCancelQQTask('retry_wait'), true)
  assert.equal(canCancelQQTask('executing'), false)
  assert.equal(canCancelQQTask('awaiting_review'), false)
})

test('target controls support explicit zero clearing and reject unsafe IDs', () => {
  assert.equal(parseQQCorporationTarget(''), 0)
  assert.equal(parseQQCorporationTarget('0'), 0)
  assert.equal(parseQQCorporationTarget(' 98824507 '), 98824507)
  for (const invalid of ['-1', '1.5', '0x1', '1e4', '9007199254740992']) {
    assert.equal(parseQQCorporationTarget(invalid), undefined)
  }
})

test('verification accepts only an official group identifier, not a URL or endpoint', () => {
  assert.equal(isQQGroupOpenID(' 8jETs_-MF_NhW82J '), true)
  assert.equal(isQQGroupOpenID('too-short'), false)
  assert.equal(isQQGroupOpenID('https://localhost/example'), false)
  assert.equal(isQQGroupOpenID('x'.repeat(129)), false)
})

test('submission keys survive failed acknowledgements and reset after receipt', () => {
  let sequence = 0
  const keys = createQQSubmissionKeys(() => `key-${++sequence}`)
  assert.equal(keys.get('one'), 'key-1')
  assert.equal(keys.get('two'), 'key-2')
  assert.equal(keys.get('one'), 'key-1')
  keys.clear('one')
  assert.equal(keys.get('one'), 'key-3')
})
