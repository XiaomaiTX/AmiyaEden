import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8')
const errorSource = readFileSync(new URL('./error.ts', import.meta.url), 'utf8')
const zhLocaleSource = readFileSync(new URL('../../locales/langs/zh.json', import.meta.url), 'utf8')
const enLocaleSource = readFileSync(new URL('../../locales/langs/en.json', import.meta.url), 'utf8')

test('HTTP error handling recognizes conflict responses explicitly', () => {
	assert.match(statusSource, /conflict = 409/)
	assert.match(errorSource, /\[ApiStatus\.conflict\]: 'httpMsg\.conflict'/)
	assert.match(zhLocaleSource, /"conflict"\s*:/)
	assert.match(enLocaleSource, /"conflict"\s*:/)
})