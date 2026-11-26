import type * as contract from '@/api/generated/client'
// The shared editor sends sort_order for both calls; the server only applies it on update.
export type CreateSkillPlanParams = contract.SkillPlanCreateInput &
  Pick<contract.SkillPlanUpdateInput, 'sort_order'>
