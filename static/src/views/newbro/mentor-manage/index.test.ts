import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../../api/mentor.ts', import.meta.url), 'utf8')
const typeSource = readFileSync(new URL('../../../types/api/api.d.ts', import.meta.url), 'utf8')
const newbroRouterSource = readFileSync(
  new URL('../../../router/modules/newbro.ts', import.meta.url),
  'utf8'
)
const systemRouterSource = readFileSync(
  new URL('../../../router/modules/system.ts', import.meta.url),
  'utf8'
)
const docSource = readFileSync(
  new URL('../../../../../docs/features/current/mentor-system.md', import.meta.url),
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

test('pending mentor applications use cancel-specific admin copy', () => {
  assert.match(source, /function revokeActionLabel/)
  assert.match(source, /function revokeActionSuccessMessage/)
  assert.match(source, /status === 'pending'/)
  assert.match(source, /newbro\.mentorManage\.cancelPending/)
  assert.match(source, /newbro\.mentorManage\.cancelPendingSuccess/)
  assert.match(source, /newbro\.mentorManage\.revoke/)
  assert.match(source, /newbro\.mentorManage\.revokeSuccess/)
})

test('mentor manage page does not render a standalone title card', () => {
  assert.doesNotMatch(source, /newbro\.mentorManage\.title/)
  assert.doesNotMatch(source, /newbro\.mentorManage\.subtitle/)
  assert.match(source, /common\.refresh/)
})

test('mentor manage page includes a reward distribution records tab with ledger pagination', () => {
  assert.match(source, /<ElTabs v-model="activeTab"/)
  assert.match(source, /newbro\.mentorManage\.relationshipsTab/)
  assert.match(source, /newbro\.mentorManage\.rewardRecordsTab/)
  assert.match(source, /newbro\.mentorManage\.rewardStagesTab/)
  assert.match(source, /fetchAdminMentorRewardDistributions/)
  assert.match(source, /visual-variant="ledger"/)
  assert.match(
    source,
    /rewardHistoryPaginationOptions = \{\s*pageSizes: \[50, 100, 200, 500, 1000\]/
  )
})

test('mentor manage page hosts reward stage settings and removes the standalone system route', () => {
  assert.match(source, /fetchMentorSettings/)
  assert.match(source, /updateMentorSettings/)
  assert.match(source, /fetchMentorRewardStages/)
  assert.match(source, /updateMentorRewardStages/)
  assert.match(source, /runMentorRewardProcessing/)
  assert.match(source, /newbro\.mentorManage\.rewardStagesLoadFailed/)
  assert.match(source, /newbro\.mentorManage\.rewardStagesNotReady/)
  assert.match(source, /v-if="rewardStagesLoadFailed"/)
  assert.match(source, /system\.mentorRewardStages\.eligibilityTitle/)
  assert.match(source, /system\.mentorRewardStages\.maxCharacterSP/)
  assert.match(source, /system\.mentorRewardStages\.maxAccountAgeDays/)
  assert.match(source, /system\.mentorRewardStages\.runProcess/)
  assert.doesNotMatch(source, /<template>\s*<div class="[^"]*\bart-full-height\b[^"]*">/)

  assert.match(newbroRouterSource, /path: 'mentor-manage'[\s\S]*roles: \['super_admin', 'admin'\]/)
  assert.doesNotMatch(systemRouterSource, /path: 'mentor-reward-stages'/)

  assert.match(
    docSource,
    /管理员可在 `导师管理` 页面查看全部导师关系；对 `pending` 状态可取消学员申请，对 `active` 状态可撤销导师关系/
  )
  assert.match(
    docSource,
    /管理员可在 `导师管理` 页面(?:的)? `设置奖励阶段` tab 配置阶段化奖励规则、学员资格阈值，并手动执行一次奖励处理/
  )

  assert.match(zhLocaleSource, /"rewardStagesTab"\s*:\s*"设置奖励阶段"/)
  assert.match(zhLocaleSource, /"rewardStagesLoadFailed"\s*:/)
  assert.match(zhLocaleSource, /"rewardStagesNotReady"\s*:/)
  assert.match(enLocaleSource, /"rewardStagesTab"\s*:/)
  assert.match(enLocaleSource, /"rewardStagesLoadFailed"\s*:/)
  assert.match(enLocaleSource, /"rewardStagesNotReady"\s*:/)
})

test('mentor manage reward stage and eligibility number inputs stay integer-only', () => {
  const inputNumbers = source.match(/<ElInputNumber[\s\S]*?\/>/g) ?? []
  const rewardStageInputs = inputNumbers.filter((inputNumber) =>
    /row\.stage_order|row\.threshold|row\.reward_amount|mentorSettings\.max_character_sp|mentorSettings\.max_account_age_days/.test(
      inputNumber
    )
  )

  assert.equal(rewardStageInputs.length, 5)

  for (const inputNumber of rewardStageInputs) {
    assert.match(inputNumber, /:controls="false"/)
    assert.match(inputNumber, /step-strictly/)
    assert.doesNotMatch(inputNumber, /:precision=/)
    assert.doesNotMatch(inputNumber, /0\.01/)
  }

  assert.match(
    source,
    /v-model="mentorSettings\.max_character_sp"[\s\S]*?:step="1000000"[\s\S]*?step-strictly/
  )
  assert.match(
    source,
    /v-model="mentorSettings\.max_account_age_days"[\s\S]*?:step="1"[\s\S]*?step-strictly/
  )
})

test('mentor manage reward records support mentor character and nickname filtering across contract and docs', () => {
  assert.match(source, /rewardHistoryKeyword/)
  assert.match(source, /newbro\.mentorManage\.rewardKeyword/)
  assert.match(source, /mentor_character_name/)
  assert.match(source, /mentor_nickname/)

  assert.match(apiSource, /export function fetchAdminMentorRewardDistributions\(/)
  assert.match(typeSource, /interface RewardDistributionView\s*\{/)
  assert.match(typeSource, /mentor_character_name: string/)
  assert.match(typeSource, /mentor_nickname: string/)
  assert.match(typeSource, /type AdminRewardDistributionsParams = Partial<\{/)
  assert.match(
    typeSource,
    /type AdminRewardDistributionsResponse = Api\.Common\.PaginatedResponse<RewardDistributionView>/
  )

  assert.match(docSource, /GET \/api\/v1\/system\/mentor\/reward-distributions/)
  assert.match(
    docSource,
    /奖励发放记录 tab：按 ledger 方式分页显示导师奖励发放记录，并支持按导师人物名或昵称搜索/
  )

  assert.match(zhLocaleSource, /"rewardRecordsTab"\s*:/)
  assert.match(zhLocaleSource, /"rewardKeyword"\s*:/)
  assert.match(enLocaleSource, /"rewardRecordsTab"\s*:/)
  assert.match(enLocaleSource, /"rewardKeyword"\s*:/)
})
