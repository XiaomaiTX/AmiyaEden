import type { Component, VNode } from 'vue'
import {
  ElCascader,
  ElCheckbox,
  ElCheckboxGroup,
  ElDatePicker,
  ElInput,
  ElInputNumber,
  ElInputTag,
  ElRadioGroup,
  ElRate,
  ElSelect,
  ElSlider,
  ElSwitch,
  ElTimePicker,
  ElTimeSelect,
  ElTreeSelect
} from 'element-plus'

type FormSlot = (() => any) | undefined
type FormRenderable = (() => VNode) | Component

export interface SharedFormItem {
  key?: string
  label?: string | FormRenderable
  labelWidth?: string | number
  type?: keyof typeof formComponentMap | string
  render?: FormRenderable
  hidden?: boolean
  span?: number
  slots?: Record<string, FormSlot>
  props?: Record<string, any>
}

export const formComponentMap = {
  input: ElInput,
  inputTag: ElInputTag,
  inputtag: ElInputTag,
  number: ElInputNumber,
  select: ElSelect,
  switch: ElSwitch,
  checkbox: ElCheckbox,
  checkboxgroup: ElCheckboxGroup,
  radiogroup: ElRadioGroup,
  date: ElDatePicker,
  daterange: ElDatePicker,
  datetime: ElDatePicker,
  datetimerange: ElDatePicker,
  rate: ElRate,
  slider: ElSlider,
  cascader: ElCascader,
  timepicker: ElTimePicker,
  timeselect: ElTimeSelect,
  treeselect: ElTreeSelect
}

const formRootProps = ['label', 'labelWidth', 'key', 'type', 'hidden', 'span', 'slots']

export function getFormItemProps<T extends SharedFormItem>(item: T) {
  if (item.props) return item.props
  const props = { ...item } as Record<string, any>
  formRootProps.forEach((key) => delete props[key])
  return props
}

export function getFormItemSlots<T extends SharedFormItem>(item: T): Record<string, () => any> {
  if (!item.slots) return {}

  const validSlots: Record<string, () => any> = {}
  Object.entries(item.slots).forEach(([key, slotFn]) => {
    if (slotFn) {
      validSlots[key] = slotFn
    }
  })
  return validSlots
}

export function resolveFormItemComponent<T extends SharedFormItem>(item: T) {
  if (item.render) {
    return item.render
  }

  const { type } = item
  return formComponentMap[type as keyof typeof formComponentMap] || formComponentMap.input
}
