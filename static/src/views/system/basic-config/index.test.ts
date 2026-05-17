import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../../api/sys-config.ts', import.meta.url), 'utf8')
const typeSource = readFileSync(new URL('../../../types/api/api.d.ts', import.meta.url), 'utf8')
const zhLocaleSource = readFileSync(
  new URL('../../../locales/langs/zh.json', import.meta.url),
  'utf8'
)
const enLocaleSource = readFileSync(
  new URL('../../../locales/langs/en.json', import.meta.url),
  'utf8'
)

test('basic config page exposes SDE status block and check/update actions', () => {
  assert.match(source, /fetchSDEStatus\(/)
  assert.match(source, /checkSDEVersion\(/)
  assert.match(source, /triggerSDEUpdate\(/)
  assert.match(source, /sdeStatus\.has_update/)
  assert.match(source, /system\.basicConfig\.sdeCheckVersion/)
  assert.match(source, /system\.basicConfig\.sdeRunUpdate/)
  assert.match(source, /system\.basicConfig\.sdeUpdateAvailable/)
})

test('sys config API and types include SDE status interfaces', () => {
  assert.match(apiSource, /export function fetchSDEStatus\(/)
  assert.match(apiSource, /export function checkSDEVersion\(/)
  assert.match(apiSource, /export function triggerSDEUpdate\(/)
  assert.match(typeSource, /interface SDEStatus \{/)
  assert.match(typeSource, /has_update: boolean/)
  assert.match(typeSource, /last_check_error: string/)
  assert.match(typeSource, /last_update_error: string/)
  assert.match(apiSource, /export function fetchCorporationAccessPolicies\(/)
  assert.match(apiSource, /export function updateCorporationAccessPolicies\(/)
  assert.match(typeSource, /interface CorporationAccessPoliciesConfig \{/)
  assert.match(typeSource, /interface CorporationAccessPolicy \{/)
})

test('sde locale strings are present in zh and en', () => {
  assert.match(zhLocaleSource, /"sdeCheckVersion"\s*:\s*"检查新版本"/)
  assert.match(zhLocaleSource, /"sdeRunUpdate"\s*:\s*"执行更新"/)
  assert.match(zhLocaleSource, /"sdeUpdateAvailable"\s*:/)
  assert.match(enLocaleSource, /"sdeCheckVersion"\s*:\s*"Check Latest Version"/)
  assert.match(enLocaleSource, /"sdeRunUpdate"\s*:\s*"Run Update"/)
  assert.match(enLocaleSource, /"sdeUpdateAvailable"\s*:/)
  assert.match(zhLocaleSource, /"corporationAccessPolicies"\s*:/)
  assert.match(enLocaleSource, /"corporationAccessPolicies"\s*:/)
  assert.match(zhLocaleSource, /"corpCapabilityGroups"\s*:/)
  assert.match(enLocaleSource, /"corpCapabilityGroups"\s*:/)
  assert.match(source, /corpCapabilityLabelKeys\[capability\]/)
  assert.doesNotMatch(source, /\{\{\s*capability\s*\}\}/)
  assert.match(source, /fetchCorporationAccessPolicies\(/)
  assert.match(source, /updateCorporationAccessPolicies\(/)
  assert.match(source, /srp\.recommendation_multiplier/)
})
