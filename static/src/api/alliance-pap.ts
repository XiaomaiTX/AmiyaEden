import axios from 'axios'

const defaultSeatUserAgent =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36'

export interface SeatPapTrackingCredentials {
  laravelSession: string
  cfClearance?: string
  userAgent?: string
}

export interface SeatPapTrackingItem {
  character: string
  pap_count: number | string | null
  logoff_date: string | null
}

export interface SeatPapTrackingResponse {
  data?: SeatPapTrackingItem[]
}

export function fetchSeatPapTracking(credentials: SeatPapTrackingCredentials) {
  const { VITE_API_URL } = import.meta.env
  const cookie =
    `laravel_session=${credentials.laravelSession}` +
    (credentials.cfClearance === '' || credentials.cfClearance == null
      ? ''
      : `;cf_clearance=${credentials.cfClearance}`)

  const axiosInstance = axios.create({
    timeout: 10000,
    baseURL: VITE_API_URL,
    headers: {
      Accept: 'application/json, text/javascript, */*; q=0.01',
      'X-Accept-Encoding': 'gzip, deflate, br, zstd',
      'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6',
      'Cache-Control': 'no-cache',
      // Contains user-provided SEAT session credentials; proxy/request logging must redact it.
      'X-Cookie': cookie,
      Pragma: 'no-cache',
      Priority: 'u=1, i',
      'X-Sec-Ch-Ua': `"Not:A-Brand";v="99", "Microsoft Edge";v="145", "Chromium";v="145"`,
      'X-Sec-Ch-Ua-Mobile': '?0',
      'X-Sec-Ch-Ua-Platform': `"Windows"`,
      'X-Sec-Fetch-Dest': 'empty',
      'X-Sec-Fetch-Mode': 'cors',
      'X-Sec-Fetch-Site': 'same-origin',
      'X-User-Agent':
        credentials.userAgent === '' || credentials.userAgent == null
          ? defaultSeatUserAgent
          : credentials.userAgent,
      'X-Requested-With': 'XMLHttpRequest'
    }
  })

  return axiosInstance.get<SeatPapTrackingResponse>('/seatproxy/tools/paptracking')
}
