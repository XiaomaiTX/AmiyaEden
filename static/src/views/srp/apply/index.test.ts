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
