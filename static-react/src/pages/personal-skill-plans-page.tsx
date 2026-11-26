import {
  createPersonalSkillPlan,
  getPersonalSkillPlan,
  listPersonalSkillPlans,
  updatePersonalSkillPlan,
} from '@/api/generated/client'
import { deletePersonalSkillPlan, reorderPersonalSkillPlans } from '@/api/generated/client'
import { SkillPlanManagementPage } from './skill-plan-management-page'

export function PersonalSkillPlansPage() {
  return (
    <SkillPlanManagementPage
      titleKey="skillPlan.personalTitle"
      subtitleKey="skillPlan.personalSubtitle"
      emptyKey="skillPlan.emptyList"
      canManage
      personal
      fetchList={listPersonalSkillPlans}
      fetchDetail={getPersonalSkillPlan}
      createPlan={createPersonalSkillPlan}
      updatePlan={updatePersonalSkillPlan}
      deletePlan={deletePersonalSkillPlan}
      reorderPlans={reorderPersonalSkillPlans}
    />
  )
}
