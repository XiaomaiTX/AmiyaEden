import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type { ToolBookmark, ToolBookmarkUpsertRequest } from '@/types/api/tool-bookmark'

export async function fetchVisibleToolBookmarks() {
  const response = await requestJson<ApiResponse<ToolBookmark[]>>('/api/v1/info/tool-bookmarks', {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch visible tool bookmarks failed')
}

export async function fetchAdminToolBookmarks() {
  const response = await requestJson<ApiResponse<ToolBookmark[]>>('/api/v1/system/tool-bookmarks', {
    method: 'GET',
  })
  return assertSuccess(response, 'fetch admin tool bookmarks failed')
}

export async function createToolBookmark(data: ToolBookmarkUpsertRequest) {
  const response = await requestJson<ApiResponse<ToolBookmark>>('/api/v1/system/tool-bookmarks', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create tool bookmark failed')
}

export async function updateToolBookmark(id: number, data: ToolBookmarkUpsertRequest) {
  const response = await requestJson<ApiResponse<ToolBookmark>>(
    `/api/v1/system/tool-bookmarks/${id}`,
    {
      method: 'PUT',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'update tool bookmark failed')
}

export async function deleteToolBookmark(id: number) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/system/tool-bookmarks/${id}`, {
    method: 'DELETE',
  })
  return assertSuccess(response, 'delete tool bookmark failed')
}
