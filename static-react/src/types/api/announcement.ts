export interface Announcement {
  id: number
  title: string
  body_html?: string
  author_id: number
  author_nickname: string
  author_character_id: number
  author_character_name: string
  hidden: boolean
  version: number
  revision: number
  created_at: string
  updated_at: string
  unread?: boolean
}
export interface AnnouncementInput {
  title: string
  body_html: string
  hidden: boolean
  remind: boolean
  revision: number
}
export interface AnnouncementList {
  list: Announcement[]
  total: number
}
