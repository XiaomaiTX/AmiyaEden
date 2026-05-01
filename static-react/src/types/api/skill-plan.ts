import type { PaginatedResponse } from '@/types/api/common'

export type SkillPlanScope = 'corp' | 'personal'

export interface SkillPlanListItem {
  id: number
  title: string
  description: string
  plan_scope: SkillPlanScope
  ship_type_id: number | null
  sort_order: number
  created_by: number
  created_at: string
  updated_at: string
  skill_count: number
}

export interface SkillRequirement {
  id: number
  skill_plan_id: number
  skill_type_id: number
  skill_name: string
  group_name: string
  required_level: number
  sort: number
}

export interface SkillPlanDetail {
  id: number
  title: string
  description: string
  plan_scope: SkillPlanScope
  ship_type_id: number | null
  ship_name: string
  sort_order: number
  created_by: number
  created_at: string
  updated_at: string
  skill_count: number
  skills: SkillRequirement[]
}

export type SkillPlanListResponse = PaginatedResponse<SkillPlanListItem>

export interface SkillPlanSearchParams {
  current?: number
  size?: number
  keyword?: string
}

export interface SkillRequirementParams {
  skill_type_id: number
  required_level: number
}

export interface CreateSkillPlanParams {
  title: string
  description?: string
  ship_type_id?: number
  sort_order?: number
  skills?: SkillRequirementParams[]
  skills_text?: string
}

export type UpdateSkillPlanParams = CreateSkillPlanParams

export interface CheckSelection {
  character_ids: number[]
}

export interface CheckPlanSelection {
  plan_ids: number[]
}

export interface CompletionMissingSkill {
  skill_type_id: number
  skill_name: string
  group_name: string
  required_level: number
  current_level: number
}

export interface CompletionPlan {
  plan_id: number
  plan_title: string
  plan_description: string
  ship_type_id: number | null
  matched_skills: number
  total_skills: number
  fully_satisfied: boolean
  missing_skills: CompletionMissingSkill[]
}

export interface CompletionCharacter {
  character_id: number
  character_name: string
  completed_plans: number
  total_plans: number
  plans: CompletionPlan[]
}

export interface CompletionCheckResult {
  characters: CompletionCharacter[]
  plan_count: number
}

export interface CompletionCheckParams {
  character_ids?: number[]
  language?: string
}
