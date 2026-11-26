import type { FleetConfigFittingItem, UpdateItemSettingsReq } from '@/api/generated/client'

export const itemImportanceOptions = ['required', 'optional', 'replaceable'] as const
export const itemPenaltyOptions = ['half', 'none'] as const

export interface ItemSettingsEdit {
  importance: (typeof itemImportanceOptions)[number] | undefined
  penalty: (typeof itemPenaltyOptions)[number] | undefined
  replacementPenalty: (typeof itemPenaltyOptions)[number] | undefined
  replacements: number[]
}

export function toItemSettingsEdit(item: FleetConfigFittingItem): ItemSettingsEdit {
  return {
    importance: itemImportanceOptions.find((value) => value === item.importance),
    penalty: itemPenaltyOptions.find((value) => value === item.penalty),
    replacementPenalty: itemPenaltyOptions.find((value) => value === item.replacement_penalty),
    replacements: item.replacements?.map((replacement) => replacement.type_id) ?? []
  }
}

export function toItemSettingsUpdate(
  item: FleetConfigFittingItem,
  edit: ItemSettingsEdit
): UpdateItemSettingsReq {
  // An unrecognized stored value has no matching option; keep it until the user chooses one.
  return {
    id: item.id,
    importance: edit.importance ?? item.importance,
    penalty: edit.penalty ?? item.penalty,
    replacement_penalty: edit.replacementPenalty ?? item.replacement_penalty,
    replacements: edit.replacements
  }
}
