import { requestJson } from '@/api/http-client'
import type { ApiResponse } from '@/api/response'
import type {
  CheckPlanSelection,
  CheckSelection,
  CompletionCheckParams,
  CompletionCheckResult,
  CreateSkillPlanParams,
  SkillPlanDetail,
  SkillPlanListResponse,
  SkillPlanSearchParams,
  UpdateSkillPlanParams,
} from '@/types/api/skill-plan'

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 0 && response.code !== 200) {
    throw new Error(response.msg || fallbackMessage)
  }

  return response.data
}

type ApiResult<T> = ApiResponse<T>

export async function fetchSkillPlanList(params?: SkillPlanSearchParams) {
  const searchParams = new URLSearchParams()
  const current = params?.current ?? 1
  const size = params?.size ?? 10
  searchParams.set('current', String(current))
  searchParams.set('size', String(size))
  if (params?.keyword) {
    searchParams.set('keyword', params.keyword)
  }

  const response = await requestJson<ApiResult<SkillPlanListResponse>>(
    `/api/v1/skill-planning/skill-plans?${searchParams.toString()}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch skill plan list failed')
}

export async function fetchPersonalSkillPlanList(params?: SkillPlanSearchParams) {
  const searchParams = new URLSearchParams()
  const current = params?.current ?? 1
  const size = params?.size ?? 10
  searchParams.set('current', String(current))
  searchParams.set('size', String(size))
  if (params?.keyword) {
    searchParams.set('keyword', params.keyword)
  }

  const response = await requestJson<ApiResult<SkillPlanListResponse>>(
    `/api/v1/skill-planning/personal-skill-plans?${searchParams.toString()}`,
    {
      method: 'GET',
    }
  )
  return assertSuccess(response, 'fetch personal skill plan list failed')
}

export async function fetchSkillPlanDetail(id: number, lang?: string) {
  const url = lang
    ? `/api/v1/skill-planning/skill-plans/${id}?${new URLSearchParams({ lang }).toString()}`
    : `/api/v1/skill-planning/skill-plans/${id}`
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, { method: 'GET' })
  return assertSuccess(response, 'fetch skill plan detail failed')
}

export async function fetchPersonalSkillPlanDetail(id: number, lang?: string) {
  const url = lang
    ? `/api/v1/skill-planning/personal-skill-plans/${id}?${new URLSearchParams({ lang }).toString()}`
    : `/api/v1/skill-planning/personal-skill-plans/${id}`
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, { method: 'GET' })
  return assertSuccess(response, 'fetch personal skill plan detail failed')
}

export async function createSkillPlan(data: CreateSkillPlanParams, lang?: string) {
  const url = lang
    ? `/api/v1/skill-planning/skill-plans?${new URLSearchParams({ lang }).toString()}`
    : '/api/v1/skill-planning/skill-plans'
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create skill plan failed')
}

export async function createPersonalSkillPlan(data: CreateSkillPlanParams, lang?: string) {
  const url = lang
    ? `/api/v1/skill-planning/personal-skill-plans?${new URLSearchParams({ lang }).toString()}`
    : '/api/v1/skill-planning/personal-skill-plans'
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create personal skill plan failed')
}

export async function updateSkillPlan(id: number, data: UpdateSkillPlanParams, lang?: string) {
  const url = lang
    ? `/api/v1/skill-planning/skill-plans/${id}?${new URLSearchParams({ lang }).toString()}`
    : `/api/v1/skill-planning/skill-plans/${id}`
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update skill plan failed')
}

export async function updatePersonalSkillPlan(
  id: number,
  data: UpdateSkillPlanParams,
  lang?: string
) {
  const url = lang
    ? `/api/v1/skill-planning/personal-skill-plans/${id}?${new URLSearchParams({ lang }).toString()}`
    : `/api/v1/skill-planning/personal-skill-plans/${id}`
  const response = await requestJson<ApiResult<SkillPlanDetail>>(url, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update personal skill plan failed')
}

export async function reorderSkillPlans(ids: number[]) {
  const response = await requestJson<ApiResult<null>>('/api/v1/skill-planning/skill-plans/reorder', {
    method: 'PUT',
    body: JSON.stringify({ ids }),
  })
  return assertSuccess(response, 'reorder skill plans failed')
}

export async function reorderPersonalSkillPlans(ids: number[]) {
  const response = await requestJson<ApiResult<null>>(
    '/api/v1/skill-planning/personal-skill-plans/reorder',
    {
      method: 'PUT',
      body: JSON.stringify({ ids }),
    }
  )
  return assertSuccess(response, 'reorder personal skill plans failed')
}

export async function deleteSkillPlan(id: number) {
  const response = await requestJson<ApiResult<null>>(`/api/v1/skill-planning/skill-plans/${id}`, {
    method: 'DELETE',
  })
  return assertSuccess(response, 'delete skill plan failed')
}

export async function deletePersonalSkillPlan(id: number) {
  const response = await requestJson<ApiResult<null>>(
    `/api/v1/skill-planning/personal-skill-plans/${id}`,
    {
      method: 'DELETE',
    }
  )
  return assertSuccess(response, 'delete personal skill plan failed')
}

export async function fetchSkillPlanCheckSelection() {
  const response = await requestJson<ApiResult<CheckSelection>>(
    '/api/v1/skill-planning/skill-plans/check/selection',
    { method: 'GET' }
  )
  return assertSuccess(response, 'fetch skill plan selection failed')
}

export async function saveSkillPlanCheckSelection(data: CheckSelection) {
  const response = await requestJson<ApiResult<CheckSelection>>(
    '/api/v1/skill-planning/skill-plans/check/selection',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'save skill plan selection failed')
}

export async function fetchSkillPlanCheckPlanSelection() {
  const response = await requestJson<ApiResult<CheckPlanSelection>>(
    '/api/v1/skill-planning/skill-plans/check/plan-selection',
    { method: 'GET' }
  )
  return assertSuccess(response, 'fetch skill plan plan selection failed')
}

export async function saveSkillPlanCheckPlanSelection(data: CheckPlanSelection) {
  const response = await requestJson<ApiResult<CheckPlanSelection>>(
    '/api/v1/skill-planning/skill-plans/check/plan-selection',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'save skill plan plan selection failed')
}

export async function runSkillPlanCompletionCheck(data?: CompletionCheckParams) {
  const response = await requestJson<ApiResult<CompletionCheckResult>>(
    '/api/v1/skill-planning/skill-plans/check/run',
    {
      method: 'POST',
      body: JSON.stringify(data ?? {}),
    }
  )
  return assertSuccess(response, 'run skill plan completion check failed')
}
