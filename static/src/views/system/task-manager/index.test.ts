import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const indexSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const tasksTabSource = readFileSync(new URL('./modules/TasksTab.vue', import.meta.url), 'utf8')
const esiControlsSource = readFileSync(
  new URL('./modules/EsiControls.vue', import.meta.url),
  'utf8'
)
const esiStatusesTabSource = readFileSync(
  new URL('./modules/EsiStatusesTab.vue', import.meta.url),
  'utf8'
)
const historyTabSource = readFileSync(new URL('./modules/HistoryTab.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../../api/task-manager.ts', import.meta.url), 'utf8')
const esiApiSource = readFileSync(new URL('../../../api/esi-refresh.ts', import.meta.url), 'utf8')
const routerSource = readFileSync(
  new URL('../../../router/modules/system.ts', import.meta.url),
  'utf8'
)
const typeSource = readFileSync(new URL('../../../types/api/api.d.ts', import.meta.url), 'utf8')
const zhLocaleSource = readFileSync(
  new URL('../../../locales/langs/zh.json', import.meta.url),
  'utf8'
)
const enLocaleSource = readFileSync(
  new URL('../../../locales/langs/en.json', import.meta.url),
  'utf8'
)
const legacyPageExists = existsSync(new URL('../esi-refresh/index.vue', import.meta.url))

test('task manager page uses tabs with extracted task and history modules', () => {
  assert.match(indexSource, /<ElTabs v-model="activeTab"/)
  assert.match(indexSource, /<TasksTab v-if="activeTab === 'tasks'" \/>/)
  assert.match(indexSource, /<EsiStatusesTab v-if="activeTab === 'esi-statuses'" \/>/)
  assert.match(indexSource, /<HistoryTab v-if="activeTab === 'history'" \/>/)
  assert.match(indexSource, /defineOptions\(\{ name: 'TaskManager' \}\)/)
  assert.doesNotMatch(indexSource, /art-full-height/)

  assert.match(tasksTabSource, /fetchTasks\(/)
  assert.match(tasksTabSource, /runTask\(/)
  assert.match(tasksTabSource, /<ElDialog/)
  assert.match(tasksTabSource, /<EsiControls \/>/)

  assert.match(esiControlsSource, /fetchESIRefreshTasks\(/)
  assert.match(esiControlsSource, /runESIRefreshTaskByName\(/)
  assert.match(esiControlsSource, /runESIRefreshAll\(/)
  assert.match(esiControlsSource, /taskManager\.esi\.sections\.tasks/)
  assert.doesNotMatch(esiControlsSource, /taskManager\.esi\.sections\.statuses/)
  assert.doesNotMatch(esiControlsSource, /fetchESIRefreshStatuses/)

  assert.match(esiStatusesTabSource, /fetchESIRefreshStatuses/)
  assert.match(esiStatusesTabSource, /runESIRefreshTask\(/)
  assert.match(esiStatusesTabSource, /taskManager\.esi\.sections\.statuses/)
  assert.match(esiStatusesTabSource, /taskManager\.esi\.filters\.taskName/)
  assert.match(esiStatusesTabSource, /taskManager\.esi\.status\.running/)

  assert.match(historyTabSource, /apiFn:\s*fetchTaskHistory/)
  assert.match(historyTabSource, /visual-variant="ledger"/)
  assert.match(historyTabSource, /apiParams:\s*\{\s*current:\s*1,\s*size:\s*200\s*\}/)
  assert.match(historyTabSource, /taskManager\.columns\.triggeredBy/)
})

test('task manager route and API wrappers use unified task endpoints', () => {
  assert.match(routerSource, /path:\s*'task-manager'/)
  assert.match(routerSource, /name:\s*'TaskManager'/)
  assert.match(routerSource, /component:\s*'\/system\/task-manager'/)
  assert.match(routerSource, /title:\s*'menus\.system\.taskManager'/)
  assert.match(routerSource, /authMark:\s*'execute_task'/)
  assert.match(routerSource, /authMark:\s*'update_schedule'/)
  assert.doesNotMatch(routerSource, /path:\s*'esi-refresh'/)

  assert.match(apiSource, /url:\s*'\/api\/v1\/tasks'/)
  assert.match(apiSource, /url:\s*'\/api\/v1\/tasks\/history'/)
  assert.match(apiSource, /url:\s*`\/api\/v1\/tasks\/\$\{name\}\/run`/)
  assert.match(apiSource, /url:\s*`\/api\/v1\/tasks\/\$\{name\}\/schedule`/)

  assert.match(esiApiSource, /fetchESIRefreshTasks/)
  assert.match(esiApiSource, /url:\s*'\/api\/v1\/tasks\/esi\/tasks'/)
  assert.match(esiApiSource, /url:\s*'\/api\/v1\/tasks\/esi\/statuses'/)
  assert.match(esiApiSource, /url:\s*'\/api\/v1\/tasks\/esi\/run'/)
  assert.match(esiApiSource, /url:\s*'\/api\/v1\/tasks\/esi\/run-task'/)
  assert.match(esiApiSource, /url:\s*'\/api\/v1\/tasks\/esi\/run-all'/)
  assert.doesNotMatch(esiApiSource, /\/api\/v1\/esi\/refresh\//)
})

test('task manager types and locales exist and the legacy page is removed', () => {
  assert.match(typeSource, /namespace TaskManager\s*\{/)
  assert.match(typeSource, /interface TaskItem\s*\{/)
  assert.match(typeSource, /interface TaskExecutionItem\s*\{/)
  assert.match(typeSource, /interface UpdateScheduleParams\s*\{/)
  assert.match(
    typeSource,
    /type TaskHistoryList = Api\.Common\.PaginatedResponse<TaskExecutionItem>/
  )

  assert.match(zhLocaleSource, /"taskManager"\s*:/)
  assert.match(zhLocaleSource, /"tabs"\s*:\s*\{[\s\S]*"tasks"\s*:/)
  assert.match(zhLocaleSource, /"esiStatuses"\s*:/)
  assert.match(zhLocaleSource, /"menus"\s*:\s*\{[\s\S]*"taskManager"\s*:/)
  assert.match(enLocaleSource, /"taskManager"\s*:/)
  assert.match(enLocaleSource, /"esiStatuses"\s*:/)
  assert.match(enLocaleSource, /"menus"\s*:\s*\{[\s\S]*"taskManager"\s*:/)

  assert.equal(legacyPageExists, false)
})
