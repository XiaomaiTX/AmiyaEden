import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const manageHookSource = readFileSync(
  new URL('../../../hooks/srp/useSrpManage.ts', import.meta.url),
  'utf8'
)
const workflowHookSource = readFileSync(
  new URL('../../../hooks/srp/useSrpWorkflow.ts', import.meta.url),
  'utf8'
)
const zhLocale = JSON.parse(
  readFileSync(new URL('../../../locales/langs/zh.json', import.meta.url), 'utf8')
)
const enLocale = JSON.parse(
  readFileSync(new URL('../../../locales/langs/en.json', import.meta.url), 'utf8')
)

test('srp manage uses the shared copy button for the character column and shared clipboard hook for copy flows', () => {
  assert.match(
    manageHookSource,
    /prop:\s*'character_name'[\s\S]*h\(ArtCopyButton,[\s\S]*text:\s*row\.character_name/
  )
  assert.match(workflowHookSource, /useClipboardCopy/)
  assert.doesNotMatch(workflowHookSource, /navigator\.clipboard\.writeText/)
})

test('srp batch payout copy text keeps exact ISK values instead of smart-abbreviated amounts', () => {
  assert.match(workflowHookSource, /formatBatchPayoutLine[\s\S]*formatIskPlain\(totalAmount\)/)
})

test('srp manage labels the last actor as the SRP officer after the review note column', () => {
  const reviewNoteColumnIndex = manageHookSource.indexOf("prop: 'review_note'")
  const lastActorColumnIndex = manageHookSource.indexOf("prop: 'last_actor_nickname'")
  const reviewNoteHeaderIndex = manageHookSource.indexOf("review_note: '审批备注'")
  const lastActorHeaderIndex = manageHookSource.indexOf("last_actor_nickname: '补损官'")
  const reviewNoteExportIndex = manageHookSource.indexOf("review_note: app.review_note || '-'")
  const lastActorExportIndex = manageHookSource.indexOf(
    "last_actor_nickname: app.last_actor_nickname || '-'"
  )
  const payoutStatusExportIndex = manageHookSource.indexOf(
    "payout_status: app.payout_status === 'paid' ? t('srp.status.paid') : t('srp.status.notpaid')"
  )

  assert.ok(reviewNoteColumnIndex >= 0)
  assert.ok(lastActorColumnIndex > reviewNoteColumnIndex)
  assert.ok(reviewNoteHeaderIndex >= 0)
  assert.ok(lastActorHeaderIndex > reviewNoteHeaderIndex)
  assert.ok(reviewNoteExportIndex >= 0)
  assert.ok(lastActorExportIndex > reviewNoteExportIndex)
  assert.ok(payoutStatusExportIndex > lastActorExportIndex)
  assert.match(manageHookSource, /label:\s*t\('srp\.manage\.columns\.lastActor'\)/)
  assert.equal(zhLocale.srp.manage.columns.lastActor, '补损官')
  assert.equal(enLocale.srp.manage.columns.lastActor, 'SRP Officer')
})
