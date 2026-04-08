import assert from 'node:assert/strict'
import test from 'node:test'

import { formatWelfareIneligibleReason, type WelfareReasonMessages } from './ineligibleReason'

const messages: WelfareReasonMessages = {
  pap: '军团PAP数不足',
  skill: '技能未达标',
  legionYears: '伏羲军团服役年限不足',
  skillPlan: (plans) => `技能规划${plans}未达成`,
  planSeparator: '或',
  reasonSeparator: '，'
}

test('formatWelfareIneligibleReason falls back to the existing skill message when no plan names exist', () => {
  assert.equal(formatWelfareIneligibleReason('skill', [], messages), '技能未达标')
})

test('formatWelfareIneligibleReason joins multiple skill plans with the localized separator', () => {
  assert.equal(
    formatWelfareIneligibleReason('skill', ['护盾方案', '装甲方案'], messages),
    '技能规划护盾方案或装甲方案未达成'
  )
})

test('formatWelfareIneligibleReason preserves the PAP warning when both checks fail', () => {
  assert.equal(
    formatWelfareIneligibleReason('pap_skill', ['护盾方案', '装甲方案'], messages),
    '技能规划护盾方案或装甲方案未达成，军团PAP数不足'
  )
})

test('formatWelfareIneligibleReason keeps the Fuxi Legion tenure warning when only tenure blocks the welfare', () => {
  assert.equal(formatWelfareIneligibleReason('legion_years', [], messages), '伏羲军团服役年限不足')
})

test('formatWelfareIneligibleReason combines skill plan and Fuxi Legion tenure warnings', () => {
  assert.equal(
    formatWelfareIneligibleReason('skill_legion_years', ['护盾方案', '装甲方案'], messages),
    '技能规划护盾方案或装甲方案未达成，伏羲军团服役年限不足'
  )
})

test('formatWelfareIneligibleReason composes all active reasons without dedicated combo templates', () => {
  assert.equal(
    formatWelfareIneligibleReason('pap_skill_legion_years', ['护盾方案', '装甲方案'], messages),
    '技能规划护盾方案或装甲方案未达成，军团PAP数不足，伏羲军团服役年限不足'
  )
})
