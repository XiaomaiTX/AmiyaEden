import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AdminAffiliationHistoryParams,
  AdminAffiliationHistoryResponse,
  AdminCaptainDetail,
  AdminCaptainsParams,
  AdminCaptainsResponse,
  AdminRecruitLinksResponse,
  AdminRewardDistributionsParams,
  AdminRewardDistributionsResponse,
  ApplyResponse,
  CheckDirectReferrerParams,
  ConfirmDirectReferrerParams,
  DirectReferralStatus,
  DirectReferrerCandidate,
  EmptyResponse,
  EndAffiliationResponse,
  GenerateLinkResponse,
  CaptainAttributionsParams,
  CaptainAttributionsResponse,
  CaptainEligiblePlayersParams,
  CaptainEligiblePlayersResponse,
  CaptainEnrollPlayerParams,
  CaptainEndAffiliationParams,
  CaptainOverview,
  CaptainPlayersParams,
  CaptainPlayersResponse,
  CaptainRewardSettlementsParams,
  CaptainRewardSettlementsResponse,
  CaptainCandidate,
  MenteeListResponse,
  MyAffiliationResponse,
  MyStatusResponse,
  AdminRelationshipsParams,
  AdminRelationshipsResponse,
  RecruitLink,
  RelationshipActionParams,
  RewardProcessResult,
  RewardStage,
  UpdateMentorSettingsParams,
  UpdateRecruitSettingsParams,
  UpdateRewardStagesParams,
  UpdateSupportSettingsParams,
  SupportSettings,
  RecruitSettings,
  RelationshipView,
  MentorSettings,
  MentorMenteesParams,
  ApplyParams as MentorApplyParams,
  MyAffiliationHistoryResponse,
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

export async function fetchNewbroCaptains() {
  const response = await requestJson<ApiResult<CaptainCandidate[]>>('/api/v1/newbro/captains')
  return assertSuccess(response, 'fetch newbro captains failed') ?? []
}

export async function fetchMyNewbroAffiliation() {
  const response = await requestJson<ApiResult<MyAffiliationResponse>>('/api/v1/newbro/affiliation/me')
  return assertSuccess(response, 'fetch my newbro affiliation failed')
}

export async function fetchMyAffiliationHistory(params: { current?: number; size?: number }) {
  const searchParams = new URLSearchParams()
  if (params.current) searchParams.set('current', String(params.current))
  if (params.size) searchParams.set('size', String(params.size))

  const response = await requestJson<ApiResult<MyAffiliationHistoryResponse>>(
    `/api/v1/newbro/affiliations/history?${searchParams.toString()}`
  )
  return assertSuccess(response, 'fetch newbro affiliation history failed')
}

export async function fetchSelectCaptain(data: { captain_user_id: number }) {
  const response = await requestJson<ApiResult<ApplyResponse>>('/api/v1/newbro/affiliation/select', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'select captain failed')
}

export async function fetchEndAffiliation() {
  const response = await requestJson<ApiResult<EndAffiliationResponse>>(
    '/api/v1/newbro/affiliation/end',
    { method: 'POST' }
  )
  return assertSuccess(response, 'end affiliation failed')
}

export async function fetchCaptainEndAffiliation(data: CaptainEndAffiliationParams) {
  const response = await requestJson<ApiResult<EndAffiliationResponse>>(
    '/api/v1/newbro/captain/affiliation/end',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'end captain affiliation failed')
}

export async function fetchCaptainOverview() {
  const response = await requestJson<ApiResult<CaptainOverview>>('/api/v1/newbro/captain/overview')
  return assertSuccess(response, 'fetch captain overview failed')
}

export async function fetchCaptainPlayers(params?: CaptainPlayersParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.status) searchParams.set('status', params.status)

  const response = await requestJson<ApiResult<CaptainPlayersResponse>>(
    `/api/v1/newbro/captain/players${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch captain players failed')
}

export async function fetchCaptainAttributions(params?: CaptainAttributionsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.player_user_id) searchParams.set('player_user_id', String(params.player_user_id))
  if (params?.ref_type) searchParams.set('ref_type', params.ref_type)
  if (params?.start_date) searchParams.set('start_date', params.start_date)
  if (params?.end_date) searchParams.set('end_date', params.end_date)

  const response = await requestJson<ApiResult<CaptainAttributionsResponse>>(
    `/api/v1/newbro/captain/attributions${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch captain attributions failed')
}

export async function fetchCaptainRewardSettlements(params?: CaptainRewardSettlementsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<CaptainRewardSettlementsResponse>>(
    `/api/v1/newbro/captain/rewards${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch captain reward settlements failed')
}

export async function fetchCaptainEligiblePlayers(params?: CaptainEligiblePlayersParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<CaptainEligiblePlayersResponse>>(
    `/api/v1/newbro/captain/eligible-players${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch captain eligible players failed')
}

export async function fetchCaptainEnrollPlayer(data: CaptainEnrollPlayerParams) {
  const response = await requestJson<ApiResult<ApplyResponse>>('/api/v1/newbro/captain/enroll', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'captain enroll player failed')
}

export async function fetchAdminCaptainList(params?: AdminCaptainsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<AdminCaptainsResponse>>(
    `/api/v1/system/newbro/captains${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin captain list failed')
}

export async function fetchAdminCaptainDetail(userId: number) {
  const response = await requestJson<ApiResult<AdminCaptainDetail>>(
    `/api/v1/system/newbro/captains/${userId}`
  )
  return assertSuccess(response, 'fetch admin captain detail failed')
}

export async function fetchAdminAffiliationHistory(params?: AdminAffiliationHistoryParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.captain_search) searchParams.set('captain_search', params.captain_search)
  if (params?.player_search) searchParams.set('player_search', params.player_search)
  if (params?.change_start_date) searchParams.set('change_start_date', params.change_start_date)
  if (params?.change_end_date) searchParams.set('change_end_date', params.change_end_date)

  const response = await requestJson<ApiResult<AdminAffiliationHistoryResponse>>(
    `/api/v1/system/newbro/affiliations/history${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin affiliation history failed')
}

export async function fetchAdminRewardSettlements(params?: AdminRewardDistributionsParams) {
  const searchParams = new URLSearchParams()
  if (params?.current) searchParams.set('current', String(params.current))
  if (params?.size) searchParams.set('size', String(params.size))
  if (params?.keyword) searchParams.set('keyword', params.keyword)

  const response = await requestJson<ApiResult<AdminRewardDistributionsResponse>>(
    `/api/v1/system/newbro/rewards${searchParams.toString() ? `?${searchParams.toString()}` : ''}`
  )
  return assertSuccess(response, 'fetch admin reward settlements failed')
}

export async function fetchAdminNewbroSupportSettings() {
  const response = await requestJson<ApiResult<SupportSettings>>('/api/v1/system/newbro/support-settings')
  return assertSuccess(response, 'fetch newbro support settings failed')
}

export async function updateAdminNewbroSupportSettings(data: UpdateSupportSettingsParams) {
  const response = await requestJson<ApiResult<SupportSettings>>('/api/v1/system/newbro/support-settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update newbro support settings failed')
}

export async function fetchAdminNewbroRecruitSettings() {
  const response = await requestJson<ApiResult<RecruitSettings>>('/api/v1/system/newbro/recruit-settings')
  return assertSuccess(response, 'fetch newbro recruit settings failed')
}

export async function updateAdminNewbroRecruitSettings(data: UpdateRecruitSettingsParams) {
  const response = await requestJson<ApiResult<RecruitSettings>>('/api/v1/system/newbro/recruit-settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update newbro recruit settings failed')
}

export async function generateRecruitLink() {
  const response = await requestJson<ApiResult<GenerateLinkResponse>>('/api/v1/newbro/recruit/link', {
    method: 'POST',
  })
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
  const response = await requestJson<ApiResult<SubmitQQResponse>>(`/api/v1/recruit/${code}/submit`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
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

export async function fetchMentorSettings() {
  const response = await requestJson<ApiResult<MentorSettings>>('/api/v1/system/mentor/settings')
  return assertSuccess(response, 'fetch mentor settings failed')
}

export async function updateMentorSettings(data: UpdateMentorSettingsParams) {
  const response = await requestJson<ApiResult<MentorSettings>>('/api/v1/system/mentor/settings', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update mentor settings failed')
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
  const response = await requestJson<ApiResult<RewardStage[]>>('/api/v1/system/mentor/reward-stages', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update mentor reward stages failed') ?? []
}

export async function runMentorRewardProcessing() {
  const response = await requestJson<ApiResult<RewardProcessResult>>(
    '/api/v1/system/mentor/reward/process',
    { method: 'POST' }
  )
  return assertSuccess(response, 'run mentor reward processing failed')
}
