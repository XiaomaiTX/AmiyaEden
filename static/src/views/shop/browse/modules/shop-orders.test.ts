import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./shop-orders.vue', import.meta.url), 'utf8')
const docSource = readFileSync(
  new URL('../../../../../../docs/features/current/commerce.md', import.meta.url),
  'utf8'
)
const zhLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/zh.json', import.meta.url), 'utf8')
)
const enLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/en.json', import.meta.url), 'utf8')
)

test('shop orders show submitter and processor remark columns distinctly', () => {
  assert.match(source, /prop:\s*'remark'[\s\S]*label:\s*t\('shop\.submitterRemark'\)/)
  assert.match(source, /prop:\s*'remark'[\s\S]*row\.remark \|\| '-'/)
  assert.match(source, /const REVIEW_REMARK_COLUMN_PROP = 'review_remark'/)
  assert.match(source, /label:\s*t\('shop\.reviewRemark'\)/)
  assert.match(source, /rows\.some\(\(row\) => Boolean\(row\.review_remark\?\.trim\(\)\)\)/)
  assert.match(source, /addColumn\(REVIEW_REMARK_COLUMN, insertIndex\)/)
  assert.match(source, /removeColumn\(REVIEW_REMARK_COLUMN_PROP\)/)
  assert.match(docSource, /下单备注/)
  assert.match(docSource, /处理备注/)
  assert.equal(zhLocale.shop.submitterRemark, '下单备注')
  assert.equal(zhLocale.shop.reviewRemark, '处理备注')
  assert.equal(enLocale.shop.submitterRemark, 'Submitter Remark')
  assert.equal(enLocale.shop.reviewRemark, 'Processor Remark')
})
