import request from '@/utils/http'

export function fetchNewbroCaptains() {
  return request.get<Api.Newbro.CaptainCandidate[]>({
    url: '/api/v1/newbro/captains'
  })
}

export function fetchMyNewbroAffiliation() {
  return request.get<Api.Newbro.MyAffiliationResponse>({
    url: '/api/v1/newbro/affiliation/me'
  })
}

export function fetchMyAffiliationHistory(params: Api.Common.PaginationParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Newbro.AffiliationSummary>>({
    url: '/api/v1/newbro/affiliations/history',
    params
  })
}

export function fetchSelectCaptain(data: Api.Newbro.SelectCaptainParams) {
  return request.post<Api.Newbro.SelectCaptainResponse>({
    url: '/api/v1/newbro/affiliation/select',
    data
  })
}

export function fetchEndAffiliation() {
  return request.post<Api.Newbro.EndAffiliationResponse>({
    url: '/api/v1/newbro/affiliation/end'
  })
}

export function fetchCaptainEndAffiliation(data: Api.Newbro.CaptainEndAffiliationParams) {
  return request.post<Api.Newbro.EndAffiliationResponse>({
    url: '/api/v1/newbro/captain/affiliation/end',
    data
  })
}

export function fetchCaptainOverview() {
  return request.get<Api.Newbro.CaptainOverview>({
    url: '/api/v1/newbro/captain/overview'
  })
}

export function fetchCaptainPlayers(params?: Api.Newbro.CaptainPlayersParams) {
  return request.get<Api.Newbro.CaptainPlayersResponse>({
    url: '/api/v1/newbro/captain/players',
    params
  })
}

export function fetchCaptainAttributions(params?: Api.Newbro.CaptainAttributionsParams) {
  return request.get<Api.Newbro.CaptainAttributionsResponse>({
    url: '/api/v1/newbro/captain/attributions',
    params
  })
}

export function fetchCaptainRewardSettlements(params?: Api.Newbro.CaptainRewardSettlementsParams) {
  return request.get<Api.Newbro.CaptainRewardSettlementsResponse>({
    url: '/api/v1/newbro/captain/rewards',
    params
  })
}

export function fetchCaptainEligiblePlayers(params?: Api.Newbro.CaptainEligiblePlayersParams) {
  return request.get<Api.Newbro.CaptainEligiblePlayersResponse>({
    url: '/api/v1/newbro/captain/eligible-players',
    params
  })
}

export function fetchCaptainEnrollPlayer(data: Api.Newbro.CaptainEnrollPlayerParams) {
  return request.post<Api.Newbro.SelectCaptainResponse>({
    url: '/api/v1/newbro/captain/enroll',
    data
  })
}

export function fetchAdminCaptainList(params?: Api.Newbro.AdminCaptainsParams) {
  return request.get<Api.Newbro.AdminCaptainsResponse>({
    url: '/api/v1/system/newbro/captains',
    params
  })
}

export function fetchAdminCaptainDetail(userId: number) {
  return request.get<Api.Newbro.AdminCaptainDetail>({
    url: `/api/v1/system/newbro/captains/${userId}`
  })
}

export function fetchAdminAffiliationHistory(params?: Api.Newbro.AdminAffiliationHistoryParams) {
  return request.get<Api.Newbro.AdminAffiliationHistoryResponse>({
    url: '/api/v1/system/newbro/affiliations/history',
    params
  })
}

export function fetchAdminRewardSettlements(params?: Api.Newbro.AdminRewardSettlementsParams) {
  return request.get<Api.Newbro.AdminRewardSettlementsResponse>({
    url: '/api/v1/system/newbro/rewards',
    params
  })
}

export function fetchAdminNewbroSupportSettings() {
  return request.get<Api.Newbro.SupportSettings>({
    url: '/api/v1/system/newbro/support-settings'
  })
}

export function updateAdminNewbroSupportSettings(data: Api.Newbro.UpdateSupportSettingsParams) {
  return request.put<Api.Newbro.SupportSettings>({
    url: '/api/v1/system/newbro/support-settings',
    data
  })
}

export function fetchAdminNewbroRecruitSettings() {
  return request.get<Api.Newbro.RecruitSettings>({
    url: '/api/v1/system/newbro/recruit-settings'
  })
}

export function updateAdminNewbroRecruitSettings(data: Api.Newbro.UpdateRecruitSettingsParams) {
  return request.put<Api.Newbro.RecruitSettings>({
    url: '/api/v1/system/newbro/recruit-settings',
    data
  })
}

export function generateRecruitLink() {
  return request.post<Api.Newbro.GenerateLinkResponse>({
    url: '/api/v1/newbro/recruit/link',
    showErrorMessage: false
  })
}

export function fetchMyRecruitLinks() {
  return request.get<Api.Newbro.RecruitLink[]>({
    url: '/api/v1/newbro/recruit/links',
    showErrorMessage: false
  })
}

export function fetchAdminRecruitLinks(params: Api.Common.CommonSearchParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Newbro.AdminRecruitLink>>({
    url: '/api/v1/system/newbro/recruit/links',
    params,
    showErrorMessage: false
  })
}

export function submitRecruitQQ(code: string, data: Api.Newbro.SubmitQQRequest) {
  return request.post<Api.Newbro.SubmitQQResponse>({
    url: `/api/v1/recruit/${code}/submit`,
    data,
    showErrorMessage: false
  })
}

export function fetchDirectReferralStatus() {
  return request.get<Api.Newbro.DirectReferralStatus>({
    url: '/api/v1/newbro/recruit/direct-referral',
    showErrorMessage: false
  })
}

export function checkDirectReferrerQQ(data: Api.Newbro.CheckDirectReferrerParams) {
  return request.post<Api.Newbro.DirectReferrerCandidate>({
    url: '/api/v1/newbro/recruit/direct-referral/check',
    data,
    showErrorMessage: false
  })
}

export function confirmDirectReferrer(data: Api.Newbro.ConfirmDirectReferrerParams) {
  return request.post<Api.Newbro.DirectReferrerCandidate>({
    url: '/api/v1/newbro/recruit/direct-referral/confirm',
    data,
    showErrorMessage: false
  })
}
