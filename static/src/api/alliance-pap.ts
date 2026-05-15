import axios from 'axios'
import request from '@/utils/http'

const defaultSeatUserAgent =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36'

export interface AlliancePAPFleet {
  id: number
  main_character: string
  character_id: string
  character_name: string
  fleet_id: string
  year: number
  month: number
  start_at: string
  end_at?: string
  title: string
  level: string
  pap: number
  ship_group_id: string
  ship_group_name: string
  ship_type_id: string
  ship_type_name: string
  is_archived: boolean
}

export interface AlliancePAPSummary {
  id: number
  main_character: string
  year: number
  month: number
  corporation_id: string
  total_pap: number
  yearly_total_pap: number
  monthly_rank: number
  yearly_rank: number
  global_monthly_rank: number
  global_yearly_rank: number
  total_in_corp: number
  total_global: number
  calculated_at: string
  is_archived: boolean
}

export interface AlliancePAPResult {
  summary: AlliancePAPSummary | null
  fleets: AlliancePAPFleet[]
}

export interface AlliancePAPAllResult {
  year: number
  month: number
  list: AlliancePAPSummary[]
}

/** 获取我的联盟 PAP（当前用户主人物，默认当月） */
export function fetchMyAlliancePAP(params?: { year?: number; month?: number }) {
  return request.get<AlliancePAPResult>({
    url: '/api/v1/operation/fleets/pap/alliance',
    params
  })
}

/** 管理员：分页获取所有成员某月联盟 PAP 汇总 */
export function fetchAllAlliancePAP(
  params?: Api.Common.CommonSearchParams & { year?: number; month?: number }
) {
  return request.get<Api.Common.PaginatedResponse<AlliancePAPSummary>>({
    url: '/api/v1/system/pap',
    params
  })
}

/** 管理员：手动触发拉取 */
export function triggerAlliancePAPFetch(params?: { year?: number; month?: number }) {
  return request.post({
    url: '/api/v1/system/pap/fetch',
    params
  })
}

/** 管理员：通过表格导入 PAP 数据 */
export interface PAPImportInfo {
  primary_character_name: string
  monthly_pap: number
  calculated_at: string
}

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

export function importAlliancePAP(params?: { year?: number; month?: number; data: PAPImportInfo }) {
  return request.post<PAPImportInfo>({ url: '/api/v1/system/pap/import', params })
}

export interface SettleMonthResult {
  year: number
  month: number
}

/** 管理员：月度归档 */
export function settleAlliancePAPMonth(data: { year: number; month: number }) {
  return request.post<SettleMonthResult>({ url: '/api/v1/system/pap/settle', data })
}
