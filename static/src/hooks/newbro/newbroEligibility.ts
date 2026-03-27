const INELIGIBILITY_REASON_KEYS: Record<string, string> = {
  skill_point_threshold_reached: 'newbro.select.ineligibleBecauseSkillThreshold',
  multi_character_skill_point_threshold_reached:
    'newbro.select.ineligibleBecauseMultiCharacterThreshold'
}

export function getNewbroIneligibilityReasonKey(reason?: string | null): string {
  if (!reason) {
    return 'newbro.select.currentlyIneligible'
  }

  return INELIGIBILITY_REASON_KEYS[reason] ?? 'newbro.select.currentlyIneligible'
}
