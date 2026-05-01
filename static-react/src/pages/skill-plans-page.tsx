import { useMemo } from 'react'
import { fetchSkillPlanDetail, fetchSkillPlanList, createSkillPlan, updateSkillPlan, deleteSkillPlan, reorderSkillPlans } from '@/api/skill-plan'
import { useSessionStore } from '@/stores'
import { SkillPlanManagementPage } from './skill-plan-management-page'

export function SkillPlansPage() {
  const roles = useSessionStore((state) => state.roles)
  const canManage = useMemo(
    () => roles.some((role) => ['super_admin', 'admin', 'senior_fc'].includes(role)),
    [roles]
  )

  return (
    <SkillPlanManagementPage
      titleKey="skillPlan.title"
      subtitleKey="skillPlan.subtitle"
      emptyKey="skillPlan.emptyList"
      canManage={canManage}
      fetchList={fetchSkillPlanList}
      fetchDetail={fetchSkillPlanDetail}
      createPlan={createSkillPlan}
      updatePlan={updateSkillPlan}
      deletePlan={deleteSkillPlan}
      reorderPlans={reorderSkillPlans}
    />
  )
}
