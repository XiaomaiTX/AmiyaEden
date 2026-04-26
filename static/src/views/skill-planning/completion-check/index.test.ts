import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('completion check renders prefixed plan labels in the result list', () => {
  assert.match(source, /formatCompletionPlanLabel\(plan\)/)
  assert.match(source, /planLabelMap/)
  assert.match(source, /skillPlanCheck\.planOptionLabel/)
  assert.match(source, /skillPlan\.scope\.personal/)
  assert.match(source, /skillPlan\.scope\.corp/)
})

test('completion check exposes one-click copy for missing skills and copies required levels', () => {
  const missingSkillsBlock = source.match(
    /<div v-if="plan\.missing_skills\.length" class="missing-skills">[\s\S]*?<\/div>\s*<\/div>\s*<\/ElCollapseItem>/
  )

  assert.ok(missingSkillsBlock, 'expected missing skills block')
  assert.match(source, /skillPlanCheck\.copyMissingSkills/)
  assert.match(source, /copyMissingSkills\(plan\)/)
  assert.match(source, /`\$\{skill\.skill_name\} \$\{skill\.required_level\}`/)
  assert.doesNotMatch(source, /ArtCopyButton/)
})
