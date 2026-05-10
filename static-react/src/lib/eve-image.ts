const EVE_IMAGE_BASE_URL = 'https://images.evetech.net'
const EVE_PORTRAIT_SIZES = [32, 64, 128, 256, 512] as const

function normalizeEvePortraitSize(size: number) {
  for (const supportedSize of EVE_PORTRAIT_SIZES) {
    if (size <= supportedSize) {
      return supportedSize
    }
  }

  return EVE_PORTRAIT_SIZES[EVE_PORTRAIT_SIZES.length - 1]
}

export function buildEveCharacterPortraitUrl(characterId: number, size = 128) {
  return characterId > 0
    ? `${EVE_IMAGE_BASE_URL}/characters/${characterId}/portrait?size=${normalizeEvePortraitSize(size)}`
    : ''
}
