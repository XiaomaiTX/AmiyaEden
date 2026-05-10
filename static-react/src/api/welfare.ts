import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import { uploadImageAsDataUrl } from '@/api/upload'
import type {
  AdminApplication,
  AdminApplicationListResponse,
  AdminApplicationSearchParams,
  AutoApproveConfig,
  CreateParams,
  EligibleWelfare,
  ImportRecordsParams,
  ImportRecordsResult,
  MyApplicationListResponse,
  MyApplicationSearchParams,
  ReviewParams,
  WelfareItem,
  WelfareListResponse,
  WelfareSearchParams,
  UpdateAutoApproveConfigParams,
  UpdateParams,
  ApplyParams,
} from '@/types/api/welfare'

type ApiResult<T> = ApiResponse<T>

export function uploadWelfareEvidence(file: File) {
  return uploadImageAsDataUrl(file, '/api/v1/welfare/upload-evidence')
}

export async function adminListWelfares(data?: WelfareSearchParams) {
  const response = await requestJson<ApiResult<WelfareListResponse>>('/api/v1/system/welfare/list', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 20 }),
  })
  return assertSuccess(response, 'list welfares failed')
}

export async function adminCreateWelfare(data: CreateParams) {
  const response = await requestJson<ApiResult<WelfareItem>>('/api/v1/system/welfare/add', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create welfare failed')
}

export async function adminUpdateWelfare(data: UpdateParams) {
  const response = await requestJson<ApiResult<WelfareItem>>('/api/v1/system/welfare/edit', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update welfare failed')
}

export async function adminReorderWelfares(ids: number[]) {
  const response = await requestJson<ApiResult<null>>('/api/v1/system/welfare/reorder', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  })
  return assertSuccess(response, 'reorder welfare failed')
}

export async function fetchWelfareAutoApproveConfig() {
  const response = await requestJson<ApiResult<AutoApproveConfig>>('/api/v1/system/welfare/settings')
  return assertSuccess(response, 'fetch welfare config failed')
}

export async function updateWelfareAutoApproveConfig(data: UpdateAutoApproveConfigParams) {
  const response = await requestJson<ApiResult<AutoApproveConfig>>('/api/v1/system/welfare/settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update welfare config failed')
}

export async function adminDeleteWelfare(id: number) {
  const response = await requestJson<ApiResult<null>>('/api/v1/system/welfare/delete', {
    method: 'POST',
    body: JSON.stringify({ id }),
  })
  return assertSuccess(response, 'delete welfare failed')
}

export async function adminImportWelfareRecords(data: ImportRecordsParams) {
  const response = await requestJson<ApiResult<ImportRecordsResult>>(
    '/api/v1/system/welfare/import',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'import welfare records failed')
}

export async function adminListApplications(data?: AdminApplicationSearchParams) {
  const response = await requestJson<ApiResult<AdminApplicationListResponse>>(
    '/api/v1/system/welfare/applications',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 50 }),
    }
  )
  return assertSuccess(response, 'list welfare applications failed')
}

export async function adminReviewApplication(data: ReviewParams) {
  const response = await requestJson<ApiResult<AdminApplication>>('/api/v1/system/welfare/review', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'review welfare application failed')
}

export async function adminDeleteApplication(id: number) {
  const response = await requestJson<ApiResult<null>>(
    '/api/v1/system/welfare/applications/delete',
    {
      method: 'POST',
      body: JSON.stringify({ id }),
    }
  )
  return assertSuccess(response, 'delete welfare application failed')
}

export async function getEligibleWelfares() {
  const response = await requestJson<ApiResult<EligibleWelfare[]>>('/api/v1/welfare/eligible', {
    method: 'POST',
  })
  return assertSuccess(response, 'fetch eligible welfares failed') ?? []
}

export async function applyForWelfare(data: ApplyParams) {
  const response = await requestJson<ApiResult<null>>('/api/v1/welfare/apply', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'apply for welfare failed')
}

export async function getMyApplications(data?: MyApplicationSearchParams) {
  const response = await requestJson<ApiResult<MyApplicationListResponse>>(
    '/api/v1/welfare/my-applications',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 20 }),
    }
  )
  return assertSuccess(response, 'fetch welfare applications failed')
}
