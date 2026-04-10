export interface CardUpdateQueueState {
  active: Api.HallOfFame.UpdateCardParams | null
  queued: Api.HallOfFame.UpdateCardParams | null
}

const missingCardPattern = /卡片\s+(\d+)\s+不存在/

export function getMissingCardIdFromError(error: unknown): number | null {
  const message = error instanceof Error ? error.message : ''
  const match = missingCardPattern.exec(message)
  if (!match) {
    return null
  }

  const cardId = Number(match[1])
  return Number.isFinite(cardId) ? cardId : null
}

export function queueCardUpdateRequest(
  state: CardUpdateQueueState,
  patch: Api.HallOfFame.UpdateCardParams
): { state: CardUpdateQueueState; patchToSend: Api.HallOfFame.UpdateCardParams | null } {
  if (state.active) {
    return {
      state: {
        active: state.active,
        queued: {
          ...(state.queued ?? {}),
          ...patch
        }
      },
      patchToSend: null
    }
  }

  return {
    state: {
      active: patch,
      queued: null
    },
    patchToSend: patch
  }
}

export function settleCardUpdateRequest(state: CardUpdateQueueState): {
  state: CardUpdateQueueState
  patchToSend: Api.HallOfFame.UpdateCardParams | null
} {
  const nextPatch = state.queued ?? null

  return {
    state: {
      active: nextPatch,
      queued: null
    },
    patchToSend: nextPatch
  }
}

export function rebuildCardFromConfirmedState(
  confirmedCard: Api.HallOfFame.Card,
  visibleCard: Api.HallOfFame.Card,
  pendingPatch: Api.HallOfFame.UpdateCardParams | null
): Api.HallOfFame.Card {
  const nextCard: Api.HallOfFame.Card = {
    ...confirmedCard,
    pos_x: visibleCard.pos_x,
    pos_y: visibleCard.pos_y,
    width: visibleCard.width,
    height: visibleCard.height,
    z_index: visibleCard.z_index
  }

  return pendingPatch
    ? {
        ...nextCard,
        ...pendingPatch
      }
    : nextCard
}
