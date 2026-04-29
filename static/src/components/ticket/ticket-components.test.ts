import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const prioritySource = readFileSync(new URL('./TicketPriorityBadge.vue', import.meta.url), 'utf8')
const statusSource = readFileSync(new URL('./TicketStatusBadge.vue', import.meta.url), 'utf8')
const replySource = readFileSync(new URL('./TicketReplyItem.vue', import.meta.url), 'utf8')

test('TicketPriorityBadge maps high/medium/low priorities to expected tag types', () => {
  assert.match(prioritySource, /if \(props\.priority === 'high'\) return 'danger'/)
  assert.match(prioritySource, /if \(props\.priority === 'medium'\) return 'warning'/)
  assert.match(prioritySource, /return 'info'/)
  assert.match(prioritySource, /t\(`ticket\.priority\.\$\{priority\}`\)/)
})

test('TicketStatusBadge maps pending/in_progress/completed to expected tag types', () => {
  assert.match(statusSource, /if \(props\.status === 'pending'\) return 'warning'/)
  assert.match(statusSource, /if \(props\.status === 'in_progress'\) return 'primary'/)
  assert.match(statusSource, /return 'success'/)
  assert.match(statusSource, /t\(`ticket\.status\.\$\{status\}`\)/)
})

test('TicketReplyItem keeps internal-note marker and content rendering', () => {
  assert.match(replySource, /v-if="reply\.is_internal"/)
  assert.match(replySource, /t\('ticket\.internalNote'\)/)
  assert.match(replySource, /ticket-reply-item__content/)
  assert.match(replySource, /\{\{ reply\.content \}\}/)
})
