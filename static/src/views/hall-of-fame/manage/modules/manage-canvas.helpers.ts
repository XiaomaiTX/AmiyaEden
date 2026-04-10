type LayoutCard = Pick<
  Api.HallOfFame.Card,
  'id' | 'pos_x' | 'pos_y' | 'width' | 'height' | 'z_index'
>

type CardPatch = Partial<Api.HallOfFame.Card>

export function buildNewCardPayload(
  name: string,
  maxZIndex: number
): Api.HallOfFame.CreateCardParams {
  return {
    name,
    pos_x: 50,
    pos_y: 50,
    width: 220,
    style_preset: 'gold',
    z_index: maxZIndex + 1,
    visible: true
  }
}

export function clampCardCoordinate(value: number) {
  return Math.max(0, Math.min(100, value))
}

export function toLayoutUpdates(cards: LayoutCard[]): Api.HallOfFame.CardLayoutUpdate[] {
  return cards.map((card) => ({
    id: card.id,
    pos_x: card.pos_x,
    pos_y: card.pos_y,
    width: card.width,
    height: card.height,
    z_index: card.z_index
  }))
}

export function patchCardById(
  cards: Api.HallOfFame.Card[],
  id: number,
  patch: CardPatch
): Api.HallOfFame.Card[] {
  return cards.map((card) =>
    card.id === id
      ? {
          ...card,
          ...patch
        }
      : card
  )
}
