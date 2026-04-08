export interface WelfareReasonMessages {
  pap: string
  skill: string
  legionYears: string
  skillPlan: (plans: string) => string
  planSeparator: string
  reasonSeparator: string
}

interface ParsedWelfareReason {
  hasPap: boolean
  hasSkill: boolean
  hasLegionYears: boolean
}

function parseWelfareIneligibleReason(
  reason: Api.Welfare.EligibleWelfare['ineligible_reason'] | undefined
): ParsedWelfareReason {
  const tokenSet = new Set((reason ?? '').split('_').filter(Boolean))

  return tokenSet.has('legion') && tokenSet.has('years')
    ? {
        hasPap: tokenSet.has('pap'),
        hasSkill: tokenSet.has('skill'),
        hasLegionYears: true
      }
    : {
        hasPap: tokenSet.has('pap'),
        hasSkill: tokenSet.has('skill'),
        hasLegionYears: false
      }
}

export function formatWelfareIneligibleReason(
  reason: Api.Welfare.EligibleWelfare['ineligible_reason'] | undefined,
  skillPlanNames: string[] | undefined,
  messages: WelfareReasonMessages
) {
  const planNames = (skillPlanNames ?? []).map((name) => name.trim()).filter(Boolean)
  const joinedPlanNames = planNames.join(messages.planSeparator)
  const parts: string[] = []
  const parsedReason = parseWelfareIneligibleReason(reason)

  if (parsedReason.hasSkill) {
    parts.push(joinedPlanNames ? messages.skillPlan(joinedPlanNames) : messages.skill)
  }
  if (parsedReason.hasPap) {
    parts.push(messages.pap)
  }
  if (parsedReason.hasLegionYears) {
    parts.push(messages.legionYears)
  }

  if (parts.length > 0) {
    return parts.join(messages.reasonSeparator)
  }
  return messages.skill
}
