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

test('ticket management page uses admin list API and readonly status/priority badges', () => {
  assert.match(manageSource, /import \{[\s\S]*adminListTickets[\s\S]*\} from '@\/api\/ticket'/)
  assert.match(manageSource, /apiFn:\s*adminListTickets/)
  assert.match(
    manageSource,
    /apiParams:\s*\{[\s\S]*keyword:\s*filters\.keyword,[\s\S]*status:\s*filters\.status,[\s\S]*priority:\s*filters\.priority[\s\S]*\}/
  )
  assert.match(manageSource, /prop:\s*'user_nickname'/)
  assert.match(manageSource, /TicketStatusBadge/)
  assert.match(manageSource, /TicketPriorityBadge/)
  assert.match(
    manageSource,
    /formatter: \(row\) => h\(TicketStatusBadge, \{ status: row\.status \}\)/
  )
  assert.match(
    manageSource,
    /formatter: \(row\) => h\(TicketPriorityBadge, \{ priority: row\.priority \}\)/
  )
  assert.doesNotMatch(manageSource, /await adminUpdateTicketStatus\(id, \{ status \}\)/)
  assert.doesNotMatch(manageSource, /await adminUpdateTicketPriority\(id, \{ priority \}\)/)
  assert.match(
    manageSource,
    /router\.push\(\{ name: 'TicketAdminDetail', params: \{ id: String\(id\) \} \}\)/
  )
  assert.match(manageSource, /<ArtTableHeader v-model:columns="columnChecks"/)
  assert.match(manageSource, /<ArtTable/)
  assert.doesNotMatch(manageSource, /<ElTable :data=/)
})

test('ticket admin detail page supports status/priority edit and status history badges', () => {
  assert.match(
    detailSource,
    /import \{[\s\S]*adminAddTicketReply,[\s\S]*adminGetTicket,[\s\S]*adminListTicketReplies,[\s\S]*adminListTicketStatusHistory,[\s\S]*adminUpdateTicketPriority,[\s\S]*adminUpdateTicketStatus[\s\S]*\} from '@\/api\/ticket'/
  )
  assert.match(
    detailSource,
    /Promise\.all\(\[[\s\S]*adminGetTicket\(ticketId\.value\),[\s\S]*adminListTicketReplies\(ticketId\.value\),[\s\S]*adminListTicketStatusHistory\(ticketId\.value\)[\s\S]*\]\)/
  )
  assert.match(
    detailSource,
    /await adminAddTicketReply\(ticketId\.value, \{[\s\S]*content: content\.value,[\s\S]*is_internal: isInternal\.value[\s\S]*\}\)/
  )
  assert.match(detailSource, /const hasMetaChanges = computed\(/)
  assert.match(
    detailSource,
    /await adminUpdateTicketStatus\(ticketId\.value, \{ status: editStatus\.value \}\)/
  )
  assert.match(
    detailSource,
    /await adminUpdateTicketPriority\(ticketId\.value, \{ priority: editPriority\.value \}\)/
  )
  assert.match(detailSource, /if \(!isTicketStatus\(status\)\) \{/)
  assert.match(detailSource, /return h\(TicketStatusBadge, \{ status \}\)/)
  assert.match(detailSource, /prop:\s*'changed_by_nickname'/)
  assert.match(detailSource, /<ArtTable :data="histories" :columns="historyColumns" \/>/)
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
