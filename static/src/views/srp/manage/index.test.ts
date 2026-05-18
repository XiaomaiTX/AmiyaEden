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
const buttonTableSource = readFileSync(
  new URL('../../../components/core/forms/art-button-table/index.vue', import.meta.url),
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
  const reviewNoteHeaderIndex = manageHookSource.indexOf(
    "review_note: t('srp.manage.exportColumns.reviewNote')"
  )
  const lastActorHeaderIndex = manageHookSource.indexOf(
    "last_actor_nickname: t('srp.manage.exportColumns.lastActor')"
  )
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
  assert.equal(zhLocale.srp.manage.exportColumns.lastActor, '补损官')
  assert.equal(enLocale.srp.manage.exportColumns.lastActor, 'SRP Officer')
})

test('srp submitted actions keep approve and reject explicitly clickable with guarded click handlers', () => {
  assert.match(manageHookSource, /prop:\s*'actions'[\s\S]*width:\s*280/)
  assert.match(
    manageHookSource,
    /row\.review_status === 'submitted'[\s\S]*label:\s*t\('srp\.manage\.approveBtn'\)[\s\S]*onClick:\s*\(\)\s*=>\s*callbacks\.openReviewDialog\(row,\s*'approve'\)/
  )
  assert.match(
    manageHookSource,
    /row\.review_status === 'submitted'[\s\S]*label:\s*t\('srp\.manage\.rejectBtn'\)[\s\S]*onClick:\s*\(\)\s*=>\s*callbacks\.openReviewDialog\(row,\s*'reject'\)/
  )
  assert.doesNotMatch(manageHookSource, /event\.stopPropagation\(\)/)
  assert.match(manageHookSource, /flex-nowrap whitespace-nowrap/)
})

test('srp review dialog template fill is resilient and shows an error when dialog initialization fails', () => {
  assert.match(workflowHookSource, /const ensureText = \(value: unknown, fallback = ''\)/)
  assert.match(workflowHookSource, /const getReviewerName = \(\)/)
  assert.match(workflowHookSource, /if \(!row\) \{/)
  assert.match(
    workflowHookSource,
    /t\('srp\.manage\.defaultApproveNote',\s*\{\s*mainCharacterName:\s*reviewerName\s*\}\)/
  )
  assert.match(
    workflowHookSource,
    /t\('srp\.manage\.defaultRejectNote',\s*\{\s*mainCharacterName:\s*reviewerName\s*\}\)/
  )
  assert.match(workflowHookSource, /ElMessage\.error\(t\('common\.operationFailed'\)\)/)
  assert.doesNotMatch(
    workflowHookSource,
    /try \{[\s\S]*reviewDialogVisible\.value = true[\s\S]*\} catch \{/
  )
})

test('art button table stops click propagation at component boundary', () => {
  assert.match(buttonTableSource, /@click\.stop\.prevent="handleClick"/)
})
