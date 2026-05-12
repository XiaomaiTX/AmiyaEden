import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./fuxi-hall.ts', import.meta.url), 'utf8')

test('fuxi hall routes expose leadership, contributors, and manage tabs', () => {
  assert.match(source, /title:\s*'menus\.fuxiHall\.title'/)
  assert.match(source, /title:\s*'menus\.fuxiHall\.leadership'/)
  assert.match(source, /title:\s*'menus\.fuxiHall\.contributors'/)
  assert.match(source, /title:\s*'menus\.fuxiHall\.manage'/)
})

test('fuxi hall root keeps login gating and manage tab requires admin roles', () => {
  const rootBlock = source.slice(
    source.indexOf("name: 'FuxiHallRoot'"),
    source.indexOf('children: [')
  )
  assert.match(rootBlock, /login:\s*true/)
  assert.match(rootBlock, /isHide:\s*true/)

  const manageBlock = source.slice(source.indexOf("'manage'"), source.indexOf("'manage'") + 300)
  assert.match(manageBlock, /roles:\s*\[\s*'super_admin',\s*'admin'\s*\]/)
})
