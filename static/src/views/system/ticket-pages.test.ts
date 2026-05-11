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

test('ticket management page uses admin list and status update APIs', () => {
  assert.match(
    manageSource,
    /import \{[\s\S]*adminListTickets,[\s\S]*adminUpdateTicketStatus[\s\S]*\} from '@\/api\/ticket'/
  )
  assert.doesNotMatch(
    manageSource,
    /adminUpdateTicketPriority|updatePriority|TicketPriority|priority/
  )
  assert.match(manageSource, /apiFn:\s*adminListTickets/)
  assert.match(
    manageSource,
    /apiParams:\s*\{[\s\S]*keyword:\s*filters\.keyword,[\s\S]*status:\s*getTabStatusQuery\('pending'\),[\s\S]*category_id:\s*filters\.category_id[\s\S]*\}/
  )
  assert.match(manageSource, /await adminUpdateTicketStatus\(id, \{ status \}\)/)
  assert.match(
    manageSource,
    /router\.push\(\{ name: 'TicketAdminDetail', params: \{ id: String\(id\) \} \}\)/
  )
  assert.match(manageSource, /<ArtTableHeader v-model:columns="columnChecks"/)
  assert.match(manageSource, /<ArtTable/)
  assert.match(manageSource, /import \{ formatTime \} from '@utils\/common'/)
  assert.match(
    manageSource,
    /formatter: \(row\) => h\('span', \{\}, formatTime\(row\.updated_at\)\)/
  )
  assert.doesNotMatch(manageSource, /<ElTable :data=/)
})

test('ticket management page shows pending/in-progress/completed tabs and ticket content preview', () => {
  assert.match(manageSource, /<ElTabs v-model="activeTab" @tab-change="handleStatusTabChange">/)
  assert.match(manageSource, /<ElTabPane :label="t\('ticket.tabs.pending'\)" name="pending" \/>/)
  assert.match(
    manageSource,
    /<ElTabPane :label="t\('ticket.tabs.inProgress'\)" name="in_progress" \/>/
  )
  assert.match(
    manageSource,
    /<ElTabPane :label="t\('ticket.tabs.completed'\)" name="completed" \/>/
  )
  assert.match(manageSource, /pending:\s*'pending'/)
  assert.match(manageSource, /in_progress:\s*'in_progress'/)
  assert.match(manageSource, /completed:\s*'completed'/)
  assert.match(manageSource, /prop: 'description'/)
  assert.match(manageSource, /ticket-content-preview/)
  assert.match(manageSource, /adminListTicketCategories/)
  assert.match(manageSource, /prop: 'requester_name'/)
  assert.match(
    manageSource,
    /row\.user_nickname \|\|[\s\S]*row\.requester_name \|\|[\s\S]*row\.requester_character_name \|\|[\s\S]*t\('ticket\.unknownUser'\)/
  )
  assert.doesNotMatch(manageSource, /String\(row\.user_id\)/)
  assert.match(manageSource, /prop: 'category_name'/)
  assert.match(manageSource, /category_id:\s*filters\.category_id/)
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
  assert.match(detailSource, /formatTicketRequester\(ticket\)/)
  assert.match(detailSource, /formatTicketRequester = \(item: Api\.Ticket\.TicketItem\)/)
  assert.doesNotMatch(detailSource, /ticket\.requester_name \|\| ticket\.user_id/)
  assert.match(detailSource, /getTicketCategoryLabel\(ticket\)/)
  assert.match(detailSource, /import \{ formatTime \} from '@utils\/common'/)
  assert.match(detailSource, /formatTime\(row\.changed_at\)/)
  assert.match(detailSource, /formatTicketHistoryOperator\(row\)/)
  assert.match(detailSource, /<TicketStatusBadge[\s\S]*v-if="row\.from_status"[\s\S]*\/>/)
  assert.match(detailSource, /<TicketStatusBadge[\s\S]*:status="row\.to_status"[\s\S]*\/>/)
  assert.doesNotMatch(detailSource, /prop="changed_by"/)
  assert.doesNotMatch(detailSource, /\{\{\s*row\.changed_by\s*\}\}/)
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
