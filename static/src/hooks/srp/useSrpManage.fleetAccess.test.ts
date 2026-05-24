import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./useSrpManage.ts', import.meta.url), 'utf8')

test('srp manage fleet filter uses role-based data source split to avoid false 403', () => {
  assert.match(
    source,
    /import\s+\{\s*fetchApplicationList,\s*fetchSrpFleetOptions\s*\}\s+from\s+'@\/api\/srp'/
  )
  assert.match(source, /const fleets = ref<Api\.Srp\.FleetOption\[]>\(\[\]\)/)
  assert.match(source, /fleets\.value = \(await fetchSrpFleetOptions\(\)\) \?\? \[\]/)
  assert.doesNotMatch(source, /fetchFleetList\(/)
  assert.doesNotMatch(source, /fetchMyFleetList\(/)
})
