import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./index.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('hall of fame temple page exists and loads temple data into a canvas component', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /import \{ fetchTemple \} from '@\/api\/hall-of-fame'/)
  assert.match(source, /import TempleCanvas from '\.\/modules\/temple-canvas\.vue'/)
  assert.match(source, /hallOfFame\.temple\.emptyTitle/)
  assert.match(source, /hallOfFame\.temple\.emptySubtitle/)
  assert.match(source, /onMounted\(/)
})
