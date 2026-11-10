export const refundStatusKeys = {
  pending: 'refund.pending',
  approved: 'refund.approved',
  rejected: 'refund.rejected',
} as const

const refundReasonKeys: Record<string, string> = {
  no_characters: 'refund.reason.no_characters',
  missing_character_data: 'refund.reason.missing_character_data',
  skill_points_too_high: 'refund.reason.skill_points_too_high',
  character_too_old: 'refund.reason.character_too_old',
}
export function refundEligibilityReasonKey(reason: string) {
  return refundReasonKeys[reason] ?? 'refund.reason.missing_character_data'
}
