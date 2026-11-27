import assert from 'node:assert/strict'
import test from 'node:test'
import { HttpError } from '@/utils/http/http-error'
import type { FleetQQPingReceipt } from '@/api/generated/client'
import {
  createFleetQQPingIntent,
  fleetQQPingErrorKey,
  isFleetCreationUnconfirmed,
  isFleetQQAdmissionError
} from './qqPing'

const receipt: FleetQQPingReceipt = {
  tasks: [{ task_id: 'task-id', status: 'queued', expires_at: '2026-11-26T20:00:00Z' }]
}

test('blocks duplicate submissions and reuses the UUID after a lost acknowledgment', async () => {
  const first = '6b6ef469-81bf-4679-8585-55d97eb0b697'
  const second = '0cddb9be-6837-4a96-852b-178f69dab062'
  const keys = [first, second]
  const intent = createFleetQQPingIntent(() => keys.shift()!)
  const sent: string[] = []
  const fail = Promise.withResolvers<FleetQQPingReceipt>()
  const request = intent.submit((key) => {
    sent.push(key)
    return fail.promise
  })!
  const send = async (key: string) => {
    sent.push(key)
    return receipt
  }

  assert.equal(intent.submit(send), undefined)
  assert.equal(intent.resetAfterAcceptance(), false)
  fail.reject(new Error('connection lost'))
  await assert.rejects(request, /connection lost/)
  assert.equal(intent.resetAfterAcceptance(), false)
  assert.deepEqual(await intent.submit(send), receipt)
  assert.equal(intent.submit(send), undefined)
  assert.equal(intent.resetAfterAcceptance(), true)
  assert.deepEqual(await intent.submit(send), receipt)
  assert.deepEqual(sent, [first, first, second])
})

test('localizes known admission rejections without exposing raw server messages', () => {
  const error = (status: number, msg: string) =>
    new HttpError('HTTP error', status, { data: { msg } })
  const rejections = [
    [error(409, 'no_targets'), 'fleet.qqPing.noTargets'],
    [error(429, 'queue_full'), 'qqOfficial.rateLimited'],
    [error(503, 'disabled'), 'fleet.qqPing.disabled'],
    [error(503, 'unavailable'), 'qqOfficial.unavailable']
  ] as const

  for (const [error, key] of rejections) {
    assert.equal(isFleetQQAdmissionError(error), true)
    assert.equal(fleetQQPingErrorKey(error), key)
  }
  assert.equal(fleetQQPingErrorKey(error(409, 'raw provider details')), 'qqOfficial.conflict')
  assert.equal(isFleetQQAdmissionError(new TypeError('connection lost')), false)
  assert.equal(isFleetCreationUnconfirmed(new TypeError('connection lost')), true)
  assert.equal(isFleetCreationUnconfirmed(error(400, 'invalid_input')), false)
})
