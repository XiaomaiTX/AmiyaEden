import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const zhLocale = readFileSync(new URL('../../../locales/langs/zh.json', import.meta.url), 'utf8')
const enLocale = readFileSync(new URL('../../../locales/langs/en.json', import.meta.url), 'utf8')

test('qq governance localizes enabled status labels', () => {
  assert.match(source, /t\('common\.enable'\)/)
  assert.match(source, /t\('common\.disable'\)/)
  assert.doesNotMatch(source, /t\('common\.enabled'\)/)
  assert.match(zhLocale, /"enable":\s*"启用"/)
  assert.match(zhLocale, /"disable":\s*"停用"/)
  assert.match(enLocale, /"enable":\s*"Enable"/)
  assert.match(enLocale, /"disable":\s*"Disable"/)
})

test('qq governance rules table shows member count and removes the old detail columns', () => {
  assert.match(source, /t\('qqGovernance\.v2\.memberCount'\)/)
  assert.match(source, /groupStatuses\.get\(row\.group_id\)\?\.member_count/)
  assert.doesNotMatch(source, /openCorporationDialog|corporationDialog|corporation-badge-list/)
  assert.doesNotMatch(source, /<ElTableColumn\s+prop="card_template"/)
})

test('qq governance group status table shows the bot administrator status', () => {
  const rulesStart = source.indexOf(
    '<ElTabPane :label="t(\'qqGovernance.v2.rules\')" name="rules">'
  )
  const groupsStart = source.indexOf(
    '<ElTabPane :label="t(\'qqGovernance.v2.groups\')" name="groups">'
  )
  const rulesSection = source.slice(rulesStart, groupsStart)
  const groupsSection = source.slice(groupsStart)
  assert.match(source, /t\('qqGovernance\.fields\.botAdmin'\)/)
  assert.doesNotMatch(rulesSection, /row\.bot_is_admin/)
  assert.match(groupsSection, /row\.bot_is_admin === true/)
  assert.match(groupsSection, /row\.bot_is_admin === false/)
  assert.match(groupsSection, /qqGovernance\.values\.unknown/)
  assert.match(zhLocale, /"botAdmin":\s*"机器人管理员"/)
  assert.match(enLocale, /"botAdmin":\s*"Bot Administrator"/)
})

test('qq governance localizes the group info refresh task', () => {
  assert.match(source, /refresh_group_info/)
  assert.match(source, /qqGovernance\.v2\.groupInfoTask/)
  assert.match(zhLocale, /"groupInfoTask":\s*"群资料刷新"/)
  assert.match(enLocale, /"groupInfoTask":\s*"Group Info Refresh"/)
})
