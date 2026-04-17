import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./admin-settings-panel.vue', import.meta.url), 'utf8')

test('admin settings panel refreshes latest settings before save and only overwrites the active mode fields', () => {
  assert.match(source, /fetchAdminNewbroSupportSettings/)
  assert.match(source, /updateAdminNewbroSupportSettings/)
  assert.match(source, /fetchAdminNewbroRecruitSettings/)
  assert.match(source, /updateAdminNewbroRecruitSettings/)
  assert.match(
    source,
    /if \(isSupportMode\.value\) \{\s*const data = await fetchAdminNewbroSupportSettings\(\)/
  )
  assert.match(
    source,
    /const data = await updateAdminNewbroSupportSettings\(\{[\s\S]*max_character_sp: form\.max_character_sp/
  )
  assert.match(
    source,
    /const data = await updateAdminNewbroSupportSettings\(\{[\s\S]*bonus_rate: form\.bonus_rate/
  )
  assert.match(source, /else \{\s*const data = await fetchAdminNewbroRecruitSettings\(\)/)
  assert.match(
    source,
    /const data = await updateAdminNewbroRecruitSettings\(\{[\s\S]*recruit_cooldown_days: form\.recruit_cooldown_days/
  )
  assert.doesNotMatch(source, /fetchAdminNewbroSettings\(/)
  assert.doesNotMatch(source, /updateAdminNewbroSettings\(/)
  assert.match(source, /defineExpose\(\{\s*reloadSettings: loadSettings\s*\}\)/)
  assert.match(source, /onMounted\(\(\) => \{\s*void loadSettings\(\)/)
})
