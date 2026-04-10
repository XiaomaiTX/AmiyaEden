import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const managePageSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const newbroApiSource = readFileSync(new URL('../../../api/newbro.ts', import.meta.url), 'utf8')

test('newbro manage page no longer exposes manual sync or reward triggers', () => {
	assert.doesNotMatch(managePageSource, /fetchRunCaptainAttributionSync/)
	assert.doesNotMatch(managePageSource, /fetchRunCaptainRewardProcessing/)
	assert.doesNotMatch(managePageSource, /newbro\.manage\.runSync/)
	assert.doesNotMatch(managePageSource, /newbro\.manage\.runRewardProcessing/)
	assert.doesNotMatch(managePageSource, /const syncing = ref\(/)
	assert.doesNotMatch(managePageSource, /const processingRewards = ref\(/)

	assert.match(managePageSource, /newbro\.manage\.performanceTab/)
	assert.match(managePageSource, /newbro\.manage\.rewardHistoryTab/)
	assert.match(managePageSource, /newbro\.manage\.affiliationHistoryTab/)

	assert.doesNotMatch(newbroApiSource, /export function fetchRunCaptainAttributionSync\(/)
	assert.doesNotMatch(newbroApiSource, /export function fetchRunCaptainRewardProcessing\(/)
	assert.doesNotMatch(newbroApiSource, /\/api\/v1\/system\/newbro\/attribution\/sync/)
	assert.doesNotMatch(newbroApiSource, /\/api\/v1\/system\/newbro\/reward\/process/)
})