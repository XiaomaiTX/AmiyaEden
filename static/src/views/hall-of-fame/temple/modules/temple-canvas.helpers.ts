import amarrSvg from '@imgs/borders/amarr.svg'
import caldariSvg from '@imgs/borders/caldari.svg'
import gallenteSvg from '@imgs/borders/gallente.svg'
import gildedSvg from '@imgs/borders/gilded.svg'
import imperialSvg from '@imgs/borders/imperial.svg'
import minmatarSvg from '@imgs/borders/minmatar.svg'
import neonCircuitSvg from '@imgs/borders/neon-circuit.svg'
import voidRiftSvg from '@imgs/borders/void-rift.svg'

type HeroCardPreset =
  | 'gold'
  | 'silver'
  | 'darkred'
  | 'yellow'
  | 'bronze'
  | 'rose'
  | 'jade'
  | 'midnight'
  | 'custom'

type HeroCardPresetStyle = {
  background: string
  color: string
  borderColor: string
  boxShadowColor: string
  titleColor: string
}

type BorderStyleConfig = {
  src: string
}

type HeroCardStyleInput = {
  width: number
  height: number
  style_preset: HeroCardPreset
  custom_bg_color: string
  custom_text_color: string
  custom_border_color: string
  title_color: string
  border_style?: string
}

type HeroCardStyle = {
  width: string
  minHeight: string
  background: string
  color: string
  borderColor: string
  borderWidth: string
  boxShadowColor: string
  titleColor: string
  frameSrc?: string
}

export const HALL_OF_FAME_MIN_ZOOM = 40
export const HALL_OF_FAME_MAX_ZOOM = 160

const PRESET_STYLES: Record<Exclude<HeroCardPreset, 'custom'>, HeroCardPresetStyle> = {
  gold: {
    background: 'linear-gradient(180deg, #3a2f0b 0%, #1a1505 100%)',
    color: '#fff7d6',
    borderColor: '#ffd700',
    boxShadowColor: 'rgba(255, 215, 0, 0.35)',
    titleColor: '#ffe89a'
  },
  silver: {
    background: 'linear-gradient(180deg, #394556 0%, #161d29 100%)',
    color: '#edf3fa',
    borderColor: '#b8c5d6',
    boxShadowColor: 'rgba(184, 197, 214, 0.38)',
    titleColor: '#dce7f7'
  },
  darkred: {
    background: 'linear-gradient(180deg, #5b101b 0%, #22070d 100%)',
    color: '#fff0f2',
    borderColor: '#cf425f',
    boxShadowColor: 'rgba(207, 66, 95, 0.45)',
    titleColor: '#ffb4c1'
  },
  yellow: {
    background: 'linear-gradient(180deg, #4a3b05 0%, #1f1802 100%)',
    color: '#fff6cf',
    borderColor: '#f2c94c',
    boxShadowColor: 'rgba(242, 201, 76, 0.42)',
    titleColor: '#ffe490'
  },
  bronze: {
    background: 'linear-gradient(180deg, #2d1f0e 0%, #1a1208 100%)',
    color: '#f7e6d2',
    borderColor: '#cd7f32',
    boxShadowColor: 'rgba(205, 127, 50, 0.35)',
    titleColor: '#f6bd85'
  },
  rose: {
    background: 'linear-gradient(180deg, #4b2238 0%, #24111e 100%)',
    color: '#ffe7f2',
    borderColor: '#ff8fc7',
    boxShadowColor: 'rgba(255, 143, 199, 0.35)',
    titleColor: '#ffc7e3'
  },
  jade: {
    background: 'linear-gradient(180deg, #12332d 0%, #081a17 100%)',
    color: '#e8fff7',
    borderColor: '#53d7ad',
    boxShadowColor: 'rgba(83, 215, 173, 0.34)',
    titleColor: '#aef1d9'
  },
  midnight: {
    background: 'linear-gradient(180deg, #182645 0%, #0a1123 100%)',
    color: '#edf3ff',
    borderColor: '#7ea8ff',
    boxShadowColor: 'rgba(126, 168, 255, 0.34)',
    titleColor: '#c4d7ff'
  }
}

const BORDER_STYLE_ASSETS: Record<string, BorderStyleConfig> = {
  gilded: { src: gildedSvg },
  imperial: { src: imperialSvg },
  'neon-circuit': { src: neonCircuitSvg },
  'void-rift': { src: voidRiftSvg },
  amarr: { src: amarrSvg },
  caldari: { src: caldariSvg },
  minmatar: { src: minmatarSvg },
  gallente: { src: gallenteSvg }
}

export function clampHallOfFameZoom(value: number) {
  if (!Number.isFinite(value)) {
    return 100
  }

  return Math.max(HALL_OF_FAME_MIN_ZOOM, Math.min(HALL_OF_FAME_MAX_ZOOM, Math.round(value)))
}

export function buildHeroCardStyle(input: HeroCardStyleInput): HeroCardStyle {
  const width = input.width > 0 ? input.width : 196
  const minHeight = input.height > 0 ? input.height : Math.max(148, Math.round(width * 0.64))
  const borderAsset =
    input.border_style && input.border_style !== 'none'
      ? BORDER_STYLE_ASSETS[input.border_style]
      : undefined
  const frameSrc = borderAsset?.src
  const borderWidth = '2px'

  if (input.style_preset === 'custom') {
    const borderColor = input.custom_border_color || '#4ad295'
    const titleColor = input.title_color || borderColor

    const base = {
      width: `${width}px`,
      minHeight: `${minHeight}px`,
      background: input.custom_bg_color || '#101820',
      color: input.custom_text_color || '#f7f7f7',
      borderColor,
      borderWidth,
      boxShadowColor: hexToRgba(borderColor, 0.35),
      titleColor
    }

    return frameSrc ? { ...base, frameSrc } : base
  }

  const preset = PRESET_STYLES[input.style_preset]

  const base = {
    width: `${width}px`,
    minHeight: `${minHeight}px`,
    background: preset.background,
    color: preset.color,
    borderColor: preset.borderColor,
    borderWidth,
    boxShadowColor: preset.boxShadowColor,
    titleColor: input.title_color || preset.titleColor
  }

  return frameSrc ? { ...base, frameSrc } : base
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
