import request from '@/utils/http'

export function fetchFuxiHallLeadership() {
  return request.get<Api.FuxiHall.PublicPageResponse>({
    url: '/api/v1/fuxi-hall/leadership'
  })
}

export function fetchFuxiHallContributors() {
  return request.get<Api.FuxiHall.PublicPageResponse>({
    url: '/api/v1/fuxi-hall/contributors'
  })
}

export function fetchFuxiHallPage(pageKey: Api.FuxiHall.PageKey) {
  return request.get<Api.FuxiHall.PageConfig>({
    url: `/api/v1/system/fuxi-hall/pages/${pageKey}`
  })
}

export function updateFuxiHallPage(
  pageKey: Api.FuxiHall.PageKey,
  data: Api.FuxiHall.UpdatePageConfigParams
) {
  return request.put<Api.FuxiHall.PageConfig>({
    url: `/api/v1/system/fuxi-hall/pages/${pageKey}`,
    data
  })
}

export function fetchFuxiHallCards(pageKey: Api.FuxiHall.PageKey) {
  return request.get<Api.FuxiHall.Card[]>({
    url: `/api/v1/system/fuxi-hall/cards/${pageKey}`
  })
}

export function createFuxiHallCard(data: Api.FuxiHall.CreateCardParams) {
  return request.post<Api.FuxiHall.Card>({
    url: '/api/v1/system/fuxi-hall/cards',
    data
  })
}

export function updateFuxiHallCard(id: number, data: Api.FuxiHall.UpdateCardParams) {
  return request.put<Api.FuxiHall.Card>({
    url: `/api/v1/system/fuxi-hall/cards/${id}`,
    data
  })
}

export function reorderFuxiHallCards(data: Api.FuxiHall.ReorderParams) {
  return request.put({
    url: '/api/v1/system/fuxi-hall/cards/reorder',
    data
  })
}

export function deleteFuxiHallCard(id: number) {
  return request.del({
    url: `/api/v1/system/fuxi-hall/cards/${id}`
  })
}
