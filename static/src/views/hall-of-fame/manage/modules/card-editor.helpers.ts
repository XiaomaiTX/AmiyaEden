export function mergePendingCardUpdates(
  current: Api.HallOfFame.UpdateCardParams,
  next: Api.HallOfFame.UpdateCardParams
): Api.HallOfFame.UpdateCardParams {
  return {
    ...current,
    ...next
  }
}
