import request from '@/utils/http'
import type { Announcement, AnnouncementInput, AnnouncementList } from '@/types/announcement'
export const listAnnouncements = (page: number) =>
  request.get<AnnouncementList>({ url: '/api/v1/announcements', params: { page } })
export const getAnnouncement = (id: number) =>
  request.get<Announcement>({ url: '/api/v1/announcements/' + id })
export const saveAnnouncement = (id: number | null, data: AnnouncementInput) =>
  id
    ? request.put<Announcement>({ url: '/api/v1/announcements/' + id, data })
    : request.post<Announcement>({ url: '/api/v1/announcements', data })
export const readAnnouncement = (id: number, version: number) =>
  request.post<null>({ url: '/api/v1/announcements/' + id + '/read', data: { version } })
