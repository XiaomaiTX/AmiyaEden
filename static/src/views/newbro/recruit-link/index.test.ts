import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const viewSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../../api/newbro.ts', import.meta.url), 'utf8')
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
  assert.match(typeSource, /matched_character_name\?: string/)
  assert.match(
    typeSource,
    /interface AdminRecruitLink extends RecruitLink\s*\{[\s\S]*user_character_name: string/
  )

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

test('recruit link admin table uses character names and supports successful recruit details', () => {
  assert.match(viewSource, /colUserCharacter/)
  assert.doesNotMatch(viewSource, /colUserId/)
  assert.match(viewSource, /openSuccessfulRecruitDialog/)
  assert.match(viewSource, /successfulRecruitsTitle/)
  assert.match(viewSource, /noSuccessfulRecruits/)
  assert.match(viewSource, /entry\.matched_character_name \|\| entry\.qq/)
  assert.match(viewSource, /entry\.status === 'valid' && entry\.matched_user_id > 0/)

  assert.match(zhLocaleSource, /"colUserCharacter"\s*:/)
  assert.match(zhLocaleSource, /"colMatchedCharacter"\s*:/)
  assert.match(zhLocaleSource, /"successfulRecruitsTitle"\s*:/)
  assert.match(enLocaleSource, /"colUserCharacter"\s*:/)
  assert.match(enLocaleSource, /"colMatchedCharacter"\s*:/)
  assert.match(enLocaleSource, /"successfulRecruitsTitle"\s*:/)
})

test('recruit link page exposes an admin link settings tab backed by recruit settings APIs', () => {
  assert.match(viewSource, /newbro\.recruitLink\.settingsTab/)
  assert.match(viewSource, /name="settings"[\s\S]*lazy/)
  assert.match(viewSource, /ref<'my' \| 'admin' \| 'settings'>\('my'\)/)
  assert.match(viewSource, /if \(!admin && tab !== 'my'\)/)
  assert.match(
    viewSource,
    /if \(tab === 'settings'\) \{[\s\S]*void linkSettingsPanelRef\.value\?\.reloadSettings\(\)/
  )

  assert.match(apiSource, /export function fetchAdminNewbroRecruitSettings\(/)
  assert.match(apiSource, /export function updateAdminNewbroRecruitSettings\(/)
  assert.doesNotMatch(apiSource, /export function fetchAdminNewbroSettings\(/)
  assert.doesNotMatch(apiSource, /export function updateAdminNewbroSettings\(/)

  assert.match(zhLocaleSource, /"settingsTab"\s*:\s*"链接设置"/)
  assert.match(enLocaleSource, /"settingsTab"\s*:\s*"Link Settings"/)
})
