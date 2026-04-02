import assert from 'node:assert/strict'
import test from 'node:test'
import {
  LEDGER_TABLE_PAGE_SIZES,
  LEDGER_TABLE_PAGINATION_LAYOUT,
  resolveArtTablePresentation
} from './presentation'

test('resolveArtTablePresentation applies ledger defaults for table visuals and pagination', () => {
  const presentation = resolveArtTablePresentation({
    visualVariant: 'ledger',
    basePaginationOptions: {
      pageSizes: [10, 20, 30, 50, 100],
      align: 'center',
      background: true,
      layout: 'total, prev, pager, next, sizes, jumper',
      hideOnSinglePage: false,
      size: 'default',
      pagerCount: 7
    }
  })

  assert.equal(presentation.border, true)
  assert.equal(presentation.stripe, true)
  assert.equal(presentation.paginationOptions.align, 'right')
  assert.deepEqual(presentation.paginationOptions.pageSizes, [...LEDGER_TABLE_PAGE_SIZES])
  assert.equal(presentation.paginationOptions.layout, 'total, sizes, prev, pager, next, jumper')
  assert.equal(presentation.paginationOptions.layout, LEDGER_TABLE_PAGINATION_LAYOUT)
  assert.deepEqual(presentation.tableClassNames, ['art-table--ledger'])
  assert.deepEqual(presentation.paginationClassNames, ['pagination--ledger'])
})

test('resolveArtTablePresentation preserves caller overrides on top of ledger defaults', () => {
  const presentation = resolveArtTablePresentation({
    visualVariant: 'ledger',
    border: false,
    paginationOptions: {
      pageSizes: [300, 600],
      pagerCount: 11
    },
    basePaginationOptions: {
      pageSizes: [10, 20, 30, 50, 100],
      align: 'center',
      background: true,
      layout: 'total, prev, pager, next, sizes, jumper',
      hideOnSinglePage: false,
      size: 'default',
      pagerCount: 7
    }
  })

  assert.equal(presentation.border, false)
  assert.deepEqual(presentation.paginationOptions.pageSizes, [300, 600])
  assert.equal(presentation.paginationOptions.pagerCount, 11)
  assert.equal(presentation.paginationOptions.align, 'right')
})
