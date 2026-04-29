import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const manageSource = readFileSync(new URL('./ticket-management/index.vue', import.meta.url), 'utf8')
const detailSource = readFileSync(new URL('./ticket-detail/index.vue', import.meta.url), 'utf8')
const categoriesSource = readFileSync(
  new URL('./ticket-categories/index.vue', import.meta.url),
  'utf8'
)
const statsSource = readFileSync(new URL('./ticket-statistics/index.vue', import.meta.url), 'utf8')

test('ticket management page uses admin list and status/priority update APIs', () => {
  assert.match(
    manageSource,
    /import \{[\s\S]*adminListTickets,[\s\S]*adminUpdateTicketPriority,[\s\S]*adminUpdateTicketStatus[\s\S]*\} from '@\/api\/ticket'/
  )
  assert.match(manageSource, /apiFn:\s*adminListTickets/)
  assert.match(
    manageSource,
    /apiParams:\s*\{[\s\S]*keyword:\s*filters\.keyword,[\s\S]*status:\s*filters\.status[\s\S]*\}/
  )
  assert.match(manageSource, /await adminUpdateTicketStatus\(id, \{ status \}\)/)
  assert.match(manageSource, /await adminUpdateTicketPriority\(id, \{ priority \}\)/)
  assert.match(
    manageSource,
    /router\.push\(\{ name: 'TicketAdminDetail', params: \{ id: String\(id\) \} \}\)/
  )
  assert.match(manageSource, /<ArtTableHeader v-model:columns="columnChecks"/)
  assert.match(manageSource, /<ArtTable/)
  assert.doesNotMatch(manageSource, /<ElTable :data=/)
})

test('ticket admin detail page loads ticket replies history and supports internal reply submit', () => {
  assert.match(
    detailSource,
    /import \{[\s\S]*adminAddTicketReply,[\s\S]*adminGetTicket,[\s\S]*adminListTicketReplies,[\s\S]*adminListTicketStatusHistory[\s\S]*\} from '@\/api\/ticket'/
  )
  assert.match(
    detailSource,
    /Promise\.all\(\[[\s\S]*adminGetTicket\(ticketId\.value\),[\s\S]*adminListTicketReplies\(ticketId\.value\),[\s\S]*adminListTicketStatusHistory\(ticketId\.value\)[\s\S]*\]\)/
  )
  assert.match(
    detailSource,
    /await adminAddTicketReply\(ticketId\.value, \{[\s\S]*content: content\.value,[\s\S]*is_internal: isInternal\.value[\s\S]*\}\)/
  )
})

test('ticket categories page supports list create update delete admin APIs', () => {
  assert.match(
    categoriesSource,
    /import \{[\s\S]*adminCreateTicketCategory,[\s\S]*adminDeleteTicketCategory,[\s\S]*adminListTicketCategories,[\s\S]*adminUpdateTicketCategory[\s\S]*\} from '@\/api\/ticket'/
  )
  assert.match(categoriesSource, /apiFn:\s*listTicketCategoriesTable/)
  assert.match(categoriesSource, /const list = await adminListTicketCategories\(\)/)
  assert.match(categoriesSource, /await adminCreateTicketCategory\(form\)/)
  assert.match(categoriesSource, /await adminUpdateTicketCategory\(editingId\.value, form\)/)
  assert.match(categoriesSource, /await adminDeleteTicketCategory\(id\)/)
  assert.match(categoriesSource, /<ArtTableHeader v-model:columns="columnChecks"/)
  assert.match(categoriesSource, /<ArtTable :loading="loading" :data="data" :columns="columns" \/>/)
  assert.doesNotMatch(categoriesSource, /<ElTable :data=/)
})

test('ticket statistics page loads dashboard stats through adminTicketStatistics API', () => {
  assert.match(statsSource, /import \{ adminTicketStatistics \} from '@\/api\/ticket'/)
  assert.match(statsSource, /stats\.value = await adminTicketStatistics\(\)/)
  assert.match(statsSource, /stats\?\.recent_7d \?\? 0/)
  assert.match(statsSource, /stats\?\.recent_30d \?\? 0/)
})
