import {
  createPersonalSkillPlan,
  deletePersonalSkillPlan,
  fetchPersonalSkillPlanDetail,
  fetchPersonalSkillPlanList,
  reorderPersonalSkillPlans,
  updatePersonalSkillPlan,
} from '@/api/skill-plan'
import { SkillPlanManagementPage } from './skill-plan-management-page'

export function PersonalSkillPlansPage() {
  return (
    <SkillPlanManagementPage
      titleKey="skillPlan.personalTitle"
      subtitleKey="skillPlan.personalSubtitle"
      emptyKey="skillPlan.emptyList"
      canManage
      fetchList={fetchPersonalSkillPlanList}
      fetchDetail={fetchPersonalSkillPlanDetail}
      createPlan={createPersonalSkillPlan}
      updatePlan={updatePersonalSkillPlan}
      deletePlan={deletePersonalSkillPlan}
      reorderPlans={reorderPersonalSkillPlans}
    />
  )
}
