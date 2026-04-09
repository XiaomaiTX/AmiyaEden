import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('skill page renders ESI refresh button alongside database refresh button', () => {
  assert.match(source, /<ElButton[\s\S]*:loading="loading"[\s\S]*@click="loadData"/)
  assert.match(source, /<ElButton[\s\S]*:loading="esiRefreshing"[\s\S]*ESI 拉取/)
  assert.match(source, /const esiRefreshing = ref\(false\)/)
})

test('ESI refresh button calls runMyCharacterESIRefresh with correct parameters', () => {
  assert.match(source, /const onESIRefreshClick = async \(\) =>/)
  assert.match(source, /await runMyCharacterESIRefresh\({/)
  assert.match(source, /task_name: 'character_skill'/)
  assert.match(source, /character_id: selectedCharacterId\.value/)
})

test('ESI refresh button shows confirmation dialog before submission', () => {
  assert.match(source, /await ElMessageBox\.confirm\(/)
  assert.match(source, /确认从 ESI 拉取角色/)
  assert.match(source, /confirmButtonText: '确认拉取'/)
  assert.match(source, /cancelButtonText: '取消'/)
  assert.match(source, /type: 'info'/)
})

test('ESI refresh button differentiates permission errors from other errors', () => {
  assert.match(source, /if \(msg\.includes\('无权'\) \|\| e\?\.response\?\.status === 403\)/)
  assert.match(source, /ElMessage\.error\('无权操作此角色'\)/)
  assert.match(source, /else if \(msg\.includes\('角色不存在'\)\)/)
  assert.match(source, /ElMessage\.error\('角色未找到'\)/)
})

test('ESI refresh button displays loading state during submission', () => {
  assert.match(source, /esiRefreshing\.value = true/)
  assert.match(source, /finally \{[\s\S]*esiRefreshing\.value = false[\s\S]*}/)
})

test('ESI refresh success message instructs user to refresh page', () => {
  assert.match(
    source,
    /ElMessage\.success\('技能数据 ESI 刷新任务已提交，稍后可点击刷新按钮查看最新数据'\)/
  )
})
