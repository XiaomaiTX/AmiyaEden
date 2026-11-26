import type { FuxiHallCard, FuxiHallContributions } from '@/types/api/fuxi-hall'
import type { HallFloor as FuxiHallFloor } from '@/api/generated/client'

export const CONTRIBUTION_KEYS = [
  'fleets',
  'welfare',
  'srp',
  'recruitment',
  'mentoring',
] as const

export type ContributionKey = (typeof CONTRIBUTION_KEYS)[number]
export type NumericInputDraft = number | ''
export type PageFloorDraft = Omit<FuxiHallFloor, 'capacity'> & { capacity: NumericInputDraft }
export type ContributionOffsetDrafts = Record<ContributionKey, NumericInputDraft>

export const FLOOR_CAPACITY_MIN = 1
export const FLOOR_CAPACITY_MAX = 10000
export const FONT_SCALE_MIN = 12
export const FONT_SCALE_MAX = 20
export const CONTRIBUTION_OFFSET_MIN = 0
export const CONTRIBUTION_OFFSET_MAX = 1_000_000_000

const DEFAULT_BUILTIN_FLOOR_KEYS = new Set(['top', 'second', 'first'])

export function parseIntegerInput(rawValue: string, max: number): NumericInputDraft {
  const trimmed = rawValue.trim()
  if (trimmed === '') return ''
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed)) return ''
  const truncated = Math.trunc(parsed)
  return Math.min(max, Math.max(0, truncated))
}

export function validateBoundedInteger(
  value: NumericInputDraft,
  min: number,
  max: number
): number | null {
  if (typeof value !== 'number' || !Number.isInteger(value)) return null
  if (value < min || value > max) return null
  return value
}

export function validateFloorDrafts(floors: PageFloorDraft[]): FuxiHallFloor[] | null {
  if (floors.length === 0 || floors.length > 12) return null
  const validated: FuxiHallFloor[] = []
  for (const floor of floors) {
    if (!DEFAULT_BUILTIN_FLOOR_KEYS.has(floor.key) && !floor.name.trim()) return null
    const capacity = validateBoundedInteger(floor.capacity, FLOOR_CAPACITY_MIN, FLOOR_CAPACITY_MAX)
    if (capacity === null) return null
    validated.push({
      key: floor.key,
      name: floor.name,
      capacity,
    })
  }
  return validated
}

export function validateContributionOffsets(
  drafts: ContributionOffsetDrafts
): FuxiHallContributions | null {
  const result = {
    fleets: 0,
    welfare: 0,
    srp: 0,
    recruitment: 0,
    mentoring: 0,
  }
  for (const key of CONTRIBUTION_KEYS) {
    const value = validateBoundedInteger(
      drafts[key],
      CONTRIBUTION_OFFSET_MIN,
      CONTRIBUTION_OFFSET_MAX
    )
    if (value === null) return null
    result[key] = value
  }
  return result
}

export function normalizeOffsetDraftsForPreview(
  drafts: ContributionOffsetDrafts
): FuxiHallContributions {
  return {
    fleets: typeof drafts.fleets === 'number' ? drafts.fleets : 0,
    welfare: typeof drafts.welfare === 'number' ? drafts.welfare : 0,
    srp: typeof drafts.srp === 'number' ? drafts.srp : 0,
    recruitment: typeof drafts.recruitment === 'number' ? drafts.recruitment : 0,
    mentoring: typeof drafts.mentoring === 'number' ? drafts.mentoring : 0,
  }
}

// Apply the draft to the full card list before filtering, so toggling visibility on an existing
// card adds or removes it from the floor occupancy preview.
export function withPreviewCard(cards: FuxiHallCard[], preview: FuxiHallCard): FuxiHallCard[] {
  const merged = cards.some((card) => card.id === preview.id)
    ? cards.map((card) => (card.id === preview.id ? preview : card))
    : [...cards, preview]
  return merged.filter((card) => card.visible)
}
