export function humanizeNumber(num: number | null | undefined): string {
  if (num == null) {
    return '-'
  }

  const value = Number(num)
  const abs = Math.abs(value)

  if (abs >= 1e12) {
    return `${(value / 1e12).toFixed(2).replace(/\.00$/, '')}t`
  }

  if (abs >= 1e9) {
    return `${(value / 1e9).toFixed(2).replace(/\.00$/, '')}b`
  }

  if (abs >= 1e6) {
    return `${(value / 1e6).toFixed(2).replace(/\.00$/, '')}m`
  }

  if (abs >= 1e3) {
    return `${(value / 1e3).toFixed(2).replace(/\.00$/, '')}k`
  }

  return value.toString()
}

export function formatTime(value: string | null | undefined): string {
  if (!value) {
    return '-'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString('en-GB', { hour12: false })
}
