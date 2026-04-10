export function buildHallOfFamePortraitUrl(characterId: number) {
  return characterId > 0 ? `https://images.evetech.net/characters/${characterId}/portrait` : ''
}
