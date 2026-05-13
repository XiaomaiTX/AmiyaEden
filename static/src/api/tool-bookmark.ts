import request from '@/utils/http'

export function fetchVisibleToolBookmarks() {
  return request.get<Api.ToolBookmark.Bookmark[]>({
    url: '/api/v1/info/tool-bookmarks'
  })
}

export function fetchAdminToolBookmarks() {
  return request.get<Api.ToolBookmark.Bookmark[]>({
    url: '/api/v1/system/tool-bookmarks'
  })
}

export function createToolBookmark(data: Api.ToolBookmark.UpsertParams) {
  return request.post<Api.ToolBookmark.Bookmark>({
    url: '/api/v1/system/tool-bookmarks',
    data
  })
}

export function updateToolBookmark(id: number, data: Api.ToolBookmark.UpsertParams) {
  return request.put<Api.ToolBookmark.Bookmark>({
    url: `/api/v1/system/tool-bookmarks/${id}`,
    data
  })
}

export function deleteToolBookmark(id: number) {
  return request.del({
    url: `/api/v1/system/tool-bookmarks/${id}`
  })
}
