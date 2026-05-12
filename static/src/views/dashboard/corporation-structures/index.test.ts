import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(
  new URL('../../../api/corporation-structures.ts', import.meta.url),
  'utf8'
)

test('corporation structures page uses single-page tab query routing', () => {
  assert.match(source, /type StructureTab = 'list' \| 'settings' \| 'assignment_salary'/)
  assert.match(source, /const activeTab = ref<StructureTab>\(normalizeTab\(route\.query\.tab\)\)/)
  assert.match(source, /watch\(\s*\(\) => route\.query\.tab,/)
  assert.match(source, /void router\.replace\(\{\s*query:\s*\{\s*\.\.\.route\.query,\s*tab/s)
})

test('corporation structures page wires settings and list tabs', () => {
  assert.match(source, /corporationStructures\.tabs\.list/)
  assert.match(source, /corporationStructures\.tabs\.settings/)
  assert.match(source, /corporationStructures\.tabs\.assignmentSalary/)
  assert.match(source, /saveAuthorizations/)
  assert.match(source, /saveAssignments/)
  assert.match(source, /assignmentTargetCharacterId/)
  assert.match(source, /assignmentFilterMode/)
  assert.match(source, /ArtTableHeader[\s\S]*assignmentColumnChecks/s)
  assert.match(source, /toggleAssignmentToTarget/)
  assert.match(source, /runSalaryPayout/)
  assert.match(source, /handleRunTaskForSelectedCorporation/)
  assert.doesNotMatch(source, /handleRefreshCorporation/)
  assert.doesNotMatch(source, /refreshThisCorporation/)
})

test('corporation structures settings includes notice thresholds and submits them together', () => {
  assert.match(source, /corporationStructures\.settings\.noticeThresholds/)
  assert.match(source, /noticeThresholds\.fuel_notice_threshold_days/)
  assert.match(source, /noticeThresholds\.timer_notice_threshold_days/)
  assert.match(source, /fuel_notice_threshold_days:\s*normalizeThresholdDays/)
  assert.match(source, /timer_notice_threshold_days:\s*normalizeThresholdDays/)
})

test('corporation structures page uses shared timestamp formatter and avoids inline locale formatting', () => {
  assert.match(source, /import\s+\{\s*formatTime\s*\}\s+from\s+'@\/utils\/common'/)
  assert.doesNotMatch(source, /toLocaleString\(/)
  assert.match(source, /formatTime\(new Date\(updatedAt \* 1000\)\.toISOString\(\)\)/)
})

test('corporation structures page avoids any typing in state tag map', () => {
  assert.doesNotMatch(source, /Record<string,\s*any>/)
  assert.match(
    source,
    /type TagType = '' \| 'success' \| 'warning' \| 'info' \| 'primary' \| 'danger'/
  )
})

test('corporation structures reinforce hour formatter keeps 00 as valid hour', () => {
  assert.match(source, /const formatReinforceHour = \(hour: number\) =>/)
  assert.match(source, /hour < 0 \|\| hour > 23/)
  assert.match(source, /return String\(hour\)\.padStart\(2, '0'\)/)
})

test('corporation structures full-height tabs define explicit height chain styles', () => {
  assert.match(source, /<style scoped lang="scss">/)
  assert.match(
    source,
    /:deep\(\.el-card__body\)\s*\{[\s\S]*display:\s*flex;[\s\S]*min-height:\s*0;/s
  )
  assert.match(source, /:deep\(\.el-tabs\)\s*\{[\s\S]*flex:\s*1;[\s\S]*min-height:\s*0;/s)
  assert.match(
    source,
    /:deep\(\.el-tabs__content\)\s*\{[\s\S]*flex:\s*1;[\s\S]*min-height:\s*0;[\s\S]*overflow:\s*hidden;/s
  )
  assert.match(
    source,
    /:deep\(\.el-tab-pane\)\s*\{[\s\S]*height:\s*100%;[\s\S]*min-height:\s*0;[\s\S]*display:\s*flex;/s
  )
  assert.match(source, /class="art-table-card corporation-structures-page__list-card"/)
  assert.match(source, /&__list-card\s*\{[\s\S]*flex:\s*1;[\s\S]*min-height:\s*0;/s)
  assert.match(
    source,
    /\.corporation-structures-page__list-card :deep\(\.el-card__body\)\s*\{[\s\S]*flex:\s*1;[\s\S]*overflow:\s*hidden;/s
  )
})

test('corporation structures settings supports disabling dashboard authorization per corporation', () => {
  assert.match(source, /corporationStructures\.options\.disabled/)
  assert.match(source, /:value="0"/)
  assert.match(source, /character_id:\s*authorizationByCorp\[corp\.corporation_id\]\s*\|\|\s*0/)
})

test('corporation structures list tab shows only valid corporations in selector', () => {
  assert.match(source, /const validCorporations = computed\(\(\) =>/)
  assert.match(source, /authorized_character_id \|\| 0\) > 0/)
  assert.match(source, /v-for="corp in validCorporations"/)
})

test('corporation structures api module exposes all required endpoints', () => {
  assert.match(apiSource, /\/api\/v1\/dashboard\/corporation-structures\/settings/)
  assert.match(apiSource, /\/settings\/authorizations/)
  assert.match(apiSource, /\/corporation-structures\/list/)
  assert.match(apiSource, /\/corporation-structures\/run-task/)
  assert.match(apiSource, /\/corporation-structures\/assignments/)
  assert.match(apiSource, /\/corporation-structures\/fuel-salary-settings/)
  assert.match(apiSource, /\/corporation-structures\/fuel-salary-payouts\/run/)
  assert.match(apiSource, /\/corporation-structures\/my-assigned-list/)
  assert.doesNotMatch(apiSource, /\/corporation-structures\/refresh/)
})
