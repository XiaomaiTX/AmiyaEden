import request from '@/utils/http'

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
