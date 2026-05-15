import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('skill page renders ESI refresh button alongside database refresh button', () => {
  assert.match(source, /<ElButton[\s\S]*:loading="loading"[\s\S]*@click="loadData"/)
  assert.match(source, /<ElButton[\s\S]*:loading="esiRefreshing"[\s\S]*info\.esiRefreshButton/)
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
  assert.match(source, /info\.skillESIRefreshConfirm/)
  assert.match(source, /confirmButtonText: t\('info\.esiRefreshConfirmButton'\)/)
  assert.match(source, /cancelButtonText: t\('common\.cancel'\)/)
  assert.match(source, /type: 'info'/)
})

test('ESI refresh button differentiates permission errors from other errors', () => {
  assert.match(source, /if \(msg\.includes\('无权'\) \|\| error\.response\?\.status === 403\)/)
  assert.match(source, /ElMessage\.error\(t\('info\.esiRefreshUnauthorized'\)\)/)
  assert.match(source, /else if \(msg\.includes\('角色不存在'\)\)/)
  assert.match(source, /ElMessage\.error\(t\('info\.esiRefreshCharacterNotFound'\)\)/)
})

test('ESI refresh button displays loading state during submission', () => {
  assert.match(source, /esiRefreshing\.value = true/)
  assert.match(source, /finally \{[\s\S]*esiRefreshing\.value = false[\s\S]*}/)
})

test('ESI refresh success message instructs user to refresh page', () => {
  assert.match(source, /ElMessage\.success\(t\('info\.skillESIRefreshSubmitted'\)\)/)
})
