export type FuxiHallPageKey = 'leadership' | 'contributors'
export type FuxiHallAvatarShape = 'circle' | 'rounded' | 'square'

export interface FuxiHallPageConfig {
  id: number
  page_key: FuxiHallPageKey
  title: string
  subtitle: string
  description_html: string
}

export interface FuxiHallCard {
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
  fleet_led_count?: number
  welfare_delivery_count?: number
  welfare_delivery_offset?: number
  created_at: string
  updated_at: string
}

export interface FuxiHallPublicPageResponse {
  page: FuxiHallPageConfig
  cards: FuxiHallCard[]
}

export interface FuxiHallPageConfigUpdate {
  title?: string
  subtitle?: string
  description_html?: string
}

export interface FuxiHallCardCreate {
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
  welfare_delivery_offset?: number
}

export interface FuxiHallCardReorder {
  page_key: FuxiHallPageKey
  ordered_ids: number[]
}
