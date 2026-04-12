import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const viewSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const typeSource = readFileSync(new URL('../../../types/api/api.d.ts', import.meta.url), 'utf8')
const zhLocaleSource = readFileSync(
  new URL('../../../locales/langs/zh.json', import.meta.url),
  'utf8'
)
const enLocaleSource = readFileSync(
  new URL('../../../locales/langs/en.json', import.meta.url),
  'utf8'
)

test('recruit link types and page support direct referral records separately from generated links', () => {
  assert.match(typeSource, /interface RecruitEntry\s*\{[\s\S]*source: 'link' \| 'direct_referral'/)
  assert.match(typeSource, /interface RecruitLink\s*\{[\s\S]*source: 'link' \| 'direct_referral'/)

  assert.match(viewSource, /link\.source === 'direct_referral'/)
  assert.match(viewSource, /newbro\.recruitLink\.source\./)
  assert.match(viewSource, /newbro\.recruitLink\.directReferralRecord/)
  assert.match(viewSource, /find\(\(link\) => link\.source === 'link'\)/)
})

test('recruit link locales include direct referral labels', () => {
  assert.match(zhLocaleSource, /"directReferralRecord"\s*:/)
  assert.match(zhLocaleSource, /"source"\s*:\s*\{[\s\S]*"direct_referral"\s*:/)

  assert.match(enLocaleSource, /"directReferralRecord"\s*:/)
  assert.match(enLocaleSource, /"source"\s*:\s*\{[\s\S]*"direct_referral"\s*:/)
})
