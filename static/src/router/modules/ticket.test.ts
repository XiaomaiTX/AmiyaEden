import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./ticket.ts', import.meta.url), 'utf8')

test('ticket route tree keeps login-gated member pages', () => {
  assert.match(source, /name:\s*'TicketRoot'/)
  assert.match(source, /title:\s*'menus\.ticket\.title'/)
  assert.match(source, /icon:\s*'ri:question-answer-line'/)

  const myListBlock = source.slice(
    source.indexOf("path: 'my-tickets'"),
    source.indexOf("path: 'create'")
  )
  assert.match(myListBlock, /name:\s*'TicketMyList'/)
  assert.match(myListBlock, /login:\s*true/)

  const createBlock = source.slice(
    source.indexOf("path: 'create'"),
    source.indexOf("path: 'detail/:'")
  )
  assert.match(createBlock, /name:\s*'TicketCreate'/)
  assert.match(createBlock, /login:\s*true/)
})

test('ticket detail route remains hidden from menus and tabs', () => {
  const detailBlock = source.slice(source.indexOf("path: 'detail/:id'"), source.length)
  assert.match(detailBlock, /name:\s*'TicketDetail'/)
  assert.match(detailBlock, /isHide:\s*true/)
  assert.match(detailBlock, /isHideTab:\s*true/)
  assert.match(detailBlock, /login:\s*true/)
})
