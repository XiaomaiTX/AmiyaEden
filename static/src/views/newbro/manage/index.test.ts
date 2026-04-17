import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const managePageSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const newbroApiSource = readFileSync(new URL('../../../api/newbro.ts', import.meta.url), 'utf8')
const routerSource = readFileSync(
  new URL('../../../router/modules/newbro.ts', import.meta.url),
  'utf8'
)
const systemRouterSource = readFileSync(
  new URL('../../../router/modules/system.ts', import.meta.url),
  'utf8'
)
const zhLocaleSource = readFileSync(
  new URL('../../../locales/langs/zh.json', import.meta.url),
  'utf8'
)
const enLocaleSource = readFileSync(
  new URL('../../../locales/langs/en.json', import.meta.url),
  'utf8'
)

test('newbro manage page no longer exposes manual sync or reward triggers', () => {
  assert.doesNotMatch(managePageSource, /fetchRunCaptainAttributionSync/)
  assert.doesNotMatch(managePageSource, /fetchRunCaptainRewardProcessing/)
  assert.doesNotMatch(managePageSource, /newbro\.manage\.runSync/)
  assert.doesNotMatch(managePageSource, /newbro\.manage\.runRewardProcessing/)
  assert.doesNotMatch(managePageSource, /const syncing = ref\(/)
  assert.doesNotMatch(managePageSource, /const processingRewards = ref\(/)

  assert.match(managePageSource, /newbro\.manage\.performanceTab/)
  assert.match(managePageSource, /newbro\.manage\.rewardHistoryTab/)
  assert.match(managePageSource, /newbro\.manage\.affiliationHistoryTab/)

  assert.doesNotMatch(newbroApiSource, /export function fetchRunCaptainAttributionSync\(/)
  assert.doesNotMatch(newbroApiSource, /export function fetchRunCaptainRewardProcessing\(/)
  assert.doesNotMatch(newbroApiSource, /\/api\/v1\/system\/newbro\/attribution\/sync/)
  assert.doesNotMatch(newbroApiSource, /\/api\/v1\/system\/newbro\/reward\/process/)
})

test('newbro manage page does not render a standalone title card', () => {
  assert.doesNotMatch(managePageSource, /newbro\.manage\.title/)
  assert.doesNotMatch(managePageSource, /newbro\.manage\.subtitle/)
  assert.match(managePageSource, /<ElTabs v-model="activeTab">/)
})

test('newbro manage route is available to captains and the page keeps captains on readonly tabs', () => {
  assert.match(routerSource, /name:\s*'NewbroManage'/)
  assert.match(routerSource, /roles:\s*\['super_admin', 'admin', 'captain'\]/)

  assert.match(managePageSource, /import \{ useUserStore \} from '@\/store\/modules\/user'/)
  assert.match(managePageSource, /const userStore = useUserStore\(\)/)
  assert.match(managePageSource, /const isCaptainReadonly = computed\(/)
  assert.match(managePageSource, /const captainReadonlyDefaultTab = 'rewards'/)
  assert.match(
    managePageSource,
    /\?\s*\['performance', 'rewards', 'history', 'settings'\]\s*:\s*\[captainReadonlyDefaultTab, 'history'\]/
  )
  assert.match(managePageSource, /if \(!managePageTabs\.value\.includes\(value\)\)/)
})

test('newbro manage page exposes an admin support settings tab and removes the old system route', () => {
  assert.match(managePageSource, /newbro\.manage\.settingsTab/)
  assert.match(managePageSource, /v-if="managePageTabs\.includes\('settings'\)"/)
  assert.match(managePageSource, /name="settings"[\s\S]*lazy/)
  assert.match(newbroApiSource, /export function fetchAdminNewbroSupportSettings\(/)
  assert.match(newbroApiSource, /export function updateAdminNewbroSupportSettings\(/)
  assert.doesNotMatch(newbroApiSource, /export function fetchAdminNewbroSettings\(/)
  assert.doesNotMatch(newbroApiSource, /export function updateAdminNewbroSettings\(/)

  assert.doesNotMatch(systemRouterSource, /path:\s*'newbro-settings'/)
  assert.doesNotMatch(systemRouterSource, /name:\s*'NewbroSettings'/)
  assert.doesNotMatch(systemRouterSource, /menus\.system\.newbroSettings/)
})

test('newbro manage page uses shared readonly record APIs for captain access', () => {
  assert.match(managePageSource, /fetchAdminRewardSettlements/)
  assert.match(managePageSource, /fetchAdminAffiliationHistory/)
  assert.match(managePageSource, /await fetchAdminAffiliationHistory\(requestParams\)/)
  assert.match(managePageSource, /await fetchAdminRewardSettlements\(requestParams\)/)
  assert.doesNotMatch(managePageSource, /fetchCaptainAffiliationHistory/)
  assert.doesNotMatch(newbroApiSource, /export function fetchCaptainAffiliationHistory\(/)
})

test('newbro manage labels use 队长管理 naming', () => {
  assert.match(zhLocaleSource, /"manage"\s*:\s*"队长管理"/)
  assert.match(zhLocaleSource, /"settingsTab"\s*:\s*"帮扶设置"/)

  assert.match(enLocaleSource, /"manage"\s*:\s*"Captain Management"/)
  assert.match(enLocaleSource, /"settingsTab"\s*:\s*"Support Settings"/)
})
