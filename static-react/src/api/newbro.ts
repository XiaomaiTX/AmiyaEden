import { requestJson } from '@/api/http-client'

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 200 && response.code !== 0) {
    throw new Error(response.msg || fallbackMessage)
  }
  return response.data
}

export async function fetchDirectReferralStatus() {
  const response = await requestJson<ApiResponse<Api.Newbro.DirectReferralStatus>>(
    '/api/v1/newbro/recruit/direct-referral'
  )

  return assertSuccess(response, 'fetch direct referral status failed')
}

export async function checkDirectReferrerQQ(data: Api.Newbro.CheckDirectReferrerParams) {
  const response = await requestJson<ApiResponse<Api.Newbro.DirectReferrerCandidate>>(
    '/api/v1/newbro/recruit/direct-referral/check',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'check direct referrer failed')
}

export async function confirmDirectReferrer(data: Api.Newbro.ConfirmDirectReferrerParams) {
  const response = await requestJson<ApiResponse<Api.Newbro.DirectReferrerCandidate>>(
    '/api/v1/newbro/recruit/direct-referral/confirm',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )

  return assertSuccess(response, 'confirm direct referrer failed')
}
