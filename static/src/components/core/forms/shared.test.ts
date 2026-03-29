import assert from 'node:assert/strict'
import test from 'node:test'

import { getFormItemProps, getFormItemSlots, resolveFormItemComponent } from './shared'

test('getFormItemProps strips framework-only keys when props is not provided', () => {
  const item = {
    key: 'status',
    label: 'Status',
    labelWidth: '80px',
    type: 'select',
    hidden: false,
    span: 12,
    slots: {
      default: undefined
    },
    placeholder: 'Pick one',
    clearable: true
  }

  assert.deepEqual(getFormItemProps(item), {
    placeholder: 'Pick one',
    clearable: true
  })
})

test('getFormItemProps returns explicit props without reshaping', () => {
  const props = { placeholder: 'Search', clearable: true }
  const item = {
    key: 'keyword',
    label: 'Keyword',
    props
  }

  assert.equal(getFormItemProps(item), props)
})

test('getFormItemSlots keeps only defined slot handlers', () => {
  const suffix = () => 'suffix'
  const item = {
    slots: {
      prefix: undefined,
      suffix
    }
  }

  assert.deepEqual(getFormItemSlots(item), { suffix })
})

test('resolveFormItemComponent prefers custom render and supports both inputTag aliases', () => {
  const customRender = () => 'custom'

  assert.equal(resolveFormItemComponent({ render: customRender }), customRender)
  assert.equal(
    resolveFormItemComponent({ type: 'inputTag' }),
    resolveFormItemComponent({ type: 'inputtag' })
  )
})
