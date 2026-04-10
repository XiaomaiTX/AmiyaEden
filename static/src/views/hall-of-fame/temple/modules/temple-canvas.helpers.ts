type HeroCardPreset = 'gold' | 'silver' | 'bronze' | 'custom'

type HeroCardPresetStyle = {
  background: string
  color: string
  borderColor: string
  boxShadowColor: string
}

type HeroCardStyleInput = {
  width: number
  height: number
  style_preset: HeroCardPreset
  custom_bg_color: string
  custom_text_color: string
  custom_border_color: string
}

const PRESET_STYLES: Record<Exclude<HeroCardPreset, 'custom'>, HeroCardPresetStyle> = {
  gold: {
    background: 'linear-gradient(180deg, #3a2f0b 0%, #1a1505 100%)',
    color: '#fff7d6',
    borderColor: '#ffd700',
    boxShadowColor: 'rgba(255, 215, 0, 0.35)'
  },
  silver: {
    background: 'linear-gradient(180deg, #1a1a2e 0%, #0d0d1a 100%)',
    color: '#f4f7fb',
    borderColor: '#c0c0c0',
    boxShadowColor: 'rgba(192, 192, 192, 0.35)'
  },
  bronze: {
    background: 'linear-gradient(180deg, #2d1f0e 0%, #1a1208 100%)',
    color: '#f7e6d2',
    borderColor: '#cd7f32',
    boxShadowColor: 'rgba(205, 127, 50, 0.35)'
  }
}

export function getTempleScale(containerWidth: number, canvasWidth: number, canvasHeight: number) {
  if (canvasWidth <= 0 || canvasHeight <= 0) {
    return {
      ratio: 1,
      wrapperHeight: Math.max(canvasHeight, 0)
    }
  }

  const ratio = containerWidth > 0 ? Math.min(1, containerWidth / canvasWidth) : 1

  return {
    ratio,
    wrapperHeight: canvasHeight * ratio
  }
}

export function buildHeroCardStyle(input: HeroCardStyleInput) {
  const width = input.width > 0 ? input.width : 220
  const minHeight = input.height > 0 ? input.height : width

  if (input.style_preset === 'custom') {
    const borderColor = input.custom_border_color || '#4ad295'

    return {
      width: `${width}px`,
      minHeight: `${minHeight}px`,
      background: input.custom_bg_color || '#101820',
      color: input.custom_text_color || '#f7f7f7',
      borderColor,
      boxShadowColor: hexToRgba(borderColor, 0.35)
    }
  }

  const preset = PRESET_STYLES[input.style_preset]

  return {
    width: `${width}px`,
    minHeight: `${minHeight}px`,
    background: preset.background,
    color: preset.color,
    borderColor: preset.borderColor,
    boxShadowColor: preset.boxShadowColor
  }
}

function hexToRgba(hex: string, alpha: number) {
  const normalized = hex.replace('#', '')
  const fullHex =
    normalized.length === 3
      ? normalized
          .split('')
          .map((part) => `${part}${part}`)
          .join('')
      : normalized

  if (!/^[0-9a-fA-F]{6}$/.test(fullHex)) {
    return `rgba(74, 210, 149, ${alpha})`
  }

  const red = parseInt(fullHex.slice(0, 2), 16)
  const green = parseInt(fullHex.slice(2, 4), 16)
  const blue = parseInt(fullHex.slice(4, 6), 16)

  return `rgba(${red}, ${green}, ${blue}, ${alpha})`
}
