import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./galaxy-registry.ts', import.meta.url), 'utf8')

test('galaxy registry member APIs keep expected endpoints and methods', () => {
  assert.match(
    source,
    /fetchGalaxyRegistrySystems\(\)[\s\S]*?request\.get[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/systems/
  )
  assert.match(
    source,
    /createGalaxyRegistryEntry\(data:[\s\S]*?request\.post[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/entries/
  )
  assert.match(
    source,
    /endGalaxyRegistryEntry\(id:\s*number\)[\s\S]*?`\/api\/v1\/dashboard\/galaxy-registry\/entries\/\$\{id\}\/end`/
  )
  assert.match(
    source,
    /fetchMyGalaxyRegistryEntries\([\s\S]*?request\.get[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/my-entries/
  )
})

test('galaxy registry admin APIs keep expected endpoints and methods', () => {
  assert.match(
    source,
    /searchGalaxyRegistrySdeSystems\([\s\S]*?request\.get[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/admin\/sde-systems/
  )
  assert.match(
    source,
    /createAdminGalaxyRegistrySystem\([\s\S]*?request\.post[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/admin\/systems/
  )
  assert.match(
    source,
    /updateAdminGalaxyRegistrySystem\(\s*id:\s*number,[\s\S]*?`\/api\/v1\/dashboard\/galaxy-registry\/admin\/systems\/\$\{id\}`/
  )
  assert.match(
    source,
    /deleteAdminGalaxyRegistrySystem\(id:\s*number\)[\s\S]*?`\/api\/v1\/dashboard\/galaxy-registry\/admin\/systems\/\$\{id\}`/
  )
  assert.match(
    source,
    /forceEndAdminGalaxyRegistryEntry\(id:\s*number\)[\s\S]*?`\/api\/v1\/dashboard\/galaxy-registry\/admin\/entries\/\$\{id\}\/force-end`/
  )
  assert.match(
    source,
    /fetchAdminGalaxyRegistryAnalytics\([\s\S]*?request\.get[\s\S]*?\/api\/v1\/dashboard\/galaxy-registry\/admin\/analytics/
  )
})
