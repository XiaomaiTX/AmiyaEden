import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./dashboard.ts', import.meta.url), 'utf8')

test('dashboard routes include galaxy registry under dashboard menu', () => {
  const block = source.slice(
    source.indexOf("path: 'galaxy-registry'"),
    source.indexOf('    }\n  ]')
  )

  assert.match(block, /name:\s*'DashboardGalaxyRegistry'/)
  assert.match(block, /component:\s*'\/dashboard\/galaxy-registry'/)
  assert.match(block, /title:\s*'menus\.dashboard\.galaxyRegistry'/)
  assert.match(block, /roles:\s*\['super_admin', 'admin', 'captain', 'user'\]/)
})
