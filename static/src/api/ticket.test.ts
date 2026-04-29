import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./ticket.ts', import.meta.url), 'utf8')

test('ticket member APIs keep expected endpoints and methods', () => {
  assert.match(
    source,
    /createTicket\(data:[\s\S]*?request\.post[\s\S]*?url:\s*'\/api\/v1\/ticket\/tickets'/
  )
  assert.match(
    source,
    /listMyTickets\([\s\S]*?request\.get[\s\S]*?url:\s*'\/api\/v1\/ticket\/tickets\/me'/
  )
  assert.match(source, /getMyTicket\(id:\s*number\)[\s\S]*?`\/api\/v1\/ticket\/tickets\/\$\{id\}`/)
  assert.match(
    source,
    /addMyTicketReply\(id:\s*number,[\s\S]*?request\.post[\s\S]*?`\/api\/v1\/ticket\/tickets\/\$\{id\}\/replies`/
  )
  assert.match(source, /listTicketCategories\([\s\S]*?url:\s*'\/api\/v1\/ticket\/categories'/)
})

test('ticket admin APIs keep expected system endpoints and methods', () => {
  assert.match(
    source,
    /adminListTickets\([\s\S]*?request\.get[\s\S]*?url:\s*'\/api\/v1\/system\/ticket\/tickets'/
  )
  assert.match(
    source,
    /adminUpdateTicketStatus\(id:\s*number,[\s\S]*?request\.put[\s\S]*?`\/api\/v1\/system\/ticket\/tickets\/\$\{id\}\/status`/
  )
  assert.match(
    source,
    /adminUpdateTicketPriority\(id:\s*number,[\s\S]*?request\.put[\s\S]*?`\/api\/v1\/system\/ticket\/tickets\/\$\{id\}\/priority`/
  )
  assert.match(
    source,
    /adminAddTicketReply\(id:\s*number,[\s\S]*?request\.post[\s\S]*?`\/api\/v1\/system\/ticket\/tickets\/\$\{id\}\/replies`/
  )
  assert.match(
    source,
    /adminDeleteTicketCategory\(id:\s*number\)[\s\S]*?request\.del[\s\S]*?`\/api\/v1\/system\/ticket\/categories\/\$\{id\}`/
  )
  assert.match(
    source,
    /adminTicketStatistics\([\s\S]*?url:\s*'\/api\/v1\/system\/ticket\/statistics'/
  )
})
