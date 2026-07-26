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

test('qq governance preserves card template placeholders in the hint', () => {
  assert.match(source, /t\('qqGovernance\.v2\.templateHint',\s*\{/)
  assert.match(source, /nickname:\s*'\{nickname\}'/)
  assert.match(source, /primary_character_name:\s*'\{primary_character_name\}'/)
  assert.match(source, /primary_corporation_name:\s*'\{primary_corporation_name\}'/)
})

test('qq governance operation buttons use icons with tooltips', () => {
  assert.match(source, /import \{ CircleCheck, Delete, Edit, Refresh, RefreshRight \}/)
  assert.match(source, /:content="t\('qqGovernance\.actions\.edit'\)"/)
  assert.match(source, /:icon="Edit"/)
  assert.match(source, /:icon="Refresh"/)
  assert.match(source, /:icon="Delete"/)
  assert.match(source, /:icon="RefreshRight"/)
  assert.match(source, /:icon="CircleCheck"/)
})

test('qq governance exposes connection recovery and global plus group rate-limit state', () => {
  assert.match(source, /recoverQQGovernanceDisconnectedTasks/)
  assert.match(source, /recoverDisconnectedTasks/)
  assert.match(source, /connection\.rate_limit\.available/)
  assert.match(source, /connection\.rate_limit\.global/)
  assert.match(source, /connection\.rate_limit\.groups/)
  assert.match(source, /rateLimitLabel\(group\.bucket\)/)
  assert.match(zhLocale, /"recoverDisconnected":\s*"恢复连接阻塞任务"/)
  assert.match(enLocale, /"recoverDisconnected":\s*"Recover Connection-blocked Tasks"/)
})

test('qq governance displays the distinct reconciliation trigger outcomes', () => {
  assert.match(source, /reconcileCreated/)
  assert.match(source, /reconcileResumed/)
  assert.match(source, /reconcileRunning/)
  assert.match(source, /reconcileBlocked/)
})

test('qq governance uses shared tables and ledger pagination for growing records', () => {
  assert.match(source, /import \{ useTable \} from '@\/hooks\/core\/useTable'/)
  assert.match(source, /<ArtTable[\s\S]*:columns="policyColumns"/)
  assert.match(source, /<ArtTable[\s\S]*visual-variant="ledger"/)
  assert.match(source, /pageSize: 200/)
  assert.match(source, /paginationKey: \{ current: 'page', size: 'pageSize' \}/)
  assert.match(source, /fetchTaskTableData/)
  assert.match(source, /fetchReviewTableData/)
  assert.match(source, /fetchAlertTableData/)
})

test('qq governance uses the shared table header for rules and group status', () => {
  assert.match(source, /v-model:columns="policyColumns"/)
  assert.match(source, /@refresh="loadRules"/)
  assert.match(source, /v-model:columns="groupColumns"/)
  assert.match(source, /@refresh="loadGroups"/)
  assert.doesNotMatch(source, /class="toolbar"/)
})

test('qq governance gives each operation table a shared header and preserves visible columns', () => {
  assert.match(source, /v-model:columns="taskColumnChecks"/)
  assert.match(source, /v-model:columns="reviewColumnChecks"/)
  assert.match(source, /v-model:columns="alertColumnChecks"/)
  assert.match(source, /:columns="visiblePolicyColumns"/)
  assert.match(source, /:columns="visibleGroupColumns"/)
  assert.match(source, /\.art-table-card \.el-card__body/)
})
