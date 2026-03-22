import request from '@/utils/http'

// ─── 管理员福利设置 ───

/** 管理员查询福利列表 */
export function adminListWelfares(data?: Api.Welfare.SearchParams) {
  return request.post<Api.Common.PaginatedResponse<Api.Welfare.WelfareItem>>({
    url: '/api/v1/system/welfare/list',
    data: data ?? { current: 1, size: 20 }
  })
}

/** 管理员创建福利 */
export function adminCreateWelfare(data: Api.Welfare.CreateParams) {
  return request.post<Api.Welfare.WelfareItem>({
    url: '/api/v1/system/welfare/add',
    data
  })
}

/** 管理员更新福利 */
export function adminUpdateWelfare(data: Api.Welfare.UpdateParams) {
  return request.post<Api.Welfare.WelfareItem>({
    url: '/api/v1/system/welfare/edit',
    data
  })
}

/** 管理员删除福利 */
export function adminDeleteWelfare(id: number) {
  return request.post({
    url: '/api/v1/system/welfare/delete',
    data: { id }
  })
}
