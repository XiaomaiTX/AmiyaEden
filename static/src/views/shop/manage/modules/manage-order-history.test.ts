import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./manage-order-history.vue', import.meta.url), 'utf8')
const ordersSource = readFileSync(
  new URL('../../browse/modules/shop-orders.vue', import.meta.url),
  'utf8'
)
const docSource = readFileSync(
  new URL('../../../../../../docs/features/current/commerce.md', import.meta.url),
  'utf8'
)
const zhLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/zh.json', import.meta.url), 'utf8')
)
const enLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/en.json', import.meta.url), 'utf8')
)

test('manage order history renders shared copy buttons for order number and main character', () => {
  assert.match(
    source,
    /import ArtCopyButton from '@\/components\/core\/forms\/art-copy-button\/index.vue'/
  )
  assert.match(source, /prop:\s*'order_no'[\s\S]*h\(ArtCopyButton,[\s\S]*text:\s*row\.order_no/)
  assert.match(
    source,
    /prop:\s*'main_character_name'[\s\S]*h\(ArtCopyButton,[\s\S]*text:\s*row\.main_character_name/
  )
})

test('shop order views label reviewer fields as operator', () => {
  assert.match(source, /shopAdmin\.orders\.table\.reviewerName/)
  assert.match(ordersSource, /shop\.reviewerName/)
  assert.equal(zhLocale.shop.reviewerName, '操作人')
  assert.equal(zhLocale.shopAdmin.orders.table.reviewerName, '操作人')
  assert.equal(enLocale.shop.reviewerName, 'Operator')
  assert.equal(enLocale.shopAdmin.orders.table.reviewerName, 'Operator')
  assert.match(docSource, /展示操作人与发放备注/)
  assert.match(docSource, /展示订单状态，以及在已发放\/已拒绝时展示操作人/)
})
