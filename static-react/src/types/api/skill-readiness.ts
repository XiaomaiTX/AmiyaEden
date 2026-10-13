export type SkillReadinessStatus = 'satisfied' | 'unsatisfied' | 'unknown'

export interface SkillReadinessRequest {
  corporation_ids?: number[]
  plan_id: number
  page?: number
  page_size?: number
  language?: 'zh' | 'en'
}

export interface SkillReadinessPlanOption {
  id: number
  title: string
  description: string
  ship_type_id: number | null
}

export interface SkillReadinessCorporationOption {
  corporation_id: number
  corporation_name: string
}

export interface SkillReadinessOptions {
  corporations: SkillReadinessCorporationOption[]
  plans: SkillReadinessPlanOption[]
}

export interface SkillReadinessSummary {
  character_count: number
  user_count: number
  satisfied_user_count: number
  satisfied_character_count: number
  unknown_character_count: number
}

export interface SkillReadinessUser {
  user_id: number
  nickname: string
  character_count: number
  satisfied_character_count: number
  unknown_character_count: number
}

export interface SkillReadinessResult {
  summary: SkillReadinessSummary
  list: SkillReadinessUser[]
  total: number
  page: number
  pageSize: number
}

export interface SkillReadinessCharacter {
  user_id: number
  character_id: number
  character_name: string
  corporation_id: number
  corporation_name: string
  status: SkillReadinessStatus
  matched_skills: number
  total_skills: number
  skill_updated_at: number | null
  token_invalid: boolean
}

export interface SkillReadinessSkill {
  skill_type_id: number
  skill_name: string
  group_name: string
  active_level: number | null
  trained_level: number | null
  skillpoints_in_skill: number | null
  required_level?: number
  current_level: number | null
  satisfied?: boolean
}

export interface SkillReadinessCharacterDetail {
  character: SkillReadinessCharacter
  requirements: SkillReadinessSkill[]
  skills: SkillReadinessSkill[]
}
