import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AdminRecruitLinksResponse,
  AdminRewardDistributionsParams,
  AdminRewardDistributionsResponse,
  CheckDirectReferrerParams,
  ConfirmDirectReferrerParams,
  DirectReferralStatus,
  DirectReferrerCandidate,
  EmptyResponse,
  GenerateLinkResponse,
  MenteeListResponse,
  MyStatusResponse,
  AdminRelationshipsParams,
  AdminRelationshipsResponse,
  RecruitLink,
  RelationshipActionParams,
  RewardProcessResult,
  RewardStage,
  UpdateRecruitSettingsParams,
  UpdateRewardStagesParams,
  RecruitSettings,
  RelationshipView,
  MentorMenteesParams,
  ApplyParams as MentorApplyParams,
  SubmitQQRequest,
  SubmitQQResponse,
  MentorCandidate,
} from '@/types/api/newbro'

type ApiResult<T> = ApiResponse<T>

export async function fetchDirectReferralStatus() {
  const response = await requestJson<ApiResult<DirectReferralStatus>>(
    '/api/v1/newbro/recruit/direct-referral'
  )
  return assertSuccess(response, 'fetch direct referral status failed')
}

export async function checkDirectReferrerQQ(data: CheckDirectReferrerParams) {
  const response = await requestJson<ApiResult<DirectReferrerCandidate>>(
    '/api/v1/newbro/recruit/direct-referral/check',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'check direct referrer failed')
}

export async function confirmDirectReferrer(data: ConfirmDirectReferrerParams) {
  const response = await requestJson<ApiResult<DirectReferrerCandidate>>(
    '/api/v1/newbro/recruit/direct-referral/confirm',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'confirm direct referrer failed')
}

export async function fetchAdminNewbroRecruitSettings() {
  const response = await requestJson<ApiResult<RecruitSettings>>(
    '/api/v1/system/newbro/recruit-settings'
  )
  return assertSuccess(response, 'fetch newbro recruit settings failed')
}

export async function updateAdminNewbroRecruitSettings(data: UpdateRecruitSettingsParams) {
  const response = await requestJson<ApiResult<RecruitSettings>>(
    '/api/v1/system/newbro/recruit-settings',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'update newbro recruit settings failed')
}

export async function generateRecruitLink() {
  const response = await requestJson<ApiResult<GenerateLinkResponse>>(
    '/api/v1/newbro/recruit/link',
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'generate recruit link failed')
}

export async function fetchMyRecruitLinks() {
  const response = await requestJson<ApiResult<RecruitLink[]>>('/api/v1/newbro/recruit/links')
  return assertSuccess(response, 'fetch my recruit links failed') ?? []
}

export async function fetchAdminRecruitLinks(params: { current?: number; size?: number }) {
  const searchParams = new URLSearchParams()
  if (params.current) searchParams.set('current', String(params.current))
  if (params.size) searchParams.set('size', String(params.size))

  const response = await requestJson<ApiResult<AdminRecruitLinksResponse>>(
    `/api/v1/system/newbro/recruit/links${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin recruit links failed')
}

export async function submitRecruitQQ(code: string, data: SubmitQQRequest) {
  const response = await requestJson<ApiResult<SubmitQQResponse>>(
    `/api/v1/recruit/${code}/submit`,
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'submit recruit qq failed')
}

export async function fetchMentorCandidates() {
  const response = await requestJson<ApiResult<MentorCandidate[]>>('/api/v1/mentor/mentors')
  return assertSuccess(response, 'fetch mentor candidates failed') ?? []
}

export async function fetchMyMentorStatus() {
  const response = await requestJson<ApiResult<MyStatusResponse>>('/api/v1/mentor/me')
  return assertSuccess(response, 'fetch mentor status failed')
}

export async function applyForMentor(data: MentorApplyParams) {
  const response = await requestJson<ApiResult<{ id: number }>>('/api/v1/mentor/apply', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'apply for mentor failed')
}

export async function fetchMentorApplications() {
  const response = await requestJson<ApiResult<RelationshipView[]>>(
    '/api/v1/mentor/dashboard/applications'
  )
  return assertSuccess(response, 'fetch mentor applications failed') ?? []
}

export async function acceptMentorApplication(data: RelationshipActionParams) {
  const response = await requestJson<ApiResult<RelationshipView>>(
    '/api/v1/mentor/dashboard/accept',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'accept mentor application failed')
}

export async function rejectMentorApplication(data: RelationshipActionParams) {
  const response = await requestJson<ApiResult<RelationshipView>>(
    '/api/v1/mentor/dashboard/reject',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'reject mentor application failed')
}

export async function fetchMentorMentees(params?: MentorMenteesParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.status) searchParams.set('status', params.status)

  const response = await requestJson<ApiResult<MenteeListResponse>>(
    `/api/v1/mentor/dashboard/mentees${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch mentor mentees failed')
}

export async function fetchMentorRewardStages() {
  const response = await requestJson<ApiResult<RewardStage[]>>(
    '/api/v1/mentor/dashboard/reward-stages'
  )
  return assertSuccess(response, 'fetch mentor reward stages failed') ?? []
}

export async function fetchAdminMentorRelationships(params?: AdminRelationshipsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.status) searchParams.set('status', params.status)
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<AdminRelationshipsResponse>>(
    `/api/v1/system/mentor/relationships${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin mentor relationships failed')
}

export async function fetchAdminMentorRewardDistributions(params?: AdminRewardDistributionsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<AdminRewardDistributionsResponse>>(
    `/api/v1/system/mentor/reward-distributions${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin mentor reward distributions failed')
}

export async function revokeMentorRelationship(data: RelationshipActionParams) {
  const response = await requestJson<ApiResult<EmptyResponse>>('/api/v1/system/mentor/revoke', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'revoke mentor relationship failed')
}

export async function updateMentorRewardStages(data: UpdateRewardStagesParams) {
  const response = await requestJson<ApiResult<RewardStage[]>>(
    '/api/v1/system/mentor/reward-stages',
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'update mentor reward stages failed') ?? []
}

export async function runMentorRewardProcessing() {
  const response = await requestJson<ApiResult<RewardProcessResult>>(
    '/api/v1/system/mentor/reward/process',
    { method: 'POST' }
  )
  return assertSuccess(response, 'run mentor reward processing failed')
}
