import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./hall-of-fame.ts', import.meta.url), 'utf8')

test('hall of fame routes expose the renamed root and the current management tab', () => {
  assert.match(source, /title:\s*'menus\.hallOfFame\.title'/)
  assert.match(source, /title:\s*'menus\.hallOfFame\.temple'/)
  assert.match(source, /title:\s*'menus\.hallOfFame\.manage'/)
  assert.match(source, /title:\s*'menus\.hallOfFame\.currentManage'/)
  assert.match(source, /component:\s*'\/hall-of-fame\/current-manage'/)
})

test('current-manage tab has no role restriction so all users and guests can view it', () => {
  // The current-manage route must not have a roles restriction
  // Extract just the current-manage block and verify it contains no roles key
  const currentManageBlock = source.slice(
    source.indexOf("'current-manage'"),
    source.indexOf("'current-manage'") + 400
  )
  assert.doesNotMatch(currentManageBlock, /roles:/)
})
