export interface FuxiHallContributions {
  corp_pap?: number
  fleets: number
  welfare: number
  srp: number
  recruitment: number
  mentoring: number
}
export interface FuxiHallFloor {
  key: string
  name: string
  capacity: number
}
export type FuxiHallPageKey = 'hall-of-fame'
export type FuxiHallAvatarShape = 'circle' | 'rounded' | 'square'

export interface FuxiHallPageConfig {
  id: number
  page_key: FuxiHallPageKey
  title: string
  subtitle: string
  description_html: string
  floors: FuxiHallFloor[]
}

export interface FuxiHallCard {
  qq?: string
  contributions?: FuxiHallContributions
  contribution_offsets?: FuxiHallContributions
  contributions_updated_at?: string | null
  floor_key: string
  is_current_management: boolean
  can_edit?: boolean
  tenure_months?: number | null
  id: number
  page_key: FuxiHallPageKey
  nickname: string
  main_character_id: number
  main_character_name: string
  title_tags: string[]
  description_html: string
  accent_color: string
  avatar_shape: FuxiHallAvatarShape
  font_scale: number
  visible: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface FuxiHallPublicPageResponse {
  page: FuxiHallPageConfig
  cards: FuxiHallCard[]
}

export interface FuxiHallPageConfigUpdate {
  floors?: FuxiHallFloor[]
  title?: string
  subtitle?: string
  description_html?: string
}

export interface FuxiHallCardCreate {
  contribution_offsets?: FuxiHallContributions
  floor_key?: string
  is_current_management?: boolean
  page_key: FuxiHallPageKey
  nickname: string
  main_character_name: string
  title_tags: string[]
  description_html?: string
  accent_color?: string
  avatar_shape?: FuxiHallAvatarShape
  font_scale?: number
  visible?: boolean
}

export type FuxiHallCardUpdate = Partial<FuxiHallCardCreate> & {
  title_tags?: string[]
}

export interface FuxiHallCardReorder {
  page_key: FuxiHallPageKey
  ordered_ids: number[]
}
