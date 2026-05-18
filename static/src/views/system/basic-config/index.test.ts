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
  assert.match(source, /last_query_error/)
  assert.match(source, /last_query_error_at/)
  assert.match(source, /last_query_error_source/)
  assert.match(source, /system\.basicConfig\.sdeCheckVersion/)
  assert.match(source, /system\.basicConfig\.sdeRunUpdate/)
  assert.match(source, /system\.basicConfig\.sdeRunUpdateHint/)
  assert.match(source, /system\.basicConfig\.sdeUpdateAvailable/)
  assert.doesNotMatch(source, /:disabled="!sdeStatus\.has_update"/)
})

test('sys config API and types include SDE status interfaces', () => {
  assert.match(apiSource, /export function fetchSDEStatus\(/)
  assert.match(apiSource, /export function checkSDEVersion\(/)
  assert.match(apiSource, /export function triggerSDEUpdate\(/)
  assert.match(typeSource, /interface SDEStatus \{/)
  assert.match(typeSource, /has_update: boolean/)
  assert.match(typeSource, /last_check_error: string/)
  assert.match(typeSource, /last_update_error: string/)
  assert.match(typeSource, /last_query_error\?: string/)
  assert.match(typeSource, /last_query_error_at\?: number/)
  assert.match(typeSource, /last_query_error_source\?: string/)
  assert.match(apiSource, /export function fetchCorporationAccessPolicies\(/)
  assert.match(apiSource, /export function updateCorporationAccessPolicies\(/)
  assert.match(typeSource, /interface CorporationAccessPoliciesConfig \{/)
  assert.match(typeSource, /interface CorporationAccessPolicy \{/)
})

test('sde locale strings are present in zh and en', () => {
  assert.match(zhLocaleSource, /"sdeCheckVersion"\s*:/)
  assert.match(zhLocaleSource, /"sdeRunUpdate"\s*:/)
  assert.match(zhLocaleSource, /"sdeRunUpdateHint"\s*:/)
  assert.match(zhLocaleSource, /"sdeUpdateAvailable"\s*:/)
  assert.match(enLocaleSource, /"sdeCheckVersion"\s*:/)
  assert.match(enLocaleSource, /"sdeRunUpdate"\s*:/)
  assert.match(enLocaleSource, /"sdeRunUpdateHint"\s*:/)
  assert.match(enLocaleSource, /"sdeUpdateAvailable"\s*:/)
  assert.match(zhLocaleSource, /"corporationAccessPolicies"\s*:/)
  assert.match(enLocaleSource, /"corporationAccessPolicies"\s*:/)
  assert.match(zhLocaleSource, /"corpCapabilityGroups"\s*:/)
  assert.match(enLocaleSource, /"corpCapabilityGroups"\s*:/)
  assert.match(zhLocaleSource, /"selectCorporationToConfigure"\s*:/)
  assert.match(enLocaleSource, /"selectCorporationToConfigure"\s*:/)
  assert.match(zhLocaleSource, /"saveCurrentCorporationPolicy"\s*:/)
  assert.match(enLocaleSource, /"saveCurrentCorporationPolicy"\s*:/)
  assert.match(zhLocaleSource, /"savedCorporations"\s*:/)
  assert.match(enLocaleSource, /"savedCorporations"\s*:/)
  assert.match(source, /v-model="selectedCorporationId"/)
  assert.match(source, /formatCorporationDisplay\(corporationID\)/)
  assert.match(source, /saved-corporations-tags/)
  assert.match(source, /saveCurrentCorporationPolicy/)
  assert.doesNotMatch(source, /v-for="policy in corpPolicyRows"/)
  assert.match(source, /corpCapabilityLabelKeys\[capability\]/)
  assert.doesNotMatch(source, /\{\{\s*capability\s*\}\}/)
  assert.match(source, /fetchCorporationAccessPolicies\(/)
  assert.match(source, /updateCorporationAccessPolicies\(/)
  assert.match(source, /srp\.recommendation_multiplier/)
  assert.match(source, /const latestPoliciesConfig = await fetchCorporationAccessPolicies\(\)/)
  assert.match(source, /mergedPolicyMap\.set\(selectedPolicy\.value\.corporation_id/)
  assert.match(source, /policies:\s*Array\.from\(mergedPolicyMap\.values\(\)\)\.sort/)
})
