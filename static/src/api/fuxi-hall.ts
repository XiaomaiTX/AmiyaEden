import request from '@/utils/http'

export function fetchFuxiHall() {
  return request.get<Api.FuxiHall.PublicPageResponse>({
    showErrorMessage: false,
    url: '/api/v1/fuxi-hall/hall-of-fame'
  })
}
export function updateOwnFuxiHallCard(
  id: number,
  data: { nickname: string; description_html: string }
) {
  return request.put<Api.FuxiHall.Card>({
    showErrorMessage: false,
    url: `/api/v1/fuxi-hall/cards/${id}/profile`,
    data
  })
}
export function fetchFuxiHallPage(pageKey: Api.FuxiHall.PageKey) {
  return request.get<Api.FuxiHall.PageConfig>({
    showErrorMessage: false,
    url: `/api/v1/system/fuxi-hall/pages/${pageKey}`
  })
}

export function updateFuxiHallPage(
  pageKey: Api.FuxiHall.PageKey,
  data: Api.FuxiHall.UpdatePageConfigParams
) {
  return request.put<Api.FuxiHall.PageConfig>({
    showErrorMessage: false,
    url: `/api/v1/system/fuxi-hall/pages/${pageKey}`,
    data
  })
}

export function fetchFuxiHallCards(pageKey: Api.FuxiHall.PageKey) {
  return request.get<Api.FuxiHall.Card[]>({
    showErrorMessage: false,
    url: `/api/v1/system/fuxi-hall/cards/${pageKey}`
  })
}

export function createFuxiHallCard(data: Api.FuxiHall.CreateCardParams) {
  return request.post<Api.FuxiHall.Card>({
    showErrorMessage: false,
    url: '/api/v1/system/fuxi-hall/cards',
    data
  })
}

export function updateFuxiHallCard(id: number, data: Api.FuxiHall.UpdateCardParams) {
  return request.put<Api.FuxiHall.Card>({
    showErrorMessage: false,
    url: `/api/v1/system/fuxi-hall/cards/${id}`,
    data
  })
}

export function reorderFuxiHallCards(data: Api.FuxiHall.ReorderParams) {
  return request.put({
    showErrorMessage: false,
    url: '/api/v1/system/fuxi-hall/cards/reorder',
    data
  })
}

export function deleteFuxiHallCard(id: number) {
  return request.del({
    showErrorMessage: false,
    url: `/api/v1/system/fuxi-hall/cards/${id}`
  })
}

export function refreshFuxiHallContributions() {
  return request.post<void>({
    showErrorMessage: false,
    url: '/api/v1/system/fuxi-hall/contributions/refresh'
  })
}
