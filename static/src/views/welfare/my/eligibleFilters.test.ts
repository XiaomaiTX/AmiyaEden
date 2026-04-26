import assert from 'node:assert/strict'
import test from 'node:test'
import {
  buildRoleFilterOptions,
  buildWelfareNameFilterOptions,
  filterEligibleRows,
  paginateEligibleRows,
  type EligibleFilters
} from './eligibleFilters'

type Row = {
  welfareName: string
  isNaturalPersonRow: boolean
  roleFilterValue: string
  id: string
}

const rows: Row[] = [
  {
    id: 'per-user-1',
    welfareName: 'Alpha',
    isNaturalPersonRow: true,
    roleFilterValue: 'Current User (Natural Person)'
  },
  {
    id: 'char-a-1',
    welfareName: 'Alpha',
    isNaturalPersonRow: false,
    roleFilterValue: 'Character A'
  },
  {
    id: 'char-b-1',
    welfareName: 'Beta',
    isNaturalPersonRow: false,
    roleFilterValue: 'Character B'
  }
]

const defaultFilters: EligibleFilters = {
  roleFilter: '',
  naturalPersonFilter: '',
  welfareNameFilter: ''
}

test('filterEligibleRows keeps all rows when filters are empty', () => {
  assert.deepEqual(
    filterEligibleRows(rows, defaultFilters).map((row) => row.id),
    ['per-user-1', 'char-a-1', 'char-b-1']
  )
})

test('filterEligibleRows applies role filter and natural person filter as intersection', () => {
  assert.deepEqual(
    filterEligibleRows(rows, {
      roleFilter: 'Current User (Natural Person)',
      naturalPersonFilter: 'per_user',
      welfareNameFilter: 'Alpha'
    }).map((row) => row.id),
    ['per-user-1']
  )
})

test('filterEligibleRows keeps only per_user rows when naturalPersonFilter is per_user', () => {
  assert.deepEqual(
    filterEligibleRows(rows, {
      ...defaultFilters,
      naturalPersonFilter: 'per_user'
    }).map((row) => row.id),
    ['per-user-1']
  )
})

test('paginateEligibleRows slices rows after filtering', () => {
  const filtered = filterEligibleRows(rows, { ...defaultFilters, welfareNameFilter: 'Alpha' })
  assert.deepEqual(
    paginateEligibleRows(filtered, 1, 1).map((row) => row.id),
    ['per-user-1']
  )
  assert.deepEqual(
    paginateEligibleRows(filtered, 2, 1).map((row) => row.id),
    ['char-a-1']
  )
})

test('buildRoleFilterOptions keeps natural person option and deduplicates character names', () => {
  const options = buildRoleFilterOptions(
    [...rows, { ...rows[1], id: 'char-a-2' }],
    'Current User (Natural Person)'
  )
  assert.deepEqual(options, [
    { label: 'Current User (Natural Person)', value: 'Current User (Natural Person)' },
    { label: 'Character A', value: 'Character A' },
    { label: 'Character B', value: 'Character B' }
  ])
})

test('buildWelfareNameFilterOptions deduplicates welfare names', () => {
  const options = buildWelfareNameFilterOptions(rows)
  assert.deepEqual(options, [
    { label: 'Alpha', value: 'Alpha' },
    { label: 'Beta', value: 'Beta' }
  ])
})
