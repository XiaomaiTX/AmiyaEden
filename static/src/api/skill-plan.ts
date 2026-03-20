import request from '@/utils/http'

/** 获取军团技能计划列表 */
export function fetchSkillPlanList(params?: Api.SkillPlan.SkillPlanSearchParams) {
  return request.get<Api.SkillPlan.SkillPlanList>({
    url: '/api/v1/operation/skill-plans',
    params
  })
}

/** 获取军团技能计划详情 */
export function fetchSkillPlanDetail(id: number, lang?: string) {
  return request.get<Api.SkillPlan.SkillPlanDetail>({
    url: `/api/v1/operation/skill-plans/${id}`,
    params: lang ? { lang } : undefined
  })
}

/** 创建军团技能计划 */
export function createSkillPlan(data: Api.SkillPlan.CreateSkillPlanParams, lang?: string) {
  return request.post<Api.SkillPlan.SkillPlanDetail>({
    url: '/api/v1/operation/skill-plans',
    data,
    params: lang ? { lang } : undefined
  })
}

/** 更新军团技能计划 */
export function updateSkillPlan(
  id: number,
  data: Api.SkillPlan.UpdateSkillPlanParams,
  lang?: string
) {
  return request.put<Api.SkillPlan.SkillPlanDetail>({
    url: `/api/v1/operation/skill-plans/${id}`,
    data,
    params: lang ? { lang } : undefined
  })
}

/** 删除军团技能计划 */
export function deleteSkillPlan(id: number) {
  return request.del({
    url: `/api/v1/operation/skill-plans/${id}`
  })
}
