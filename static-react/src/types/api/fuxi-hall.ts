import type { HallCard, HallContributions, HallPublic } from '@/api/generated/client'

export type FuxiHallContributions = Omit<HallContributions, 'corp_pap'> &
  Partial<Pick<HallContributions, 'corp_pap'>>

type PresentationFields =
  | 'qq'
  | 'can_edit'
  | 'contributions_updated_at'
  | 'tenure_months'
  | 'avatar_image'

export type FuxiHallCard = Omit<HallCard, PresentationFields | 'contributions'> &
  Partial<Pick<HallCard, PresentationFields>> & {
    contributions?: FuxiHallContributions
    contribution_offsets?: FuxiHallContributions
  }

export type FuxiHallPublicPageResponse = Omit<HallPublic, 'cards'> & { cards: FuxiHallCard[] }
