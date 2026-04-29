import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const myTicketsSource = readFileSync(new URL('./my-tickets/index.vue', import.meta.url), 'utf8')
const createSource = readFileSync(new URL('./create/index.vue', import.meta.url), 'utf8')
const detailSource = readFileSync(new URL('./detail/index.vue', import.meta.url), 'utf8')

test('my tickets page uses member ticket list API with status filter and detail/create navigation', () => {
  assert.match(myTicketsSource, /import \{ listMyTickets \} from '@\/api\/ticket'/)
  assert.match(
    myTicketsSource,
    /listMyTickets\(\{ current: 1, size: 50, status: filters\.value\.status \}\)/
  )
  assert.match(myTicketsSource, /router\.push\(\{ name: 'TicketCreate' \}\)/)
  assert.match(
    myTicketsSource,
    /router\.push\(\{ name: 'TicketDetail', params: \{ id: String\(id\) \} \}\)/
  )
  assert.match(myTicketsSource, /TicketStatusBadge/)
  assert.match(myTicketsSource, /TicketPriorityBadge/)
})

test('ticket create page loads categories and submits through createTicket API', () => {
  assert.match(
    createSource,
    /import \{ createTicket, listTicketCategories \} from '@\/api\/ticket'/
  )
  assert.match(createSource, /categories\.value = await listTicketCategories\(\)/)
  assert.match(createSource, /await createTicket\(form\)/)
  assert.match(createSource, /priority:\s*'medium'/)
  assert.match(createSource, /router\.push\(\{ name: 'TicketMyList' \}\)/)
})

test('ticket detail page loads ticket and replies, then posts member replies', () => {
  assert.match(
    detailSource,
    /import \{ addMyTicketReply, getMyTicket, listMyTicketReplies \} from '@\/api\/ticket'/
  )
  assert.match(
    detailSource,
    /Promise\.all\(\[\s*getMyTicket\(ticketId\.value\),\s*listMyTicketReplies\(ticketId\.value\)\s*\]\)/
  )
  assert.match(
    detailSource,
    /await addMyTicketReply\(ticketId\.value, \{ content: content\.value \}\)/
  )
  assert.match(detailSource, /TicketReplyItem/)
})
