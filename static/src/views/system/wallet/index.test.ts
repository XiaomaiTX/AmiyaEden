import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('analysis tab uses dedicated scroll container to avoid clipped content', () => {
  assert.match(source, /<div class="analysis-tab-pane">\s*<WalletAnalysis \/>/)
  assert.match(source, /\.analysis-tab-pane\s*\{[\s\S]*overflow-y:\s*auto/)
})
