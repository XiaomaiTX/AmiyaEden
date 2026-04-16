import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const applySource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const zhLocale = JSON.parse(
  readFileSync(new URL('../../../locales/langs/zh.json', import.meta.url), 'utf8')
)
const enLocale = JSON.parse(
  readFileSync(new URL('../../../locales/langs/en.json', import.meta.url), 'utf8')
)

test('srp apply table shows review note column', () => {
  assert.match(applySource, /prop:\s*'review_note'/)
  assert.match(applySource, /label:\s*t\('srp\.apply\.columns\.reviewNote'\)/)
  assert.match(applySource, /row\.review_note \|\| '-'/)
  assert.equal(zhLocale.srp.apply.columns.reviewNote, '审批备注')
  assert.equal(enLocale.srp.apply.columns.reviewNote, 'Review Note')
})

test('srp apply requests backend-filtered dropdown killmails and defaults the application table to 20 rows', () => {
  assert.match(applySource, /const KILLMAIL_DROPDOWN_LIMIT = 50/)
  assert.match(applySource, /exclude_submitted:\s*true/)
  assert.match(applySource, /fetchFleetKillmails\(form\.fleet_id,\s*killmailDropdownParams\)/)
  assert.match(applySource, /fetchMyKillmails\(killmailDropdownParams\)/)
  assert.match(applySource, /apiParams:\s*\{\s*current:\s*1,\s*size:\s*20\s*\}/)
  assert.doesNotMatch(applySource, /submittedKmIds/)
})

test('srp apply completes the table overflow chain for the lower card', () => {
  assert.match(applySource, /class="art-table-card srp-apply-table-card"/)
  assert.match(applySource, /class="srp-apply-table-shell"/)
  assert.match(
    applySource,
    /\.srp-apply-table-card :deep\(\.el-card__body\)\s*\{[\s\S]*overflow:\s*hidden/
  )
  assert.match(applySource, /\.srp-apply-table-shell\s*\{[\s\S]*min-height:\s*0/)
})
