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

export function fetchAdminNewbroSettings() {
  return request.get<Api.Newbro.Settings>({
    url: '/api/v1/system/newbro/settings'
  })
}

export function updateAdminNewbroSettings(data: Api.Newbro.UpdateSettingsParams) {
  return request.put<Api.Newbro.Settings>({
    url: '/api/v1/system/newbro/settings',
    data
  })
}

export function fetchRunCaptainAttributionSync() {
  return request.post<Api.Newbro.AttributionSyncResult>({
    url: '/api/v1/system/newbro/attribution/sync'
  })
}

export function fetchRunCaptainRewardProcessing() {
  return request.post<Api.Newbro.RewardProcessResult>({
    url: '/api/v1/system/newbro/reward/process'
  })
}
