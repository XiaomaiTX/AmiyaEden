import { readFileSync } from 'node:fs'
import assert from 'node:assert/strict'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('fleet PAP input enforces 0.5 step in UI', () => {
  assert.match(source, /v-model="formData\.pap_count"/)
  assert.match(source, /:step="0\.5"/)
  assert.match(source, /step-strictly/)
})

test('fleet PAP form rule validates half-step values before submit', () => {
  assert.match(source, /const isHalfStepPap = \(value: number\) =>/)
  assert.match(source, /Math\.abs\(value \* 2 - Math\.round\(value \* 2\)\) < 1e-9/)
  assert.match(source, /callback\(new Error\('PAP 数量必须大于 0 且按 0\.5 粒度'\)\)/)
})
