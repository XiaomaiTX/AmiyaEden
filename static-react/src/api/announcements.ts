import { requestJson } from './http-client'
import type { Announcement, AnnouncementInput, AnnouncementList } from '@/types/api/announcement'
async function call<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const res = await requestJson<{ code: number; msg: string; data: T }>(
    '/api/v1/announcements' + path,
    { method, body: body === undefined ? undefined : JSON.stringify(body) }
  )
  if (res.code !== 0 && res.code !== 200) throw new Error(res.msg)
  return res.data
}
export const listAnnouncements = (page: number) => call<AnnouncementList>('?page=' + page)
export const getAnnouncement = (id: number) => call<Announcement>('/' + id)
export const saveAnnouncement = (id: number | null, body: AnnouncementInput) =>
  call<Announcement>(id ? '/' + id : '', id ? 'PUT' : 'POST', body)
export const readAnnouncement = (id: number, version: number) =>
  call<null>('/' + id + '/read', 'POST', { version })
