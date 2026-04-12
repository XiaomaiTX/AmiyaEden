import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const newbroApiSource = readFileSync(new URL('../../../api/newbro.ts', import.meta.url), 'utf8')
const zhLocaleSource = readFileSync(
  new URL('../../../locales/langs/zh.json', import.meta.url),
  'utf8'
)
const enLocaleSource = readFileSync(
  new URL('../../../locales/langs/en.json', import.meta.url),
  'utf8'
)

test('characters page renders a dedicated expired-esi alert', () => {
  assert.match(source, /hasInvalidCharacterToken/)
  assert.match(source, /enforceCharacterESIRestriction/)
  assert.match(source, /characters\.tokenHealth/)
  assert.match(source, /type="error"/)
})

test('corp km enable button calls the zero-argument handler without passing a character', () => {
  assert.match(source, /const handleEnableCorpKm = async \(\) =>/)
  assert.doesNotMatch(source, /@click="handleEnableCorpKm\(char\)"/)
  assert.match(source, /@click="handleEnableCorpKm"/)
})

test('corp km controls stay limited to admin roles', () => {
  assert.match(source, /const canManageCorpKm = computed\(\(\) => \{/)
  assert.match(source, /roles\.some\(\(r\) => \['super_admin', 'admin'\]\.includes\(r\)\)/)
})

test('characters page wires a direct referral card below the profile form', () => {
  assert.match(newbroApiSource, /export function fetchDirectReferralStatus\(/)
  assert.match(newbroApiSource, /export function checkDirectReferrerQQ\(/)
  assert.match(newbroApiSource, /export function confirmDirectReferrer\(/)

  assert.match(source, /characters\.directReferral\.title/)
  assert.match(source, /characters\.directReferral\.subtitle/)
  assert.match(source, /characters\.directReferral\.referrerQQ/)
  assert.match(source, /characters\.directReferral\.checkBtn/)
  assert.match(source, /characters\.directReferral\.confirmBtn/)
  assert.match(source, /const directReferralStatus = ref</)
  assert.match(source, /const directReferrerCandidate = ref</)
  assert.match(source, /const loadDirectReferralStatus = async \(\) =>/)
  assert.match(source, /const handleCheckDirectReferrer = async \(\) =>/)
  assert.match(source, /const handleConfirmDirectReferrer = async \(\) =>/)
  assert.match(source, /v-if="directReferralStatus\.needs_profile_qq"/)
  assert.match(source, /referrer_user_id: directReferrerCandidate\.value\.user_id/)
})

test('characters page locales include direct referral copy', () => {
  assert.match(zhLocaleSource, /"directReferral"\s*:\s*\{[\s\S]*"title"\s*:/)
  assert.match(zhLocaleSource, /"checkBtn"\s*:\s*"检查 QQ"/)
  assert.match(zhLocaleSource, /"confirmBtn"\s*:/)

  assert.match(enLocaleSource, /"directReferral"\s*:\s*\{[\s\S]*"title"\s*:/)
  assert.match(enLocaleSource, /"checkBtn"\s*:/)
  assert.match(enLocaleSource, /"confirmBtn"\s*:/)
})
