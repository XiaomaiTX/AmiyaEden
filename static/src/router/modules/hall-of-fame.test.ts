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

test('hall of fame root keeps login gating and current-manage stays login-only', () => {
  const rootBlock = source.slice(
    source.indexOf("name: 'HallOfFameRoot'"),
    source.indexOf('children: [')
  )
  assert.match(rootBlock, /login:\s*true/)

  const currentManageBlock = source.slice(
    source.indexOf("'current-manage'"),
    source.indexOf("'current-manage'") + 400
  )
  assert.match(currentManageBlock, /login:\s*true/)
  assert.doesNotMatch(currentManageBlock, /roles:/)
})

test('manage tab still requires admin roles', () => {
  const manageBlock = source.slice(source.indexOf("'manage'"), source.indexOf("'manage'") + 400)
  assert.match(manageBlock, /roles:\s*\[\s*'super_admin',\s*'admin'\s*\]/)
})
